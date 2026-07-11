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
	"strings"
	"sync"

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
	g     *game.Game
	pilot *bot.Pilot
}

func main() {
	addr := flag.String("addr", "localhost:8080", "endereço HTTP")
	dataPath := flag.String("data", "data/cards.json", "base de cartas")
	myType := flag.String("mytype", "Fire", "tipo do seu deck")
	botType := flag.String("bottype", "Water", "tipo do deck do bot")
	seed := flag.Int64("seed", 1, "seed da partida")
	flag.Parse()

	store, err := cards.Load(*dataPath)
	if err != nil {
		log.Fatal(err)
	}
	myDeck, err := buildDeck(store, *myType, *seed)
	if err != nil {
		log.Fatalf("deck do jogador: %v", err)
	}
	botDeck, err := buildDeck(store, *botType, *seed+1)
	if err != nil {
		log.Fatalf("deck do bot: %v", err)
	}
	g, err := game.New(store, [2][]string{myDeck.CardIDs(), botDeck.CardIDs()}, *seed, -1)
	if err != nil {
		log.Fatal(err)
	}
	s := &server{g: g, pilot: &bot.Pilot{Player: botP}}
	if err := s.pilot.Setup(g); err != nil {
		log.Fatalf("setup do bot: %v", err)
	}

	dist, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		log.Fatal(err)
	}
	http.Handle("/", http.FileServer(http.FS(dist)))
	http.HandleFunc("/api/state", s.handleState)
	http.HandleFunc("/api/action", s.handleAction)

	log.Printf("partida em http://%s — você: %s | bot: %s | seed %d", *addr, *myType, *botType, *seed)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func buildDeck(store *cards.Store, typ string, seed int64) (*deck.Deck, error) {
	return bot.BuildDeck(store, []string{strings.Title(strings.ToLower(typ))}, seed)
}

// advance faz o bot agir sempre que for a vez dele (promoção e turno completo).
func (s *server) advance() {
	for s.g.Phase == game.PhaseTurn {
		s.pilot.PromoteIfNeeded(s.g)
		if s.g.NeedPromote[human] {
			return // aguarda o humano promover
		}
		if s.g.Current != botP {
			return
		}
		s.pilot.TakeTurn(s.g)
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

	g := s.g
	var err error
	switch req.Action {
	case "place_active":
		err = g.PlaceActive(human, req.Hand)
	case "place_bench":
		err = g.PlaceBench(human, req.Hand)
	case "finish_setup":
		err = g.FinishSetup(human)
	case "attach_energy":
		err = g.AttachEnergy(human, req.Hand, req.Slot)
	case "evolve":
		err = g.Evolve(human, req.Hand, req.Slot)
	case "attach_tool":
		err = g.AttachTool(human, req.Hand, req.Slot)
	case "play_item":
		err = g.PlayItem(human, req.Hand)
	case "play_supporter":
		err = g.PlaySupporter(human, req.Hand)
	case "play_stadium":
		err = g.PlayStadium(human, req.Hand)
	case "retreat":
		err = g.Retreat(human, req.Bench, req.Energies)
	case "attack":
		err = g.Attack(human, req.Attack)
	case "promote":
		err = g.Promote(human, req.Bench)
	case "end_turn":
		err = g.EndTurn(human)
	// Arbitragem manual de efeitos de carta.
	case "arb_damage":
		err = g.ApplyDamage(req.Player, req.Slot, req.Amount)
	case "arb_heal":
		err = g.Heal(req.Player, req.Slot, req.Amount)
	case "arb_condition":
		err = g.SetCondition(req.Player, req.Condition)
	case "arb_draw":
		g.DrawCards(req.Player, req.Amount)
	case "arb_switch":
		err = g.SwitchActive(req.Player, req.Bench)
	case "arb_shuffle":
		g.ShuffleDeck(req.Player)
	default:
		err = fmt.Errorf("ação desconhecida: %q", req.Action)
	}
	s.advance()

	resp := s.stateJSON()
	if err != nil {
		resp["error"] = err.Error()
	}
	writeJSON(w, resp)
}

func (s *server) handleState(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	writeJSON(w, s.stateJSON())
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}

// ---- visão do estado (esconde mão/deck/prêmios do bot) ----

func (s *server) cardView(id string) map[string]any {
	c := s.g.Card(id)
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
	return v
}
