// Package deck define decklists e a validação de construção do formato
// Standard (seção "Construção do deck" do CLAUDE.md). Uma única implementação
// valida tanto o deck do jogador humano quanto o gerado pelo bot.
package deck

import (
	"fmt"
	"sort"
	"strings"

	"github.com/alvaropsouza/pokemon-self-play/internal/cards"
)

// Deck é uma decklist: ID canônico de carta → quantidade.
type Deck struct {
	Counts map[string]int `json:"counts"`
}

func New() *Deck { return &Deck{Counts: make(map[string]int)} }

// Add soma n cópias de uma carta.
func (d *Deck) Add(cardID string, n int) { d.Counts[cardID] += n }

// Size é o total de cartas.
func (d *Deck) Size() int {
	total := 0
	for _, n := range d.Counts {
		total += n
	}
	return total
}

// CardIDs expande a decklist em 60 IDs (com repetição), ordem determinística.
func (d *Deck) CardIDs() []string {
	ids := make([]string, 0, len(d.Counts))
	for id := range d.Counts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var out []string
	for _, id := range ids {
		for i := 0; i < d.Counts[id]; i++ {
			out = append(out, id)
		}
	}
	return out
}

// Validate confere as regras de construção: 60 cartas, máx. 4 por nome (exceto
// Energia Básica), máx. 1 ACE SPEC, ≥1 Pokémon Básico, legalidade Standard.
// Devolve todos os problemas encontrados (vazio = deck válido).
func (d *Deck) Validate(store *cards.Store) []error {
	var errs []error
	byName := map[string]int{}
	aceSpecs := 0
	hasBasic := false

	for id, n := range d.Counts {
		c := store.Cards[id]
		if c == nil {
			errs = append(errs, fmt.Errorf("carta %q não existe na base", id))
			continue
		}
		if n <= 0 {
			errs = append(errs, fmt.Errorf("%s: quantidade inválida (%d)", c.Name.EN, n))
			continue
		}
		// Energia Básica é sempre legal (qualquer impressão) e sem limite de cópias.
		if !isBasicEnergy(c) {
			if !c.StandardLegal() {
				errs = append(errs, fmt.Errorf("%s (%s): fora do Standard (marca %q)", c.Name.EN, id, c.RegulationMark))
			}
			byName[c.Name.EN] += n
		}
		if isAceSpec(c) {
			aceSpecs += n
		}
		if c.Category == cards.CategoryPokemon && c.Stage == "Basic" {
			hasBasic = true
		}
	}

	if total := d.Size(); total != 60 {
		errs = append(errs, fmt.Errorf("deck tem %d cartas (esperado 60)", total))
	}
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if byName[name] > 4 {
			errs = append(errs, fmt.Errorf("%s: %d cópias (máx. 4)", name, byName[name]))
		}
	}
	if aceSpecs > 1 {
		errs = append(errs, fmt.Errorf("%d cartas ACE SPEC (máx. 1)", aceSpecs))
	}
	if !hasBasic {
		errs = append(errs, fmt.Errorf("deck sem Pokémon Básico"))
	}
	return errs
}

// isBasicEnergy: TCGdex usa "Normal" para Energia Básica ("Special" para Especial).
func isBasicEnergy(c *cards.Card) bool {
	return c.Category == cards.CategoryEnergy && c.EnergyType != "Special"
}

func isAceSpec(c *cards.Card) bool {
	return strings.Contains(strings.ToUpper(c.Rarity), "ACE SPEC")
}
