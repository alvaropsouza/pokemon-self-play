package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/alvaropsouza/pokemon-self-play/internal/bot"
	"github.com/alvaropsouza/pokemon-self-play/internal/cards"
	"github.com/alvaropsouza/pokemon-self-play/internal/deck"
	"github.com/alvaropsouza/pokemon-self-play/internal/game"
	"github.com/alvaropsouza/pokemon-self-play/web"
)

const (
	human = 0
	botP  = 1
)

type server struct {
	mu    sync.Mutex
	store *cards.Store
	g     *game.Game
}

func main() {
	addr := flag.String("addr", "localhost:8080", "endereço HTTP")
	dataPath := flag.String("data", "data/cards.json", "base de cartas")
	flag.Parse()

	store, err := cards.Load(*dataPath)
	if err != nil {
		log.Fatal(err)
	}
	if err := game.LoadEffectDB("data/effects.json"); err != nil {
		log.Fatal(err)
	}
	s := &server{store: store}

	dist, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		log.Fatal(err)
	}
	http.Handle("/", http.FileServer(http.FS(dist)))
	http.HandleFunc("/api/state", safeHandler(s.handleState))
	http.HandleFunc("/api/action", safeHandler(s.handleAction))
	http.HandleFunc("/api/new", safeHandler(s.handleNew))
	http.HandleFunc("/api/decks", safeHandler(s.handleDecks))

	log.Printf("em http://%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func (s *server) handleDecks(w http.ResponseWriter, r *http.Request) {
	var out []map[string]any
	for _, info := range bot.BattleDecks() {
		d, err := bot.BattleDeck(s.store, info.ID)
		if err != nil {
			continue
		}
		counts := map[string]int{}
		type row struct {
			c *cards.Card
			n int
		}
		var rows []row
		for id, n := range d.Counts {
			c := s.store.Cards[id]
			counts[string(c.Category)] += n
			rows = append(rows, row{c, n})
		}
		catOrder := map[cards.Category]int{cards.CategoryPokemon: 0, cards.CategoryTrainer: 1, cards.CategoryEnergy: 2}
		sort.Slice(rows, func(i, j int) bool {
			if a, b := catOrder[rows[i].c.Category], catOrder[rows[j].c.Category]; a != b {
				return a < b
			}
			return rows[i].c.ID < rows[j].c.ID
		})
		var list []map[string]any
		for _, r := range rows {
			list = append(list, map[string]any{"card": cardJSON(r.c), "count": r.n})
		}
		out = append(out, map[string]any{
			"id": info.ID, "type": info.Type, "name": info.Name,
			"star": cardJSON(s.store.Cards[info.Star]),
			"counts": counts, "cards": list,
		})
	}
	writeJSON(w, out)
}

func (s *server) handleNew(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		MyType  string `json:"mytype"`
		BotType string `json:"bottype"`
		Seed    int64  `json:"seed"`
	}
	req.MyType = "Fire"
	req.BotType = "Water"
	json.NewDecoder(r.Body).Decode(&req)
	if req.Seed <= 0 {
		req.Seed = time.Now().UnixNano()
	}

	myDeck, err := buildDeck(s.store, req.MyType, req.Seed)
	if err != nil {
		log.Printf("[new] erro deck jogador (%s): %v", req.MyType, err)
		writeJSON(w, map[string]any{"phase": "lobby", "error": err.Error()})
		return
	}
	botDeck, err := buildDeck(s.store, req.BotType, req.Seed+1)
	if err != nil {
		log.Printf("[new] erro deck bot (%s): %v", req.BotType, err)
		writeJSON(w, map[string]any{"phase": "lobby", "error": err.Error()})
		return
	}
	g, err := game.New(s.store, [2][]string{myDeck.CardIDs(), botDeck.CardIDs()}, req.Seed, -1)
	if err != nil {
		log.Printf("[new] erro game.New: %v", err)
		writeJSON(w, map[string]any{"phase": "lobby", "error": err.Error()})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.g = g
	if err := bot.Setup(g, botP); err != nil {
		s.g = nil
		log.Printf("[new] erro setup bot: %v", err)
		writeJSON(w, map[string]any{"phase": "lobby", "error": err.Error()})
		return
	}
	log.Printf("[new] partida: você %s (%d cartas) | bot %s (%d cartas) | seed %d",
		req.MyType, myDeck.Size(), req.BotType, botDeck.Size(), req.Seed)
	writeJSON(w, s.stateJSON())
}

func buildDeck(store *cards.Store, key string, seed int64) (*deck.Deck, error) {
	if d, err := bot.BattleDeck(store, key); err == nil {
		return d, nil
	}
	typ := strings.ToLower(key)
	if typ != "" {
		typ = strings.ToUpper(typ[:1]) + typ[1:]
	}
	if d, err := bot.BattleDeck(store, typ); err == nil {
		return d, nil
	}
	return bot.BuildDeck(store, []string{typ}, seed)
}

func (s *server) advance() {
	for s.g.Phase == game.PhaseTurn {
		bot.PromoteIfNeeded(s.g, botP)
		if s.g.NeedPromote[human] {
			return
		}
		if pc := s.g.Pending; pc != nil {
			if pc.Player == human {
				return
			}
			bot.ResolvePending(s.g, botP)
		}
		if s.g.Current != botP {
			return
		}
		bot.TakeTurn(s.g, botP)
	}
}

type actionReq struct {
	Action    string `json:"action"`
	Hand      int    `json:"hand"`
	Slot      int    `json:"slot"`
	Bench     int    `json:"bench"`
	Attack    int    `json:"attack"`
	Energies  []int  `json:"energies"`
	Player    int    `json:"player"`
	Amount    int    `json:"amount"`
	Condition string `json:"condition"`
	Picks     []int  `json:"picks"`
}

func parseCommand(defaultPlayer int, req actionReq) (game.Command, error) {
	p := defaultPlayer
	switch req.Action {
	case "place_active":   return game.PlaceActiveCmd{Player: p, HandIdx: req.Hand}, nil
	case "place_bench":    return game.PlaceBenchCmd{Player: p, HandIdx: req.Hand}, nil
	case "finish_setup":   return game.FinishSetupCmd{Player: p}, nil
	case "attach_energy":  return game.AttachEnergyCmd{Player: p, HandIdx: req.Hand, Slot: req.Slot}, nil
	case "evolve":         return game.EvolveCmd{Player: p, HandIdx: req.Hand, Slot: req.Slot}, nil
	case "attach_tool":    return game.AttachToolCmd{Player: p, HandIdx: req.Hand, Slot: req.Slot}, nil
	case "play_item":      return game.PlayItemCmd{Player: p, HandIdx: req.Hand}, nil
	case "play_supporter": return game.PlaySupporterCmd{Player: p, HandIdx: req.Hand}, nil
	case "play_stadium":   return game.PlayStadiumCmd{Player: p, HandIdx: req.Hand}, nil
	case "retreat":        return game.RetreatCmd{Player: p, BenchIdx: req.Bench, Energies: req.Energies}, nil
	case "attack":         return game.AttackCmd{Player: p, AtkIdx: req.Attack}, nil
	case "promote":        return game.PromoteCmd{Player: p, BenchIdx: req.Bench}, nil
	case "end_turn":       return game.EndTurnCmd{Player: p}, nil
	case "resolve_choice": return game.ResolveChoiceCmd{Player: p, Picks: req.Picks}, nil
	case "arb_damage":     return game.ArbDamageCmd{Player: req.Player, Slot: req.Slot, Amount: req.Amount}, nil
	case "arb_heal":       return game.ArbHealCmd{Player: req.Player, Slot: req.Slot, Amount: req.Amount}, nil
	case "arb_condition":  return game.ArbConditionCmd{Player: req.Player, Condition: req.Condition}, nil
	case "arb_draw":       return game.ArbDrawCmd{Player: req.Player, Amount: req.Amount}, nil
	case "arb_switch":     return game.ArbSwitchCmd{Player: req.Player, BenchIdx: req.Bench}, nil
	case "arb_shuffle":    return game.ArbShuffleCmd{Player: req.Player}, nil
	default:               return nil, fmt.Errorf("ação desconhecida: %q", req.Action)
	}
}

func (s *server) handleAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST", http.StatusMethodNotAllowed)
		return
	}
	var req actionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.g == nil {
		log.Printf("[action] %s: sem partida ativa", req.Action)
		writeJSON(w, map[string]any{"phase": "lobby", "error": "sem partida ativa"})
		return
	}

	cmd, err := parseCommand(human, req)
	if err == nil {
		log.Printf("[action] %s", req.Action)
		err = cmd.Execute(s.g)
	}
	s.advance()

	resp := s.stateJSON()
	if err != nil {
		log.Printf("[action] erro %s: %v", req.Action, err)
		resp["error"] = err.Error()
	}
	writeJSON(w, resp)
}

func (s *server) handleState(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.g == nil {
		writeJSON(w, map[string]any{"phase": "lobby"})
		return
	}
	writeJSON(w, s.stateJSON())
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("[writeJSON] encode error: %v", err)
	}
}

func safeHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[panic] %s %s: %v", r.Method, r.URL.Path, rec)
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]any{"error": fmt.Sprintf("panic: %v", rec)})
			}
		}()
		fn(w, r)
	}
}

func (s *server) cardView(id string) map[string]any {
	return cardJSON(s.g.Card(id))
}

func cardJSON(c *cards.Card) map[string]any {
	if c == nil {
		return nil
	}
	name := c.Name.PT
	if name == "" {
		name = c.Name.EN
	}
	img := c.Image.PT
	if img == "" {
		img = c.Image.EN
	}
	if img != "" {
		img += "/low.webp"
	}
	v := map[string]any{
		"id": c.ID, "name": name, "nameEN": c.Name.EN, "image": img,
		"category": c.Category, "stage": c.Stage, "trainerType": c.TrainerType,
		"hp": c.HP, "retreat": c.Retreat,
	}
	var atks []map[string]any
	for _, a := range c.Attacks {
		an := a.Name.PT
		if an == "" {
			an = a.Name.EN
		}
		atks = append(atks, map[string]any{"name": an, "cost": a.Cost, "damage": a.Damage})
	}
	v["attacks"] = atks
	return v
}

func (s *server) pokemonView(p *game.PokemonInPlay) map[string]any {
	if p == nil {
		return nil
	}
	var energies []map[string]any
	for _, id := range p.Energies {
		energies = append(energies, s.cardView(id))
	}
	conds := []string{}
	if p.Rot != game.CondNone {
		conds = append(conds, string(p.Rot))
	}
	if p.Poisoned {
		conds = append(conds, "poisoned")
	}
	if p.Burned {
		conds = append(conds, "burned")
	}
	v := map[string]any{
		"card": s.cardView(p.TopID()), "damage": p.Damage,
		"energies": energies, "conditions": conds,
	}
	if p.Tool != "" {
		v["tool"] = s.cardView(p.Tool)
	}
	return v
}

func (s *server) sideView(p int, full bool) map[string]any {
	ps := s.g.Players[p]
	var bench []map[string]any
	for _, b := range ps.Bench {
		bench = append(bench, s.pokemonView(b))
	}
	var discard []map[string]any
	for _, id := range ps.Discard {
		discard = append(discard, s.cardView(id))
	}
	v := map[string]any{
		"deck": len(ps.Deck), "prizes": len(ps.Prizes), "prizesTaken": ps.PrizesTaken,
		"active": s.pokemonView(ps.Active), "bench": bench, "discard": discard,
		"handCount": len(ps.Hand),
	}
	if full {
		var hand []map[string]any
		for _, id := range ps.Hand {
			hand = append(hand, s.cardView(id))
		}
		v["hand"] = hand
	}
	return v
}

func (s *server) stateJSON() map[string]any {
	g := s.g
	logTail := g.Log
	if len(logTail) > 40 {
		logTail = logTail[len(logTail)-40:]
	}
	v := map[string]any{
		"phase": g.Phase, "turn": g.TurnNumber, "current": g.Current,
		"winner": g.Winner, "needPromote": g.NeedPromote, "log": logTail,
		"you": s.sideView(human, true), "bot": s.sideView(botP, false),
	}
	if g.Stadium != "" {
		v["stadium"] = s.cardView(g.Stadium)
	}
	if pc := g.Pending; pc != nil && pc.Player == human {
		var cand []map[string]any
		switch pc.Kind {
		case game.ChoiceSwitchSelf:
			for _, benchIdx := range pc.Candidates {
				cand = append(cand, s.pokemonView(g.Players[human].Bench[benchIdx]))
			}
		case game.ChoiceSwitchOpp:
			for _, benchIdx := range pc.Candidates {
				cand = append(cand, s.pokemonView(g.Players[botP].Bench[benchIdx]))
			}
		case game.ChoiceDiscardHand:
			for _, hi := range pc.Candidates {
				cand = append(cand, s.cardView(g.Players[human].Hand[hi]))
			}
		default:
			for _, di := range pc.Candidates {
				cand = append(cand, s.cardView(g.Players[human].Deck[di]))
			}
		}
		v["pendingChoice"] = map[string]any{
			"kind": pc.Kind, "max": pc.Max, "min": pc.Min, "dest": pc.Dest, "candidates": cand,
		}
	}
	return v
}
