package game

import (
	"strings"
	"testing"

	"github.com/alvaropsouza/pokemon-self-play/internal/cards"
)

// testStore monta uma base mínima de cartas sintéticas para os testes.
func testStore() *cards.Store {
	s := cards.NewStore()
	put := func(c cards.Card) { s.Put(&c) }

	put(cards.Card{
		ID: "t-fire1", Name: cards.Localized{EN: "Firebug"}, Category: cards.CategoryPokemon,
		Stage: "Basic", HP: 60, Types: []string{"Fire"}, Retreat: 1, RegulationMark: "H",
		Attacks:    []cards.Attack{{Name: cards.Localized{EN: "Ember"}, Cost: []string{"Fire"}, Damage: "30"}},
		Weaknesses: []cards.TypeValue{{Type: "Water", Value: "×2"}},
	})
	put(cards.Card{
		ID: "t-fire2", Name: cards.Localized{EN: "Firebug II"}, Category: cards.CategoryPokemon,
		Stage: "Stage1", EvolveFrom: cards.Localized{EN: "Firebug"}, HP: 120,
		Types: []string{"Fire"}, Retreat: 2, RegulationMark: "H",
		Attacks: []cards.Attack{{Name: cards.Localized{EN: "Flame Burst"}, Cost: []string{"Fire", "Colorless"}, Damage: "80"}},
	})
	put(cards.Card{
		ID: "t-water1", Name: cards.Localized{EN: "Aquaduck"}, Category: cards.CategoryPokemon,
		Stage: "Basic", HP: 180, Types: []string{"Water"}, Retreat: 2, RegulationMark: "H",
		Attacks: []cards.Attack{{Name: cards.Localized{EN: "Splash"}, Cost: []string{"Water"}, Damage: "30"}},
	})
	put(cards.Card{
		ID: "t-fireE", Name: cards.Localized{EN: "Fire Energy"}, Category: cards.CategoryEnergy,
		EnergyType: "Basic",
	})
	put(cards.Card{
		ID: "t-waterE", Name: cards.Localized{EN: "Water Energy"}, Category: cards.CategoryEnergy,
		EnergyType: "Basic",
	})
	put(cards.Card{
		ID: "t-sup", Name: cards.Localized{EN: "Test Supporter"}, Category: cards.CategoryTrainer,
		TrainerType: "Supporter", RegulationMark: "H",
	})
	put(cards.Card{
		ID: "t-item", Name: cards.Localized{EN: "Test Item"}, Category: cards.CategoryTrainer,
		TrainerType: "Item", RegulationMark: "H",
	})
	return s
}

// deck60 monta um deck de 60 cartas repetindo os IDs na proporção dada.
func deck60(ids map[string]int) []string {
	var out []string
	for id, n := range ids {
		for i := 0; i < n; i++ {
			out = append(out, id)
		}
	}
	return out
}

// newTestGame cria uma partida pronta: setup feito, jogador 0 começa.
func newTestGame(t *testing.T) *Game {
	t.Helper()
	d := deck60(map[string]int{"t-fire1": 30, "t-fireE": 30})
	d2 := deck60(map[string]int{"t-water1": 30, "t-waterE": 30})
	g, err := New(testStore(), [2][]string{d, d2}, 42, 0)
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

// findInHand devolve o índice da primeira carta da categoria na mão.
func findInHand(t *testing.T, g *Game, p int, cat cards.Category) int {
	t.Helper()
	for i, id := range g.Players[p].Hand {
		if g.Card(id).Category == cat {
			return i
		}
	}
	t.Fatalf("jogador %d sem carta %s na mão", p+1, cat)
	return -1
}

func TestSetupDealsPrizesAndFirstDraw(t *testing.T) {
	g := newTestGame(t)
	if g.Phase != PhaseTurn || g.TurnNumber != 1 || g.Current != 0 {
		t.Fatalf("estado inicial errado: fase=%s turno=%d jogador=%d", g.Phase, g.TurnNumber, g.Current)
	}
	for p := 0; p < 2; p++ {
		if len(g.Players[p].Prizes) != 6 {
			t.Errorf("jogador %d: %d prêmios (esperado 6)", p+1, len(g.Players[p].Prizes))
		}
	}
	// Jogador 0 começou: 7 da mão inicial −1 Ativo +1 compra = 7.
	if len(g.Players[0].Hand) != 7 {
		t.Errorf("mão do jogador 0: %d (esperado 7)", len(g.Players[0].Hand))
	}
}

// Regressão: mão/prêmios eram fatias do array do deck; ganhos diretos à mão
// (pegar prêmio, busca) sobrescreviam cartas vivas do deck e dos prêmios.
func TestNoZoneAliasingOnHandGrowth(t *testing.T) {
	g := newTestGame(t)
	ps := g.Players[0]
	deck := append([]string(nil), ps.Deck...)
	prizes := append([]string(nil), ps.Prizes...)
	// Simula ganhos à mão sem compra (mesmo append de TakePrize/busca).
	for i := 0; i < 10; i++ {
		ps.Hand = append(ps.Hand, "t-item")
	}
	for i, id := range ps.Deck {
		if id != deck[i] {
			t.Fatalf("deck corrompido em %d: %q → %q", i, deck[i], id)
		}
	}
	for i, id := range ps.Prizes {
		if id != prizes[i] {
			t.Fatalf("prêmios corrompidos em %d: %q → %q", i, prizes[i], id)
		}
	}
}

func TestFirstTurnRestrictions(t *testing.T) {
	g := newTestGame(t)
	// Liga energia pra pagar custo e tenta atacar no turno 1.
	eIdx := findInHand(t, g, 0, cards.CategoryEnergy)
	if err := g.AttachEnergy(0, eIdx, ActiveSlot); err != nil {
		t.Fatal(err)
	}
	if err := g.Attack(0, 0); err == nil {
		t.Error("ataque no turno 1 de quem começa deveria falhar")
	}
	// Segunda energia no mesmo turno: proibido.
	eIdx = findInHand(t, g, 0, cards.CategoryEnergy)
	if err := g.AttachEnergy(0, eIdx, ActiveSlot); err == nil {
		t.Error("segunda energia no turno deveria falhar")
	}
}

func TestAttackWeaknessKOAndPrize(t *testing.T) {
	g := newTestGame(t)
	// Jogador 1 (Aquaduck) ataca Firebug (fraqueza Water): 30×2 = 60 ≥ HP 60 → KO.
	if err := g.EndTurn(0); err != nil {
		t.Fatal(err)
	}
	// Banco do jogador 0 pra partida não acabar no KO.
	g.Players[0].Bench = append(g.Players[0].Bench, &PokemonInPlay{Stack: []string{"t-fire1"}})
	g.Players[1].Active.Energies = []string{"t-waterE"}
	if err := g.Attack(1, 0); err != nil {
		t.Fatal(err)
	}
	if g.Players[0].Active != nil {
		t.Fatal("Firebug deveria ter sido nocauteado")
	}
	if !g.NeedPromote[0] {
		t.Error("jogador 0 deveria ter promoção pendente")
	}
	if g.Players[1].PrizesTaken != 1 {
		t.Errorf("jogador 1 pegou %d prêmios (esperado 1)", g.Players[1].PrizesTaken)
	}
	// Ações bloqueadas até promover.
	if err := g.EndTurn(g.Current); err == nil {
		t.Error("ações deveriam estar bloqueadas com promoção pendente")
	}
	if err := g.Promote(0, 0); err != nil {
		t.Fatal(err)
	}
	if g.Players[0].Active == nil {
		t.Fatal("promoção não colocou Ativo")
	}
}

func TestAttackRequiresEnergy(t *testing.T) {
	g := newTestGame(t)
	if err := g.EndTurn(0); err != nil {
		t.Fatal(err)
	}
	// Aquaduck sem energia: custo não pago.
	if err := g.Attack(1, 0); err == nil || !strings.Contains(err.Error(), "custo") {
		t.Errorf("ataque sem energia deveria falhar por custo, veio: %v", err)
	}
}

func TestEvolveRestrictions(t *testing.T) {
	g := newTestGame(t)
	ps := g.Players[0]
	ps.Hand = append(ps.Hand, "t-fire2")
	evoIdx := len(ps.Hand) - 1
	// Primeiro turno do jogador: não evolui.
	if err := g.Evolve(0, evoIdx, ActiveSlot); err == nil {
		t.Error("evolução no primeiro turno do jogador deveria falhar")
	}
	// Avança um ciclo de turnos.
	if err := g.EndTurn(0); err != nil {
		t.Fatal(err)
	}
	if err := g.EndTurn(1); err != nil {
		t.Fatal(err)
	}
	evoIdx = -1
	for i, id := range ps.Hand {
		if id == "t-fire2" {
			evoIdx = i
		}
	}
	if err := g.Evolve(0, evoIdx, ActiveSlot); err != nil {
		t.Fatal(err)
	}
	if g.Card(ps.Active.TopID()).Name.EN != "Firebug II" {
		t.Errorf("topo deveria ser Firebug II, é %s", g.Card(ps.Active.TopID()).Name.EN)
	}
	// Evoluir de novo no mesmo turno: proibido (mesmo com carta válida na mão).
	ps.Hand = append(ps.Hand, "t-fire2")
	if err := g.Evolve(0, len(ps.Hand)-1, ActiveSlot); err == nil {
		t.Error("segunda evolução no mesmo turno deveria falhar")
	}
}

func TestPoisonCheckupAndConditionClearOnRetreat(t *testing.T) {
	g := newTestGame(t)
	if err := g.SetCondition(0, "poisoned"); err != nil {
		t.Fatal(err)
	}
	if err := g.EndTurn(0); err != nil {
		t.Fatal(err)
	}
	if g.Players[0].Active.Damage != 10 {
		t.Errorf("veneno deveria dar 10 de dano no checkup, tem %d", g.Players[0].Active.Damage)
	}
	// Recuo remove condições: prepara banco e energia pro custo (Recuo 1).
	if err := g.EndTurn(1); err != nil {
		t.Fatal(err)
	}
	ps := g.Players[0]
	ps.Bench = append(ps.Bench, &PokemonInPlay{Stack: []string{"t-fire1"}})
	ps.Active.Energies = []string{"t-fireE"}
	if err := g.Retreat(0, 0, []int{0}); err != nil {
		t.Fatal(err)
	}
	if ps.Bench[0].Poisoned {
		t.Error("recuo deveria remover Envenenado")
	}
	if len(ps.Active.Energies) != 0 && len(ps.Bench[0].Energies) != 0 {
		t.Error("energia do recuo deveria ter sido descartada")
	}
}

func TestSupporterOncePerTurn(t *testing.T) {
	g := newTestGame(t)
	ps := g.Players[0]
	ps.Hand = append(ps.Hand, "t-sup", "t-sup")
	// Quem começa não joga Suporte no turno 1.
	if err := g.PlaySupporter(0, len(ps.Hand)-1); err == nil {
		t.Error("Suporte no turno 1 de quem começa deveria falhar")
	}
	if err := g.EndTurn(0); err != nil {
		t.Fatal(err)
	}
	if err := g.EndTurn(1); err != nil {
		t.Fatal(err)
	}
	idx := -1
	for i, id := range ps.Hand {
		if id == "t-sup" {
			idx = i
			break
		}
	}
	if err := g.PlaySupporter(0, idx); err != nil {
		t.Fatal(err)
	}
	for i, id := range ps.Hand {
		if id == "t-sup" {
			idx = i
			break
		}
	}
	if err := g.PlaySupporter(0, idx); err == nil {
		t.Error("segundo Suporte no turno deveria falhar")
	}
}

func TestDeterminism(t *testing.T) {
	run := func() []string {
		g := newTestGame(t)
		_ = g.EndTurn(0)
		_ = g.EndTurn(1)
		return append([]string{}, g.Log...)
	}
	a, b := run(), run()
	if strings.Join(a, "\n") != strings.Join(b, "\n") {
		t.Error("mesma seed deveria produzir o mesmo log")
	}
}
