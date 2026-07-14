package bot

import "testing"

// TestBattleDecksValid: todo Battle Deck fixo passa no validador (60 cartas,
// máx. 4 por nome, ≥1 Básico, legalidade Standard).
func TestBattleDecksValid(t *testing.T) {
	store := loadStore(t)
	infos := BattleDecks()
	// 2 por tipo, exceto Dragon (pool me01 só tem a linha Latias/Latios).
	if len(infos) != 19 {
		t.Fatalf("esperado 19 Battle Decks, veio %d", len(infos))
	}
	for _, info := range infos {
		d, err := BattleDeck(store, info.ID)
		if err != nil {
			t.Errorf("%s (%s): %v", info.Name, info.Type, err)
			continue
		}
		if d.Size() != 60 {
			t.Errorf("%s: %d cartas", info.Name, d.Size())
		}
		if store.Cards[info.Star] == nil {
			t.Errorf("%s: carta-estrela %s não existe na base", info.Name, info.Star)
		}
	}
}
