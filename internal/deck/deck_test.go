package deck

import (
	"strings"
	"testing"

	"github.com/alvaropsouza/pokemon-self-play/internal/cards"
)

func testStore() *cards.Store {
	s := cards.NewStore()
	put := func(c cards.Card) { s.Put(&c) }
	put(cards.Card{ID: "p1", Name: cards.Localized{EN: "Firebug"}, Category: cards.CategoryPokemon, Stage: "Basic", RegulationMark: "H"})
	put(cards.Card{ID: "p1-alt", Name: cards.Localized{EN: "Firebug"}, Category: cards.CategoryPokemon, Stage: "Basic", RegulationMark: "I"})
	put(cards.Card{ID: "old", Name: cards.Localized{EN: "Oldmon"}, Category: cards.CategoryPokemon, Stage: "Basic", RegulationMark: "G"})
	put(cards.Card{ID: "e1", Name: cards.Localized{EN: "Fire Energy"}, Category: cards.CategoryEnergy, EnergyType: "Basic"})
	put(cards.Card{ID: "ace", Name: cards.Localized{EN: "Mega Item"}, Category: cards.CategoryTrainer, TrainerType: "Item", Rarity: "ACE SPEC Rare", RegulationMark: "H"})
	put(cards.Card{ID: "ace2", Name: cards.Localized{EN: "Mega Rod"}, Category: cards.CategoryTrainer, TrainerType: "Item", Rarity: "ACE SPEC Rare", RegulationMark: "H"})
	return s
}

func TestValidateOK(t *testing.T) {
	d := New()
	d.Add("p1", 4)
	d.Add("ace", 1)
	d.Add("e1", 55)
	if errs := d.Validate(testStore()); len(errs) != 0 {
		t.Fatalf("deck deveria ser válido: %v", errs)
	}
}

func TestValidateErrors(t *testing.T) {
	d := New()
	d.Add("p1", 3)
	d.Add("p1-alt", 2) // 5 cópias do mesmo nome somando impressões
	d.Add("old", 1)    // fora do Standard
	d.Add("ace", 1)
	d.Add("ace2", 1) // 2 ACE SPEC
	d.Add("e1", 50)  // total 58 ≠ 60
	errs := d.Validate(testStore())
	want := []string{"cópias", "Standard", "ACE SPEC", "58"}
	for _, w := range want {
		found := false
		for _, e := range errs {
			if contains(e.Error(), w) {
				found = true
			}
		}
		if !found {
			t.Errorf("faltou erro contendo %q em %v", w, errs)
		}
	}
}

func TestValidateNoBasic(t *testing.T) {
	d := New()
	d.Add("e1", 60)
	errs := d.Validate(testStore())
	found := false
	for _, e := range errs {
		if contains(e.Error(), "Básico") {
			found = true
		}
	}
	if !found {
		t.Errorf("faltou erro de deck sem Básico: %v", errs)
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }
