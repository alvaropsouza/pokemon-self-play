package cards

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// TCGdexClient acessa a API pública do TCGdex (https://tcgdex.dev),
// que fornece dados de cartas em EN e PT-BR com IDs compartilhados.
type TCGdexClient struct {
	BaseURL string
	HTTP    *http.Client
}

func NewTCGdexClient() *TCGdexClient {
	return &TCGdexClient{
		BaseURL: "https://api.tcgdex.net/v2",
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

// tcgdexCard espelha o JSON de /v2/{lang}/cards/{id}.
type tcgdexCard struct {
	ID             string   `json:"id"`
	LocalID        string   `json:"localId"`
	Name           string   `json:"name"`
	Category       string   `json:"category"`
	Image          string   `json:"image"`
	Rarity         string   `json:"rarity"`
	RegulationMark string   `json:"regulationMark"`
	HP             int      `json:"hp"`
	Types          []string `json:"types"`
	Stage          string   `json:"stage"`
	EvolveFrom     string   `json:"evolveFrom"`
	Retreat        int      `json:"retreat"`
	TrainerType    string   `json:"trainerType"`
	EnergyType     string   `json:"energyType"`
	Effect         string   `json:"effect"`
	Set            struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"set"`
	Attacks []struct {
		Name   string          `json:"name"`
		Effect string          `json:"effect"`
		Cost   []string        `json:"cost"`
		Damage json.RawMessage `json:"damage"`
	} `json:"attacks"`
	Abilities []struct {
		Type   string `json:"type"`
		Name   string `json:"name"`
		Effect string `json:"effect"`
	} `json:"abilities"`
	Weaknesses  []TypeValue `json:"weaknesses"`
	Resistances []TypeValue `json:"resistances"`
}

// tcgdexSet espelha o JSON de /v2/{lang}/sets/{id} (apenas o necessário).
type tcgdexSet struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Cards []struct {
		ID string `json:"id"`
	} `json:"cards"`
}

func (c *TCGdexClient) get(path string, out any) error {
	url := c.BaseURL + path
	resp, err := c.HTTP.Get(url)
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return errNotFound
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("GET %s: status %d: %s", url, resp.StatusCode, body)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", url, err)
	}
	return nil
}

var errNotFound = fmt.Errorf("not found")

// SetCardIDs retorna os IDs de todas as cartas de um set.
func (c *TCGdexClient) SetCardIDs(setID string) ([]string, error) {
	var s tcgdexSet
	if err := c.get("/en/sets/"+setID, &s); err != nil {
		return nil, err
	}
	ids := make([]string, len(s.Cards))
	for i, card := range s.Cards {
		ids[i] = card.ID
	}
	return ids, nil
}

// FetchCard busca a carta em EN e PT e devolve o registro canônico mesclado.
// A versão PT pode não existir (sets sem localização); nesse caso os campos
// PT ficam vazios.
func (c *TCGdexClient) FetchCard(id string) (*Card, error) {
	var en tcgdexCard
	if err := c.get("/en/cards/"+id, &en); err != nil {
		return nil, fmt.Errorf("card %s (en): %w", id, err)
	}
	var pt tcgdexCard
	hasPT := true
	if err := c.get("/pt/cards/"+id, &pt); err != nil {
		if err != errNotFound {
			return nil, fmt.Errorf("card %s (pt): %w", id, err)
		}
		hasPT = false
	}

	card := &Card{
		ID:             en.ID,
		SetID:          en.Set.ID,
		SetName:        Localized{EN: en.Set.Name},
		LocalID:        en.LocalID,
		Name:           Localized{EN: en.Name},
		Category:       Category(en.Category),
		RegulationMark: en.RegulationMark,
		Rarity:         en.Rarity,
		Image:          Localized{EN: en.Image},
		HP:             en.HP,
		Types:          en.Types,
		Stage:          en.Stage,
		EvolveFrom:     Localized{EN: en.EvolveFrom},
		Retreat:        en.Retreat,
		TrainerType:    en.TrainerType,
		EnergyType:     en.EnergyType,
		Effect:         Localized{EN: en.Effect},
		Weaknesses:     en.Weaknesses,
		Resistances:    en.Resistances,
	}
	for _, a := range en.Attacks {
		card.Attacks = append(card.Attacks, Attack{
			Name:   Localized{EN: a.Name},
			Effect: Localized{EN: a.Effect},
			Cost:   a.Cost,
			Damage: rawDamage(a.Damage),
		})
	}
	for _, ab := range en.Abilities {
		card.Abilities = append(card.Abilities, Ability{
			Type:   ab.Type,
			Name:   Localized{EN: ab.Name},
			Effect: Localized{EN: ab.Effect},
		})
	}

	if hasPT {
		card.SetName.PT = pt.Set.Name
		card.Name.PT = pt.Name
		card.Image.PT = pt.Image
		card.EvolveFrom.PT = pt.EvolveFrom
		card.Effect.PT = pt.Effect
		// Ataques e habilidades pareiam por índice: mesma carta, mesma ordem.
		for i := range card.Attacks {
			if i < len(pt.Attacks) {
				card.Attacks[i].Name.PT = pt.Attacks[i].Name
				card.Attacks[i].Effect.PT = pt.Attacks[i].Effect
			}
		}
		for i := range card.Abilities {
			if i < len(pt.Abilities) {
				card.Abilities[i].Name.PT = pt.Abilities[i].Name
				card.Abilities[i].Effect.PT = pt.Abilities[i].Effect
			}
		}
	}
	return card, nil
}

// rawDamage normaliza o campo damage, que a API devolve como número (30)
// ou string ("30+", "20×").
func rawDamage(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	return strings.Trim(string(raw), `"`)
}
