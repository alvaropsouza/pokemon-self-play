// Battle Decks: decklists fixas, 1 por tipo, no formato dos produtos oficiais
// "ex Battle Deck" (estrela Mega ex + linha secundária + core de consistência +
// 16 energias). Os produtos oficiais reais quase todos rotacionaram (marca G),
// então as listas abaixo são curadas do pool me01 (Mega Evolution) + sve,
// sempre válidas pelo mesmo validador de deck.Validate.
package bot

import (
	"fmt"
	"sort"

	"github.com/alvaropsouza/pokemon-self-play/internal/cards"
	"github.com/alvaropsouza/pokemon-self-play/internal/deck"
)

// BattleDeckInfo descreve um Battle Deck disponível para seleção na UI.
type BattleDeckInfo struct {
	ID   string `json:"id"`   // slug estável, ex.: "grass-venusaur"
	Type string `json:"type"` // tipo de energia (Grass, Fire...)
	Name string `json:"name"` // nome de exibição, ex.: "Mega Venusaur ex Battle Deck"
	Star string `json:"star"` // ID da carta-estrela (para arte na UI)
}

type bdEntry struct {
	id string
	n  int
}

type battleDeck struct {
	typ  string
	name string
	star string
	list []bdEntry
}

// Core de treinadores (28 cartas) compartilhado; decks com Estágio 2 trocam
// Energy Switch + Air Balloon por 4 Rare Candy.
func trainerCore(stadiumID string, rareCandy bool) []bdEntry {
	core := []bdEntry{
		{"me01-167", 4}, // Buddy-Buddy Poffin
		{"me01-131", 4}, // Ultra Ball
		{"me01-130", 3}, // Switch
		{"me01-173", 2}, // Night Stretcher
		{"me01-119", 4}, // Lillie's Determination
		{"me01-114", 3}, // Boss's Orders
		{"me01-123", 2}, // Pokémon Center Lady
		{stadiumID, 2},
	}
	if rareCandy {
		core = append(core, bdEntry{"me01-125", 4}) // Rare Candy
	} else {
		core = append(core,
			bdEntry{"me01-115", 2}, // Energy Switch
			bdEntry{"me01-166", 2}, // Air Balloon
		)
	}
	return core
}

func energy(id string, n int) bdEntry { return bdEntry{id, n} }

var battleDecks = map[string]battleDeck{
	"grass-venusaur": {
		typ: "Grass", name: "Mega Venusaur ex Battle Deck", star: "me01-003",
		list: append([]bdEntry{
			{"me01-001", 4}, // Bulbasaur
			{"me01-002", 2}, // Ivysaur
			{"me01-003", 3}, // Mega Venusaur ex
			{"me01-006", 3}, // Tangela
			{"me01-007", 2}, // Tangrowth
			{"me01-012", 2}, // Celebi
			energy("sve-001", 16),
		}, trainerCore("me01-117", true)...), // Forest of Vitality
	},
	"fire-camerupt": {
		typ: "Fire", name: "Mega Camerupt ex Battle Deck", star: "me01-022",
		list: append([]bdEntry{
			{"me01-021", 4}, // Numel
			{"me01-022", 3}, // Mega Camerupt ex
			{"me01-025", 3}, // Volcanion
			{"me01-023", 3}, // Litleo
			{"me01-024", 2}, // Pyroar
			{"me01-031", 1}, // Chi-Yu
			energy("sve-002", 16),
		}, trainerCore("me01-122", false)...), // Mystery Garden
	},
	"water-abomasnow": {
		typ: "Water", name: "Mega Abomasnow ex Battle Deck", star: "me01-036",
		list: append([]bdEntry{
			{"me01-035", 4}, // Snover
			{"me01-036", 3}, // Mega Abomasnow ex
			{"me01-037", 3}, // Clauncher
			{"me01-038", 3}, // Clawitzer
			{"me01-034", 3}, // Kyogre
			energy("sve-003", 16),
		}, trainerCore("me01-129", false)...), // Surfing Beach
	},
	"lightning-manectric": {
		typ: "Lightning", name: "Mega Manectric ex Battle Deck", star: "me01-050",
		list: append([]bdEntry{
			{"me01-049", 4}, // Electrike
			{"me01-050", 3}, // Mega Manectric ex
			{"me01-048", 3}, // Raikou
			{"me01-045", 3}, // Magnemite
			{"me01-046", 2}, // Magneton
			{"me01-047", 1}, // Magnezone
			energy("sve-004", 16),
		}, trainerCore("me01-122", false)...),
	},
	"psychic-gardevoir": {
		typ: "Psychic", name: "Mega Gardevoir ex Battle Deck", star: "me01-060",
		list: append([]bdEntry{
			{"me01-058", 4}, // Ralts
			{"me01-059", 2}, // Kirlia
			{"me01-060", 3}, // Mega Gardevoir ex
			{"me01-064", 3}, // Xerneas
			{"me01-065", 2}, // Greavard
			{"me01-066", 2}, // Houndstone
			energy("sve-005", 16),
		}, trainerCore("me01-122", true)...),
	},
	"fighting-lucario": {
		typ: "Fighting", name: "Mega Lucario ex Battle Deck", star: "me01-077",
		list: append([]bdEntry{
			{"me01-076", 4}, // Riolu
			{"me01-077", 3}, // Mega Lucario ex
			{"me01-080", 3}, // Marshadow
			{"me01-072", 3}, // Makuhita
			{"me01-073", 2}, // Hariyama
			{"me01-081", 1}, // Stonjourner
			energy("sve-006", 16),
		}, trainerCore("me01-122", false)...),
	},
	"darkness-absol": {
		typ: "Darkness", name: "Mega Absol ex Battle Deck", star: "me01-086",
		list: append([]bdEntry{
			{"me01-086", 3}, // Mega Absol ex
			{"me01-088", 4}, // Yveltal
			{"me01-089", 3}, // Nickit
			{"me01-090", 2}, // Thievul
			{"me01-087", 2}, // Spiritomb
			{"me01-091", 2}, // Shroodle
			energy("sve-007", 16),
		}, trainerCore("me01-127", false)...), // Risky Ruins
	},
	"metal-mawile": {
		typ: "Metal", name: "Mega Mawile ex Battle Deck", star: "me01-094",
		list: append([]bdEntry{
			{"me01-094", 3}, // Mega Mawile ex
			{"me01-095", 3}, // Dialga
			{"me01-096", 4}, // Tinkatink
			{"me01-097", 3}, // Tinkatuff
			{"me01-098", 3}, // Tinkaton
			energy("sve-008", 16),
		}, trainerCore("me01-122", true)...),
	},
	"dragon-latias": {
		typ: "Dragon", name: "Mega Latias ex Battle Deck", star: "me01-100",
		// Mega Latias ex ataca com Fire+Psychic+Colorless → energia dividida.
		list: append([]bdEntry{
			{"me01-100", 4}, // Mega Latias ex
			{"me01-101", 4}, // Latios
			{"me01-111", 4}, // Stufful
			{"me01-112", 2}, // Bewear
			{"me01-106", 2}, // Miltank
			energy("sve-002", 8), // Fire
			energy("sve-005", 8), // Psychic
		}, trainerCore("me01-122", false)...),
	},
	"colorless-kangaskhan": {
		typ: "Colorless", name: "Mega Kangaskhan ex Battle Deck", star: "me01-104",
		// Custos Colorless aceitam qualquer energia; Fighting é a escolha fixa.
		list: append([]bdEntry{
			{"me01-104", 3}, // Mega Kangaskhan ex
			{"me01-111", 4}, // Stufful
			{"me01-112", 3}, // Bewear
			{"me01-106", 3}, // Miltank
			{"me01-105", 3}, // Delibird
			energy("sve-006", 16),
		}, trainerCore("me01-122", false)...),
	},

	// ---- segundos arquétipos por tipo (sem Mega ex, linhas de evolução) ----
	"grass-meganium": {
		typ: "Grass", name: "Meganium Battle Deck", star: "me01-010",
		list: append([]bdEntry{
			{"me01-008", 4}, // Chikorita
			{"me01-009", 3}, // Bayleef
			{"me01-010", 3}, // Meganium
			{"me01-011", 3}, // Shuckle
			{"me01-018", 3}, // Dhelmise
			energy("sve-001", 16),
		}, trainerCore("me01-117", true)...),
	},
	"fire-cinderace": {
		typ: "Fire", name: "Cinderace Battle Deck", star: "me01-028",
		list: append([]bdEntry{
			{"me01-026", 4}, // Scorbunny
			{"me01-027", 3}, // Raboot
			{"me01-028", 3}, // Cinderace
			{"me01-029", 3}, // Sizzlipede
			{"me01-030", 2}, // Centiskorch
			{"me01-031", 1}, // Chi-Yu
			energy("sve-002", 16),
		}, trainerCore("me01-122", true)...),
	},
	"water-kyogre": {
		typ: "Water", name: "Kyogre Battle Deck", star: "me01-034",
		list: append([]bdEntry{
			{"me01-034", 3}, // Kyogre
			{"me01-039", 4}, // Sobble
			{"me01-040", 2}, // Drizzile
			{"me01-041", 2}, // Inteleon
			{"me01-044", 3}, // Eiscue
			{"me01-032", 2}, // Mantine
			energy("sve-003", 16),
		}, trainerCore("me01-129", true)...),
	},
	"lightning-magnezone": {
		typ: "Lightning", name: "Magnezone Battle Deck", star: "me01-047",
		list: append([]bdEntry{
			{"me01-045", 4}, // Magnemite
			{"me01-046", 3}, // Magneton
			{"me01-047", 3}, // Magnezone
			{"me01-048", 3}, // Raikou
			{"me01-051", 3}, // Pachirisu
			energy("sve-004", 16),
		}, trainerCore("me01-122", true)...),
	},
	"psychic-houndstone": {
		typ: "Psychic", name: "Houndstone Battle Deck", star: "me01-066",
		list: append([]bdEntry{
			{"me01-065", 4}, // Greavard
			{"me01-066", 3}, // Houndstone
			{"me01-064", 3}, // Xerneas
			{"me01-057", 3}, // Jynx
			{"me01-062", 2}, // Spoink
			{"me01-063", 1}, // Grumpig
			energy("sve-005", 16),
		}, trainerCore("me01-122", false)...),
	},
	"fighting-garganacl": {
		typ: "Fighting", name: "Garganacl Battle Deck", star: "me01-084",
		list: append([]bdEntry{
			{"me01-082", 4}, // Nacli
			{"me01-083", 3}, // Naclstack
			{"me01-084", 3}, // Garganacl
			{"me01-072", 3}, // Makuhita
			{"me01-073", 2}, // Hariyama
			{"me01-080", 1}, // Marshadow
			energy("sve-006", 16),
		}, trainerCore("me01-122", true)...),
	},
	"darkness-yveltal": {
		typ: "Darkness", name: "Yveltal Battle Deck", star: "me01-088",
		list: append([]bdEntry{
			{"me01-088", 4}, // Yveltal
			{"me01-091", 4}, // Shroodle
			{"me01-092", 3}, // Grafaiai
			{"me01-087", 3}, // Spiritomb
			{"me01-089", 2}, // Nickit
			energy("sve-007", 16),
		}, trainerCore("me01-127", false)...),
	},
	"metal-steelix": {
		typ: "Metal", name: "Steelix Battle Deck", star: "me01-093",
		list: append([]bdEntry{
			{"me01-070", 4}, // Onix
			{"me01-093", 3}, // Steelix
			{"me01-095", 3}, // Dialga
			{"me01-067", 3}, // Gimmighoul
			{"me01-099", 3}, // Gholdengo
			energy("sve-008", 16),
		}, trainerCore("me01-122", false)...),
	},
	"colorless-bewear": {
		typ: "Colorless", name: "Bewear Battle Deck", star: "me01-112",
		list: append([]bdEntry{
			{"me01-111", 4}, // Stufful
			{"me01-112", 3}, // Bewear
			{"me01-109", 4}, // Yungoos
			{"me01-110", 3}, // Gumshoos
			{"me01-105", 2}, // Delibird
			energy("sve-006", 16),
		}, trainerCore("me01-122", false)...),
	},
}

// BattleDecks lista os decks disponíveis, ordenados por tipo e nome.
func BattleDecks() []BattleDeckInfo {
	out := make([]BattleDeckInfo, 0, len(battleDecks))
	for id, bd := range battleDecks {
		out = append(out, BattleDeckInfo{ID: id, Type: bd.typ, Name: bd.name, Star: bd.star})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// BattleDeck devolve o Battle Deck fixo pelo ID (slug), validado contra a base.
// Aceita também um tipo (ex.: "Fire"): devolve o primeiro deck daquele tipo.
func BattleDeck(store *cards.Store, key string) (*deck.Deck, error) {
	bd, ok := battleDecks[key]
	if !ok {
		for _, info := range BattleDecks() {
			if info.Type == key {
				bd, ok = battleDecks[info.ID], true
				break
			}
		}
	}
	if !ok {
		return nil, fmt.Errorf("sem Battle Deck %q", key)
	}
	d := deck.New()
	for _, e := range bd.list {
		d.Add(e.id, e.n)
	}
	if errs := d.Validate(store); len(errs) > 0 {
		return nil, fmt.Errorf("Battle Deck %s inválido: %v", bd.name, errs)
	}
	return d, nil
}
