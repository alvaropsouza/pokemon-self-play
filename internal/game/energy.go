package game

import (
	"strings"

	"github.com/alvaropsouza/pokemon-self-play/internal/cards"
)

// providedType devolve o tipo de Energia que uma carta ligada fornece.
// Energia Básica (TCGdex: energyType "Normal") fornece o tipo do nome
// ("Fire Energy" → "Fire"); Energia Especial é tratada como 1 Colorless
// (efeitos extras: arbitragem manual).
func providedType(c *cards.Card) string {
	if c == nil || c.Category != cards.CategoryEnergy {
		return ""
	}
	if c.EnergyType == "Special" {
		return "Colorless"
	}
	name := strings.TrimPrefix(c.Name.EN, "Basic ")
	fields := strings.Fields(name)
	if len(fields) > 0 {
		return fields[0]
	}
	return "Colorless"
}

// CostPaid verifica se as Energias ligadas pagam o custo do ataque.
// Requisitos tipados exigem o tipo exato; Colorless aceita qualquer Energia.
func (g *Game) CostPaid(t *PokemonInPlay, cost []string) bool {
	provided := map[string]int{}
	total := 0
	for _, id := range t.Energies {
		typ := providedType(g.Card(id))
		if typ == "" {
			continue
		}
		provided[typ]++
		total++
	}
	if total < len(cost) {
		return false
	}
	for _, req := range cost {
		if req == "Colorless" {
			continue
		}
		if provided[req] == 0 {
			return false
		}
		provided[req]--
	}
	return true
}
