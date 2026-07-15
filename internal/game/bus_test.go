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

func TestTriggerSilentWithoutHandler(t *testing.T) {
	g := newTestGame(t)
	if err := g.AttachEnergy(0, findInHand(t, g, 0, cards.CategoryEnergy), ActiveSlot); err != nil {
		t.Fatal(err)
	}
}
