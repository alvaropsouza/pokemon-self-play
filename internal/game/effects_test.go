package game

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/alvaropsouza/pokemon-self-play/internal/cards"
)

func TestCompileEffect(t *testing.T) {
	tests := []struct {
		name string
		text string
		want CompiledEffect
	}{
		{"vazio", "", CompiledEffect{}},
		{"professor's research", "Discard your hand and draw 7 cards.",
			CompiledEffect{Ops: []Op{{Kind: OpDiscardHand}, {Kind: OpDraw, N: 7}}}},
		{"hop", "Draw 3 cards.",
			CompiledEffect{Ops: []Op{{Kind: OpDraw, N: 3}}}},
		{"draw until", "Draw cards until you have 5 cards in your hand.",
			CompiledEffect{Ops: []Op{{Kind: OpDrawUntil, N: 5}}}},
		{"iono", "Each player shuffles their hand into the bottom of their deck. Then, each player draws a card for each of their remaining Prize cards.",
			CompiledEffect{Ops: []Op{{Kind: OpShuffleHandBoth}, {Kind: OpDrawPerPrizeBoth}}}},
		{"judge", "Each player shuffles their hand into their deck and draws 4 cards.",
			CompiledEffect{Ops: []Op{{Kind: OpShuffleHandBoth}, {Kind: OpDrawBoth, N: 4}}}},
		{"bench spread opp", "This attack also does 30 damage to each of your opponent's Benched Pokémon. (Don't apply Weakness and Resistance for Benched Pokémon.)",
			CompiledEffect{Ops: []Op{{Kind: OpDamageOppBench, N: 30}}}},
		{"heal", "Heal 30 damage from this Pokémon.",
			CompiledEffect{Ops: []Op{{Kind: OpHealSelf, N: 30}}}},
		{"discard all energy", "Discard all Energy from this Pokémon.",
			CompiledEffect{Ops: []Op{{Kind: OpDiscardSelfEnergy, N: -1}}}},
		{"discard 2 energy", "Discard 2 Fire Energy from this Pokémon.",
			CompiledEffect{Ops: []Op{{Kind: OpDiscardSelfEnergy, N: 2}}}},
		{"scale self", "This attack does 30 more damage for each Water Energy attached to this Pokémon.",
			CompiledEffect{Ops: []Op{{Kind: OpScalePerEnergySelf, N: 30}}}},
		{"status opp", "Your opponent's Active Pokémon is now Poisoned.",
			CompiledEffect{Ops: []Op{{Kind: OpStatus, Cond: "poisoned"}}}},
		{"status duplo", "The Defending Pokémon is now Asleep and Poisoned.",
			CompiledEffect{Ops: []Op{{Kind: OpStatus, Cond: "asleep"}, {Kind: OpStatus, Cond: "poisoned"}}}},
		{"status self", "This Pokémon is now Confused.",
			CompiledEffect{Ops: []Op{{Kind: OpStatus, Cond: "confused", OnSelf: true}}}},
		{"flip status", "Flip a coin. If heads, the Defending Pokémon is now Paralyzed.",
			CompiledEffect{Ops: []Op{{Kind: OpStatus, Cond: "paralyzed", Flip: true}}}},
		{"descarte da mão + draw", "Discard 2 cards from your hand. Then, draw 4 cards.",
			CompiledEffect{Ops: []Op{{Kind: OpDiscardFromHand, N: 2}, {Kind: OpDraw, N: 4}}}},
		{"ultra ball", "You can use this card only if you discard 2 other cards from your hand.\n\nSearch your deck for a Pokémon, reveal it, and put it into your hand. Then, shuffle your deck.",
			CompiledEffect{Ops: []Op{
				{Kind: OpDiscardFromHand, N: 2, Cost: true},
				{Kind: OpSearch, N: 1, Dest: "hand", Find: []Find{{Category: "Pokemon"}}},
				{Kind: OpShuffleDeck},
			}}},
		// Cobertura total: cláusula extra não coberta → manual, sem resolução parcial.
		{"parcial vira manual", "Discard 2 cards from your hand. Then, banana 4 cards.",
			CompiledEffect{Manual: true}},
		{"condicional vira manual", "You can play this card only if you have exactly 6 Prize cards. Draw 3 cards.",
			CompiledEffect{Manual: true}},
		{"flip sem heads coberto vira manual", "Flip a coin. If tails, this attack does nothing.",
			CompiledEffect{Manual: true}},
		{"busca para o banco", "Search your deck for a Basic Pokémon and put it onto your Bench.",
			CompiledEffect{Ops: []Op{{Kind: OpSearch, N: 1, Dest: "bench", Find: []Find{{Category: "Pokemon", Stage: "Basic"}}}}}},
		{"fighting gong", "Search your deck for a Basic {F} Energy card or a Basic {F} Pokémon, reveal it, and put it into your hand. Then, shuffle your deck.",
			CompiledEffect{Ops: []Op{
				{Kind: OpSearch, N: 1, Dest: "hand", Find: []Find{{Category: "Energy", Type: "Fighting"}, {Category: "Pokemon", Stage: "Basic", Type: "Fighting"}}},
				{Kind: OpShuffleDeck},
			}}},
		{"busca até 2 para o banco", "Search your deck for up to 2 Basic Pokémon and put them onto your Bench. Then, shuffle your deck.",
			CompiledEffect{Ops: []Op{
				{Kind: OpSearch, N: 2, Dest: "bench", Find: []Find{{Category: "Pokemon", Stage: "Basic"}}},
				{Kind: OpShuffleDeck},
			}}},
		{"buddy-buddy poffin", "Search your deck for up to 2 Basic Pokémon with 70 HP or less and put them onto your Bench. Then, shuffle your deck.",
			CompiledEffect{Ops: []Op{
				{Kind: OpSearch, N: 2, Dest: "bench", Find: []Find{{Category: "Pokemon", Stage: "Basic", MaxHP: 70}}},
				{Kind: OpShuffleDeck},
			}}},
		{"busca de evolução vira manual", "Search your deck for a Mega Evolution Pokémon ex, reveal it, and put it into your hand. Then, shuffle your deck.",
			CompiledEffect{Manual: true}},
		{"lillie's determination", "Shuffle your hand into your deck. Then, draw 6 cards. If you have exactly 6 Prize cards remaining, draw 8 cards instead.",
			CompiledEffect{Ops: []Op{
				{Kind: OpShuffleHandSelf},
				{Kind: OpDrawOrMore, N: 6, Alt: 8, ExactPrizes: 6},
			}}},
		{"nest ball", "Search your deck for a Basic Pokémon and put it onto your Bench. Shuffle your deck afterward.",
			CompiledEffect{Ops: []Op{
				{Kind: OpSearch, N: 1, Dest: "bench", Find: []Find{{Category: "Pokemon", Stage: "Basic"}}},
				{Kind: OpShuffleDeck},
			}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compile(tt.text)
			if got.Manual != tt.want.Manual || fmt.Sprint(got.Ops) != fmt.Sprint(tt.want.Ops) {
				t.Errorf("compile(%q)\n got %+v\nwant %+v", tt.text, got, tt.want)
			}
		})
	}
}

// TestEffectCoverage compila todos os efeitos de data/cards.json e reporta a
// cobertura automática, listando as cláusulas manuais mais frequentes.
// Não falha — é ferramenta de medição para guiar padrões novos.
func TestEffectCoverage(t *testing.T) {
	s, err := cards.Load("../../data/cards.json")
	if err != nil || len(s.Cards) == 0 {
		t.Skip("data/cards.json indisponível")
	}
	type bucket struct{ auto, total int }
	var trainers, attacks bucket
	unmatched := map[string]int{}

	countText := func(b *bucket, text string) {
		if text == "" {
			return
		}
		b.total++
		if !CompileEffect(text).Manual {
			b.auto++
			return
		}
		low := strings.ReplaceAll(strings.ToLower(text), "é", "e")
		low = reParen.ReplaceAllString(low, "")
		for _, cl := range strings.Split(low, ".") {
			if cl = strings.TrimSpace(cl); cl != "" {
				unmatched[cl]++
			}
		}
	}
	for _, c := range s.Cards {
		if !c.StandardLegal() {
			continue
		}
		if c.Category == cards.CategoryTrainer {
			countText(&trainers, c.Effect.EN)
		}
		for _, atk := range c.Attacks {
			countText(&attacks, atk.Effect.EN)
		}
	}

	t.Logf("treinadores: %d/%d auto (%.0f%%)", trainers.auto, trainers.total, 100*float64(trainers.auto)/float64(max(trainers.total, 1)))
	t.Logf("ataques:     %d/%d auto (%.0f%%)", attacks.auto, attacks.total, 100*float64(attacks.auto)/float64(max(attacks.total, 1)))

	type uc struct {
		clause string
		n      int
	}
	list := make([]uc, 0, len(unmatched))
	for cl, n := range unmatched {
		list = append(list, uc{cl, n})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].n > list[j].n })
	for i := 0; i < len(list) && i < 20; i++ {
		t.Logf("%3d× %s", list[i].n, list[i].clause)
	}
}
