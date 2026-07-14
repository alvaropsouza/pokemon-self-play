// cmd/play sobe a interface web da partida contra o bot (PLANO.md etapa 2).
// Os dois decks são gerados do pool por tipo; o humano é o jogador 1 (índice 0)
// e joga pelo navegador; o bot joga automaticamente no turno dele.
//
//	go run ./cmd/play -mytype Fire -bottype Water -seed 7
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

// handleDecks lista os Battle Decks para a tela de seleção: capa (carta-estrela),
// contagens por categoria e a decklist completa com imagem de cada carta.
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
		// Pokémon primeiro, depois Treinadores, Energias por último; ID desempata.
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
	// Sem seed no request: sorteia (cada partida diferente). Seed explícita
	// (>0) mantém reprodutibilidade para teste/depuração.
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

// buildDeck resolve a escolha do lobby: ID de Battle Deck ("grass-venusaur"),
// tipo ("Fire" → primeiro Battle Deck do tipo) ou fallback heurístico do pool.
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

// advance faz o bot agir sempre que for a vez dele (promoção e turno completo).
func (s *server) advance() {
	for s.g.Phase == game.PhaseTurn {
		bot.PromoteIfNeeded(s.g, botP)
		if s.g.NeedPromote[human] {
			return // aguarda o humano promover
		}
		if pc := s.g.Pending; pc != nil {
			if pc.Player == human {
				return // aguarda o humano escolher
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
	Action   string `json:"action"`
	Hand     int    `json:"hand"`
	Slot     int    `json:"slot"`
	Bench    int    `json:"bench"`
	Attack   int    `json:"attack"`
	Energies []int  `json:"energies"`
	// Campos de arbitragem manual.
	Player    int    `json:"player"`
	Amount    int    `json:"amount"`
	Condition string `json:"condition"`
	// Picks: posições escolhidas na lista de candidatos da busca pendente.
	Picks []int `json:"picks"`
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

	g := s.g
	var err error
	switch req.Action {
	case "place_active":
		log.Printf("[action] place_active hand=%d", req.Hand)
		err = g.PlaceActive(human, req.Hand)
	case "place_bench":
		log.Printf("[action] place_bench hand=%d", req.Hand)
		err = g.PlaceBench(human, req.Hand)
	case "finish_setup":
		log.Printf("[action] finish_setup")
		err = g.FinishSetup(human)
	case "attach_energy":
		log.Printf("[action] attach_energy hand=%d slot=%d", req.Hand, req.Slot)
		err = g.AttachEnergy(human, req.Hand, req.Slot)
	case "evolve":
		log.Printf("[action] evolve hand=%d slot=%d", req.Hand, req.Slot)
		err = g.Evolve(human, req.Hand, req.Slot)
	case "attach_tool":
		log.Printf("[action] attach_tool hand=%d slot=%d", req.Hand, req.Slot)
		err = g.AttachTool(human, req.Hand, req.Slot)
	case "play_item":
		log.Printf("[action] play_item hand=%d", req.Hand)
		err = g.PlayItem(human, req.Hand)
	case "play_supporter":
		log.Printf("[action] play_supporter hand=%d", req.Hand)
		err = g.PlaySupporter(human, req.Hand)
	case "play_stadium":
		log.Printf("[action] play_stadium hand=%d", req.Hand)
		err = g.PlayStadium(human, req.Hand)
	case "retreat":
		log.Printf("[action] retreat bench=%d energies=%v", req.Bench, req.Energies)
		err = g.Retreat(human, req.Bench, req.Energies)
	case "attack":
		log.Printf("[action] attack idx=%d", req.Attack)
		err = g.Attack(human, req.Attack)
	case "promote":
		log.Printf("[action] promote bench=%d", req.Bench)
		err = g.Promote(human, req.Bench)
	case "resolve_choice":
		log.Printf("[action] resolve_choice picks=%v", req.Picks)
		err = g.ResolveChoice(human, req.Picks)
	case "end_turn":
		log.Printf("[action] end_turn turno=%d", g.TurnNumber)
		err = g.EndTurn(human)
	case "arb_damage":
		log.Printf("[arb] damage player=%d slot=%d amount=%d", req.Player, req.Slot, req.Amount)
		err = g.ApplyDamage(req.Player, req.Slot, req.Amount)
	case "arb_heal":
		log.Printf("[arb] heal player=%d slot=%d amount=%d", req.Player, req.Slot, req.Amount)
		err = g.Heal(req.Player, req.Slot, req.Amount)
	case "arb_condition":
		log.Printf("[arb] condition player=%d condition=%q", req.Player, req.Condition)
		err = g.SetCondition(req.Player, req.Condition)
	case "arb_draw":
		log.Printf("[arb] draw player=%d amount=%d", req.Player, req.Amount)
		g.DrawCards(req.Player, req.Amount)
	case "arb_switch":
		log.Printf("[arb] switch player=%d bench=%d", req.Player, req.Bench)
		err = g.SwitchActive(req.Player, req.Bench)
	case "arb_shuffle":
		log.Printf("[arb] shuffle player=%d", req.Player)
		g.ShuffleDeck(req.Player)
	default:
		err = fmt.Errorf("ação desconhecida: %q", req.Action)
		log.Printf("[action] %v", err)
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

// safeHandler envolve um handler com recover: panics viram respostas JSON de erro.
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

// ---- visão do estado (esconde mão/deck/prêmios do bot) ----

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
		default: // ChoiceSearch
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
