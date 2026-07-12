package bot

import (
	"testing"

	"github.com/alvaropsouza/pokemon-self-play/internal/cards"
	"github.com/alvaropsouza/pokemon-self-play/internal/game"
)

// loadStore carrega a base real; pula o teste se data/cards.json não existe.
func loadStore(t *testing.T) *cards.Store {
	t.Helper()
	s, err := cards.Load("../../data/cards.json")
	if err != nil || len(s.Cards) == 0 {
		t.Skip("data/cards.json ausente — rode go run ./cmd/import me01 sve")
	}
	return s
}

func TestBuildDeckValidAndDeterministic(t *testing.T) {
	s := loadStore(t)
	for _, typ := range []string{"Fire", "Water", "Grass", "Psychic"} {
		d1, err := BuildDeck(s, []string{typ}, 7)
		if err != nil {
			t.Fatalf("%s: %v", typ, err)
		}
		if d1.Size() != 60 {
			t.Errorf("%s: %d cartas", typ, d1.Size())
		}
		d2, _ := BuildDeck(s, []string{typ}, 7)
		if len(d1.CardIDs()) != len(d2.CardIDs()) {
			t.Fatalf("%s: tamanhos diferentes com mesma seed", typ)
		}
		for i, id := range d1.CardIDs() {
			if d2.CardIDs()[i] != id {
				t.Fatalf("%s: mesma seed gerou decks diferentes", typ)
			}
		}
	}
}

// TestBotVsBotFullGame joga partidas completas bot contra bot: não pode travar
// nem violar o motor; alguém tem que vencer.
func TestBotVsBotFullGame(t *testing.T) {
	s := loadStore(t)
	for seed := int64(1); seed <= 5; seed++ {
		d1, err := BuildDeck(s, []string{"Fire"}, seed)
		if err != nil {
			t.Fatal(err)
		}
		d2, err := BuildDeck(s, []string{"Water"}, seed+100)
		if err != nil {
			t.Fatal(err)
		}
		g, err := game.New(s, [2][]string{d1.CardIDs(), d2.CardIDs()}, seed, -1)
		if err != nil {
			t.Fatal(err)
		}
		if err := Setup(g, 0); err != nil {
			t.Fatal(err)
		}
		if err := Setup(g, 1); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 500 && g.Phase == game.PhaseTurn; i++ {
			PromoteIfNeeded(g, 0)
			PromoteIfNeeded(g, 1)
			if g.Phase != game.PhaseTurn {
				break
			}
			TakeTurn(g, g.Current)
		}
		if g.Phase != game.PhaseFinished {
			t.Fatalf("seed %d: partida não terminou em 500 iterações\nlog: %v", seed, g.Log[len(g.Log)-10:])
		}
		t.Logf("seed %d: vencedor jogador %d em %d turnos", seed, g.Winner+1, g.TurnNumber)
	}
}
