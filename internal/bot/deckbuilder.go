// Package bot implementa o oponente automatizado: construção de deck a partir
// do pool da API (por tipo, determinístico por seed) e o piloto de turno.
// Ver seção "Construção de deck pelo oponente" do CLAUDE.md.
package bot

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"

	"github.com/alvaropsouza/pokemon-self-play/internal/cards"
	"github.com/alvaropsouza/pokemon-self-play/internal/deck"
	"github.com/alvaropsouza/pokemon-self-play/internal/game"
)

const (
	pokemonTarget = 16
	// 14 (não os ~10 de deck competitivo): sem draw engines automatizados,
	// 10 energias deixam ~26% das mãos iniciais sem energia nenhuma.
	energyTarget = 14
)

// BuildDeck monta um deck de 60 cartas do(s) tipo(s) dado(s) usando o pool da
// base. Determinístico: mesmo pool + tipos + seed → mesmo deck.
//
// Estrutura meta: ~16 Pokémon, ~10 Energia Básica, ~34 Treinadores.
// Linha principal 4-3-2, secundária 3-2-1, treinadores em round-robin por subtipo.
func BuildDeck(store *cards.Store, types []string, seed int64) (*deck.Deck, error) {
	if len(types) == 0 {
		return nil, fmt.Errorf("escolha ao menos 1 tipo")
	}
	rng := rand.New(rand.NewSource(seed))

	var pool []*cards.Card
	for _, c := range store.Cards {
		pool = append(pool, c)
	}
	sort.Slice(pool, func(i, j int) bool { return pool[i].ID < pool[j].ID })

	wantType := func(c *cards.Card) bool {
		for _, t := range c.Types {
			for _, w := range types {
				if t == w {
					return true
				}
			}
		}
		return false
	}

	var basics []*cards.Card
	evoByFrom := map[string][]*cards.Card{}
	for _, c := range pool {
		if c.Category != cards.CategoryPokemon || !c.StandardLegal() || !wantType(c) {
			continue
		}
		if c.Stage == "Basic" {
			if hasDamagingAttack(c) {
				basics = append(basics, c)
			}
			continue
		}
		if c.EvolveFrom.EN != "" {
			evoByFrom[c.EvolveFrom.EN] = append(evoByFrom[c.EvolveFrom.EN], c)
		}
	}
	if len(basics) == 0 {
		return nil, fmt.Errorf("pool sem Pokémon Básico atacante do(s) tipo(s) %v", types)
	}

	rng.Shuffle(len(basics), func(i, j int) { basics[i], basics[j] = basics[j], basics[i] })
	sort.SliceStable(basics, func(i, j int) bool {
		ei, ej := len(evoByFrom[basics[i].Name.EN]) > 0, len(evoByFrom[basics[j].Name.EN]) > 0
		if ei != ej {
			return ei
		}
		return basics[i].HP > basics[j].HP
	})

	d := deck.New()
	byName := map[string]int{} // name EN → total copies in deck

	addCard := func(c *cards.Card, n int) int {
		avail := 4 - byName[c.Name.EN]
		if n > avail {
			n = avail
		}
		if n <= 0 {
			return 0
		}
		d.Add(c.ID, n)
		byName[c.Name.EN] += n
		return n
	}

	seenName := map[string]bool{}
	pokemon := 0

	addPoke := func(c *cards.Card, n int) {
		if rem := pokemonTarget - pokemon; n > rem {
			n = rem
		}
		got := addCard(c, n)
		pokemon += got
		seenName[c.Name.EN] = true
	}

	// Main line: 4-3-2
	if len(basics) > 0 {
		b := basics[0]
		addPoke(b, 4)
		if e1 := firstEvo(evoByFrom, b.Name.EN, seenName); e1 != nil {
			addPoke(e1, 3)
			if e2 := firstEvo(evoByFrom, e1.Name.EN, seenName); e2 != nil {
				addPoke(e2, 2)
			}
		}
	}

	// Secondary lines: 3-2-1
	for _, b := range basics[1:] {
		if pokemon >= pokemonTarget {
			break
		}
		if seenName[b.Name.EN] {
			continue
		}
		addPoke(b, 3)
		if e1 := firstEvo(evoByFrom, b.Name.EN, seenName); e1 != nil {
			addPoke(e1, 2)
			if e2 := firstEvo(evoByFrom, e1.Name.EN, seenName); e2 != nil {
				addPoke(e2, 1)
			}
		}
	}

	// Energy
	energyID := findBasicEnergy(pool, types[0])
	if energyID == "" {
		return nil, fmt.Errorf("pool sem %s Energy (importe o set de energias, ex.: sve)", types[0])
	}
	d.Add(energyID, energyTarget)

	// Trainers: fill remaining slots (~34)
	remaining := 60 - d.Size()
	if remaining > 0 {
		fillTrainers(d, byName, pool, rng, remaining, types)
	}

	// Fallback: pad with energy if no trainers available
	if d.Size() < 60 {
		d.Add(energyID, 60-d.Size())
	}

	if errs := d.Validate(store); len(errs) > 0 {
		return nil, fmt.Errorf("deck gerado inválido: %v", errs)
	}
	return d, nil
}

func findBasicEnergy(pool []*cards.Card, typ string) string {
	want := typ + " Energy"
	for _, c := range pool {
		if c.Category == cards.CategoryEnergy && c.EnergyType != "Special" &&
			strings.TrimPrefix(c.Name.EN, "Basic ") == want {
			return c.ID
		}
	}
	return ""
}

// fillTrainers preenche target slots com Treinadores legais no Standard,
// distribuídos por subtipo em round-robin (até 4 cópias por nome).
// deckTypes é usado para excluir Itens/Suportes que buscam tipos incompatíveis.
func fillTrainers(d *deck.Deck, byName map[string]int, pool []*cards.Card, rng *rand.Rand, target int, deckTypes []string) {
	var supporters, items, tools, stadiums []*cards.Card
	for _, c := range pool {
		if c.Category != cards.CategoryTrainer || !c.StandardLegal() {
			continue
		}
		if strings.Contains(strings.ToUpper(c.Rarity), "ACE SPEC") {
			continue
		}
		switch c.TrainerType {
		case "Supporter":
			if trainerCompatible(c, deckTypes) {
				supporters = append(supporters, c)
			}
		case "Tool":
			tools = append(tools, c)
		case "Stadium":
			stadiums = append(stadiums, c)
		default:
			if trainerCompatible(c, deckTypes) {
				items = append(items, c)
			}
		}
	}
	for _, g := range []*[]*cards.Card{&supporters, &items, &tools, &stadiums} {
		rng.Shuffle(len(*g), func(i, j int) { (*g)[i], (*g)[j] = (*g)[j], (*g)[i] })
	}

	added := 0
	// quotas: supporters ~12, items ~15, tools ~4, stadiums ~3
	type groupSpec struct {
		cards []*cards.Card
		quota int
	}
	groups := []groupSpec{
		{supporters, 12},
		{items, 15},
		{tools, 4},
		{stadiums, 3},
	}

	addGroup := func(group []*cards.Card, quota int) {
		if quota > target-added {
			quota = target - added
		}
		for quota > 0 {
			prev := quota
			for _, c := range group {
				if quota == 0 {
					break
				}
				if byName[c.Name.EN] < 4 {
					d.Add(c.ID, 1)
					byName[c.Name.EN]++
					added++
					quota--
				}
			}
			if quota == prev {
				break // no progress, group exhausted
			}
		}
	}

	for _, g := range groups {
		addGroup(g.cards, g.quota)
	}
	// Fill any remainder with items
	if added < target {
		addGroup(items, target-added)
	}
}

func firstEvo(evoByFrom map[string][]*cards.Card, from string, seen map[string]bool) *cards.Card {
	for _, e := range evoByFrom[from] {
		if !seen[e.Name.EN] {
			return e
		}
	}
	return nil
}

func hasDamagingAttack(c *cards.Card) bool {
	for _, a := range c.Attacks {
		if len(a.Damage) > 0 && a.Damage[0] >= '1' && a.Damage[0] <= '9' {
			return true
		}
	}
	return false
}

// trainerCompatible retorna false se o treinador tiver busca de tipo específico
// incompatível com o deck (ex.: Fighting Gong num deck Fire). Cartas type-neutral
// ou com efeito manual passam sempre.
func trainerCompatible(c *cards.Card, deckTypes []string) bool {
	ce := game.CompileEffect(c.Effect.EN)
	for _, op := range ce.Ops {
		if op.Kind != game.OpSearch {
			continue
		}
		compatible := false
		for _, f := range op.Find {
			if f.Type == "" {
				compatible = true
				break
			}
			for _, dt := range deckTypes {
				if dt == f.Type {
					compatible = true
					break
				}
			}
			if compatible {
				break
			}
		}
		if !compatible {
			return false
		}
	}
	return true
}
