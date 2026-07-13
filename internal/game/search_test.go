package game

import (
	"testing"

	"github.com/alvaropsouza/pokemon-self-play/internal/cards"
)

// newSearchGame: partida com um Item de busca (estilo Fighting Gong, tipo Fire)
// na mão do jogador 0.
func newSearchGame(t *testing.T) *Game {
	t.Helper()
	s := testStore()
	s.Put(&cards.Card{
		ID: "t-gong", Name: cards.Localized{EN: "Fire Gong"}, Category: cards.CategoryTrainer,
		TrainerType: "Item", RegulationMark: "H",
		Effect: cards.Localized{EN: "Search your deck for a Basic {R} Energy card or a Basic {R} Pokémon, reveal it, and put it into your hand. Then, shuffle your deck."},
	})
	d := deck60(map[string]int{"t-fire1": 20, "t-fireE": 20, "t-gong": 20})
	d2 := deck60(map[string]int{"t-water1": 30, "t-waterE": 30})
	g, err := New(s, [2][]string{d, d2}, 42, 0)
	if err != nil {
		t.Fatal(err)
	}
	for p := 0; p < 2; p++ {
		if err := g.PlaceActive(p, findInHand(t, g, p, cards.CategoryPokemon)); err != nil {
			t.Fatal(err)
		}
		if err := g.FinishSetup(p); err != nil {
			t.Fatal(err)
		}
	}
	return g
}

func handIdxOf(g *Game, p int, id string) int {
	for i, h := range g.Players[p].Hand {
		if h == id {
			return i
		}
	}
	return -1
}

func TestSearchItemPendingAndResolve(t *testing.T) {
	g := newSearchGame(t)
	idx := handIdxOf(g, 0, "t-gong")
	if idx < 0 {
		t.Skip("t-gong não veio na mão inicial com esta seed")
	}
	handBefore := len(g.Players[0].Hand)
	deckBefore := len(g.Players[0].Deck)

	if err := g.PlayItem(0, idx); err != nil {
		t.Fatal(err)
	}
	pc := g.Pending
	if pc == nil || pc.Player != 0 || pc.Dest != "hand" || pc.Max != 1 {
		t.Fatalf("escolha pendente errada: %+v", pc)
	}
	if len(pc.Candidates) == 0 {
		t.Fatal("sem candidatos — deck tem Fire básico e Fire Energy")
	}
	// Candidatos são só Fire básico / Fire Energy (nunca o próprio Item).
	for _, di := range pc.Candidates {
		id := g.Players[0].Deck[di]
		if id != "t-fire1" && id != "t-fireE" {
			t.Errorf("candidato inesperado: %s", id)
		}
	}
	// Ações bloqueadas enquanto pendente.
	if err := g.EndTurn(0); err == nil {
		t.Error("EndTurn deveria falhar com escolha pendente")
	}

	if err := g.ResolveChoice(0, []int{0}); err != nil {
		t.Fatal(err)
	}
	if g.Pending != nil {
		t.Error("pendência deveria ter sido limpa")
	}
	// Item saiu da mão (-1), busca trouxe 1 (+1) → mesmo tamanho.
	if got := len(g.Players[0].Hand); got != handBefore {
		t.Errorf("mão: esperado %d, veio %d", handBefore, got)
	}
	if got := len(g.Players[0].Deck); got != deckBefore-1 {
		t.Errorf("deck: esperado %d, veio %d", deckBefore-1, got)
	}
}

func TestSearchResolveEmptyPicks(t *testing.T) {
	g := newSearchGame(t)
	idx := handIdxOf(g, 0, "t-gong")
	if idx < 0 {
		t.Skip("t-gong não veio na mão inicial com esta seed")
	}
	if err := g.PlayItem(0, idx); err != nil {
		t.Fatal(err)
	}
	deckBefore := len(g.Players[0].Deck)
	if err := g.ResolveChoice(0, nil); err != nil {
		t.Fatal(err)
	}
	if g.Pending != nil || len(g.Players[0].Deck) != deckBefore {
		t.Error("busca vazia: pendência limpa e deck intacto esperados")
	}
	// Turno segue normal.
	if err := g.EndTurn(0); err != nil {
		t.Errorf("EndTurn após resolução: %v", err)
	}
}

func TestSearchInvalidPicks(t *testing.T) {
	g := newSearchGame(t)
	idx := handIdxOf(g, 0, "t-gong")
	if idx < 0 {
		t.Skip("t-gong não veio na mão inicial com esta seed")
	}
	if err := g.PlayItem(0, idx); err != nil {
		t.Fatal(err)
	}
	if err := g.ResolveChoice(0, []int{0, 1}); err == nil {
		t.Error("acima do máximo deveria falhar")
	}
	if err := g.ResolveChoice(0, []int{999}); err == nil {
		t.Error("índice inválido deveria falhar")
	}
	if err := g.ResolveChoice(1, []int{0}); err == nil {
		t.Error("jogador errado deveria falhar")
	}
	if g.Pending == nil {
		t.Error("pendência deve sobreviver a resoluções inválidas")
	}
}
