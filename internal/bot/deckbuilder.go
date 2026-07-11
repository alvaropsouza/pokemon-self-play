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
)

// BuildDeck monta um deck de 60 cartas do(s) tipo(s) dado(s) usando o pool da
// base. Determinístico: mesmo pool + tipos + seed → mesmo deck.
//
// Esqueleto: linhas de evolução do tipo (básico + estágios do pool) até ~20
// Pokémon, completado com Energias Básicas do tipo principal. Treinadores
// ficam de fora enquanto o bot não sabe usá-los (efeitos são arbitragem
// manual — ver PLANO.md etapa 3).
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

	// Candidatos: básicos legais do tipo, com ao menos 1 ataque com dano.
	var basics []*cards.Card
	evoByFrom := map[string][]*cards.Card{} // nome EN do pré-evoluído → evoluções
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

	// Variedade pela seed: embaralha e reordena estável por prioridade
	// (linha com evolução primeiro, depois HP) — empates ficam na ordem sorteada.
	rng.Shuffle(len(basics), func(i, j int) { basics[i], basics[j] = basics[j], basics[i] })
	sort.SliceStable(basics, func(i, j int) bool {
		ei, ej := len(evoByFrom[basics[i].Name.EN]) > 0, len(evoByFrom[basics[j].Name.EN]) > 0
		if ei != ej {
			return ei
		}
		return basics[i].HP > basics[j].HP
	})

	d := deck.New()
	seenName := map[string]bool{}
	pokemon := 0
	const pokemonTarget = 20
	for _, b := range basics {
		if pokemon >= pokemonTarget {
			break
		}
		if seenName[b.Name.EN] {
			continue
		}
		seenName[b.Name.EN] = true
		d.Add(b.ID, 3)
		pokemon += 3
		// Estágio 1 e, se houver, Estágio 2 da mesma linha.
		for _, e1 := range firstEvo(evoByFrom, b.Name.EN, seenName) {
			d.Add(e1.ID, 2)
			pokemon += 2
			seenName[e1.Name.EN] = true
			for _, e2 := range firstEvo(evoByFrom, e1.Name.EN, seenName) {
				d.Add(e2.ID, 1)
				pokemon++
				seenName[e2.Name.EN] = true
			}
		}
	}

	// Completa com Energia Básica do tipo principal.
	energyID := ""
	wantEnergy := types[0] + " Energy"
	for _, c := range pool {
		if c.Category == cards.CategoryEnergy && c.EnergyType != "Special" &&
			strings.TrimPrefix(c.Name.EN, "Basic ") == wantEnergy {
			energyID = c.ID
			break
		}
	}
	if energyID == "" {
		return nil, fmt.Errorf("pool sem %s (importe o set de energias, ex.: sve)", wantEnergy)
	}
	if d.Size() > 60 {
		return nil, fmt.Errorf("esqueleto passou de 60 cartas (%d)", d.Size())
	}
	d.Add(energyID, 60-d.Size())

	if errs := d.Validate(store); len(errs) > 0 {
		return nil, fmt.Errorf("deck gerado inválido: %v", errs)
	}
	return d, nil
}

// firstEvo devolve no máximo 1 evolução ainda não usada de um nome.
func firstEvo(evoByFrom map[string][]*cards.Card, from string, seen map[string]bool) []*cards.Card {
	for _, e := range evoByFrom[from] {
		if !seen[e.Name.EN] {
			return []*cards.Card{e}
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
