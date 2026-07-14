// Package cards define o modelo canônico de cartas e a base local bilíngue.
//
// Cada carta física existe em EN e PT-BR compartilhando o mesmo ID canônico
// (ex.: "me01-001"). A mecânica (HP, custos, dano) vem da versão EN; nome e
// textos são armazenados nos dois idiomas. O motor de regras opera apenas
// sobre o ID canônico e os campos mecânicos — idioma é atributo de
// exibição/reconhecimento.
package cards

// Localized guarda um texto nos dois idiomas da coleção.
type Localized struct {
	EN string `json:"en,omitempty"`
	PT string `json:"pt,omitempty"`
}

// Category é a categoria principal da carta.
type Category string

const (
	CategoryPokemon Category = "Pokemon"
	CategoryTrainer Category = "Trainer"
	CategoryEnergy  Category = "Energy"
)

// Attack descreve um ataque de Pokémon.
type Attack struct {
	Name   Localized `json:"name"`
	Effect Localized `json:"effect,omitempty"`
	// Cost é a lista de tipos de Energia exigidos (ex.: ["Grass","Colorless"]).
	Cost []string `json:"cost,omitempty"`
	// Damage é o dano impresso; string porque pode ser "30+", "20×" ou vazio.
	Damage string `json:"damage,omitempty"`
}

// Ability descreve uma Habilidade (ou Poké-Power/Body em cartas antigas).
type Ability struct {
	Type   string    `json:"type,omitempty"`
	Name   Localized `json:"name"`
	Effect Localized `json:"effect,omitempty"`
}

// TypeValue é um par tipo/valor usado em Fraqueza e Resistência.
type TypeValue struct {
	Type  string `json:"type"`
	Value string `json:"value,omitempty"`
}

// Card é o registro canônico de uma carta na base local.
type Card struct {
	// ID canônico compartilhado entre idiomas (ex.: "me01-001").
	ID      string   `json:"id"`
	SetID   string   `json:"setId"`
	SetName Localized `json:"setName"`
	// LocalID é o número da carta dentro da coleção (igual nos dois idiomas).
	LocalID  string    `json:"localId"`
	Name     Localized `json:"name"`
	Category Category  `json:"category"`
	// RegulationMark é a letra de legalidade (H, I, J...). Vazio em cartas antigas.
	RegulationMark string `json:"regulationMark,omitempty"`
	Rarity         string `json:"rarity,omitempty"`
	// Image é a URL base da imagem por idioma (TCGdex; sufixos /low.webp, /high.webp).
	Image Localized `json:"image,omitempty"`

	// Campos de Pokémon.
	HP          int         `json:"hp,omitempty"`
	Types       []string    `json:"types,omitempty"`
	Stage       string      `json:"stage,omitempty"`
	EvolveFrom  Localized   `json:"evolveFrom,omitempty"`
	Attacks     []Attack    `json:"attacks,omitempty"`
	Abilities   []Ability   `json:"abilities,omitempty"`
	Weaknesses  []TypeValue `json:"weaknesses,omitempty"`
	Resistances []TypeValue `json:"resistances,omitempty"`
	Retreat     int         `json:"retreat,omitempty"`

	// Campos de Treinador/Energia.
	// TrainerType: Item, Supporter, Tool, Stadium...
	TrainerType string `json:"trainerType,omitempty"`
	// EnergyType: Basic ou Special.
	EnergyType string    `json:"energyType,omitempty"`
	Effect     Localized `json:"effect,omitempty"`
}

// StandardLegal informa se a carta é legal no Standard vigente (marcas H/I/J).
func (c *Card) StandardLegal() bool {
	switch c.RegulationMark {
	case "H", "I", "J":
		return true
	}
	return false
}

// IsBasicPokemon reporta se é um Pokémon Básico.
func (c *Card) IsBasicPokemon() bool {
	return c.Category == CategoryPokemon && c.Stage == "Basic"
}

// IsBasicEnergy reporta se é uma Energia Básica (TCGdex usa EnergyType != "Special").
func (c *Card) IsBasicEnergy() bool {
	return c.Category == CategoryEnergy && c.EnergyType != "Special"
}
