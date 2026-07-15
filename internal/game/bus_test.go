package game

import (
	"testing"

	"github.com/alvaropsouza/pokemon-self-play/internal/cards"
)

func TestTriggerFiresForCardInPlay(t *testing.T) {
	var got []Trigger
	triggerDB["t-fire1"] = func(g *Game, owner, slot int, tr Trigger) {
		if owner == 0 && slot == ActiveSlot {
			got = append(got, tr)
		}
	}
	defer delete(triggerDB, "t-fire1")

	g := newTestGame(t)
	if err := g.AttachEnergy(0, findInHand(t, g, 0, cards.CategoryEnergy), ActiveSlot); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Kind != TrigEnergyAttached || got[0].Player != 0 || got[0].Slot != ActiveSlot {
		t.Fatalf("esperava 1 TrigEnergyAttached do jogador 0 no Ativo, recebi %v", got)
	}

	if err := g.EndTurn(0); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[1].Kind != TrigTurnEnded {
		t.Fatalf("esperava TrigTurnEnded após fim de turno, recebi %v", got)
	}
}

func TestRiskyRuinsDamagesNonDarknessBenchedBasic(t *testing.T) {
	g := newTestGame(t)
	g.Stadium = "me01-127"
	ps := g.Players[0]

	ps.Hand = append(ps.Hand, "t-fire1", "t-dark1")
	if err := g.PlaceBench(0, len(ps.Hand)-2); err != nil {
		t.Fatal(err)
	}
	if got := ps.Bench[0].Damage; got != 20 {
		t.Fatalf("Básico não-Escuridão deveria receber 20 de dano, recebeu %d", got)
	}
	if err := g.PlaceBench(0, len(ps.Hand)-1); err != nil {
		t.Fatal(err)
	}
	if got := ps.Bench[1].Damage; got != 0 {
		t.Fatalf("Básico de Escuridão não deveria receber dano, recebeu %d", got)
	}
}

func TestRiskyRuinsIgnoresSetupPlacement(t *testing.T) {
	d := deck60(map[string]int{"t-fire1": 30, "t-fireE": 30})
	d2 := deck60(map[string]int{"t-water1": 30, "t-waterE": 30})
	g, err := New(testStore(), [2][]string{d, d2}, 42, 0)
	if err != nil {
		t.Fatal(err)
	}
	g.Stadium = "me01-127"
	if err := g.PlaceActive(0, findInHand(t, g, 0, cards.CategoryPokemon)); err != nil {
		t.Fatal(err)
	}
	if err := g.PlaceBench(0, findInHand(t, g, 0, cards.CategoryPokemon)); err != nil {
		t.Fatal(err)
	}
	if got := g.Players[0].Bench[0].Damage; got != 0 {
		t.Fatalf("colocação no setup não deveria disparar o Estádio, dano = %d", got)
	}
}

func TestTriggerSilentWithoutHandler(t *testing.T) {
	g := newTestGame(t)
	if err := g.AttachEnergy(0, findInHand(t, g, 0, cards.CategoryEnergy), ActiveSlot); err != nil {
		t.Fatal(err)
	}
}
