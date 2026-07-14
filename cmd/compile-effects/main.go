// cmd/compile-effects mantém o banco de efeitos (data/effects.json).
//
// Varre data/cards.json, encontra textos de efeito que o compilador regex não
// resolve e que ainda não estão no banco, e:
//   - sem ANTHROPIC_API_KEY: só lista os pendentes (para preencher à mão);
//   - com a chave: traduz cada texto em ops via Claude API, valida
//     estruturalmente e grava no banco. Entradas verified:true (humanas)
//     nunca são sobrescritas.
//
// O jogo nunca chama a API: o banco é compilado offline, 1× por texto novo.
//
//	go run ./cmd/compile-effects [-dry-run]
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/alvaropsouza/pokemon-self-play/internal/cards"
	"github.com/alvaropsouza/pokemon-self-play/internal/game"
)

const (
	apiURL    = "https://api.anthropic.com/v1/messages"
	model     = "claude-opus-4-8"
	batchSize = 15
)

func main() {
	cardsPath := flag.String("cards", "data/cards.json", "base de cartas")
	dbPath := flag.String("db", "data/effects.json", "banco de efeitos")
	dryRun := flag.Bool("dry-run", false, "só lista pendentes, não chama a API")
	flag.Parse()

	store, err := cards.Load(*cardsPath)
	if err != nil {
		log.Fatal(err)
	}
	if err := game.LoadEffectDB(*dbPath); err != nil {
		log.Fatal(err)
	}
	db := map[string]game.EffectEntry{}
	for k, v := range game.EffectDB() {
		db[k] = v
	}

	pending := pendingTexts(store, db)
	fmt.Printf("banco: %d entradas | pendentes: %d\n", len(db), len(pending))
	if len(pending) == 0 {
		return
	}

	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" || *dryRun {
		for _, t := range pending {
			fmt.Printf("  - %s\n", t)
		}
		if key == "" {
			fmt.Println("\nANTHROPIC_API_KEY ausente — defina para compilar via Claude API.")
		}
		return
	}

	added := 0
	for start := 0; start < len(pending); start += batchSize {
		end := min(start+batchSize, len(pending))
		batch := pending[start:end]
		results, err := compileBatch(key, batch)
		if err != nil {
			log.Fatalf("lote %d-%d: %v", start, end, err)
		}
		for text, entry := range results {
			if existing, ok := db[text]; ok && existing.Verified {
				continue
			}
			if !entry.Manual {
				if err := game.ValidateOps(entry.Ops); err != nil {
					fmt.Printf("  rejeitado (%v): %s\n", err, text)
					entry = game.EffectEntry{Manual: true}
				}
			}
			entry.Source = "llm"
			db[text] = entry
			added++
		}
		fmt.Printf("lote %d-%d ok\n", start+1, end)
	}

	if err := saveDB(*dbPath, db); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("gravado %s: %d entradas (+%d)\n", *dbPath, len(db), added)
}

// pendingTexts coleta textos de efeito Standard que o regex marca Manual e o
// banco ainda não conhece, ordenados para saída determinística.
func pendingTexts(store *cards.Store, db map[string]game.EffectEntry) []string {
	seen := map[string]bool{}
	var out []string
	add := func(text string) {
		if text == "" || seen[text] {
			return
		}
		seen[text] = true
		if _, ok := db[text]; ok {
			return
		}
		if game.CompileEffect(text).Manual {
			out = append(out, text)
		}
	}
	for _, c := range store.Cards {
		if !c.StandardLegal() {
			continue
		}
		if c.Category == cards.CategoryTrainer {
			add(c.Effect.EN)
		}
		for _, atk := range c.Attacks {
			add(atk.Effect.EN)
		}
	}
	sort.Strings(out)
	return out
}

const systemPrompt = `You translate Pokémon TCG card effect text into a small JSON IR for a rules engine.

Available ops (JSON objects):
  {"kind":"draw","n":N}                      draw N cards (the player of the effect)
  {"kind":"draw_until","n":N}                draw until hand has N cards
  {"kind":"discard_hand"}                    discard your whole hand
  {"kind":"shuffle_hand_both","tools":bool}  both players shuffle hand into deck (tools:true if Pokémon Tools go back too)
  {"kind":"draw_both","n":N}                 both players draw N
  {"kind":"draw_per_prize_both"}             both players draw = their remaining prize cards
  {"kind":"damage_opp_bench","n":N}          N damage to each of opponent's benched Pokémon
  {"kind":"damage_self_bench","n":N}         N damage to each of your benched Pokémon
  {"kind":"heal_self","n":N}                 heal N from the attacking Pokémon
  {"kind":"discard_self_energy","n":N}       discard N energies from the attacker (n:-1 = all)
  {"kind":"discard_opp_energy","n":N}        discard N energies from opponent's Active (n:-1 = all); use with "flip":true for coin-flip discard
  {"kind":"scale_per_energy_self","n":N}     attack does +N per energy attached to attacker
  {"kind":"scale_per_energy_opp","n":N}      attack does +N per energy on opponent's Active
  {"kind":"damage_self","n":N}               put N damage on own Active (recoil / "this Pokémon does N damage to itself")
  {"kind":"status","cond":C,"onSelf":bool,"flip":bool}  special condition on the Active (cond: asleep|confused|paralyzed|poisoned|burned; onSelf:true = attacker's own Active; flip:true = only on heads)
  {"kind":"switch_self"}                     switch own Active with a chosen Benched Pokémon (player picks; clears conditions on the retiring Active)
  {"kind":"switch_opp"}                      the attacker chooses 1 of opponent's Benched Pokémon and forces it to become Active ("switch in", "gust" effects; clears conditions on retiring opp Active)
  {"kind":"search","n":N,"dest":D,"find":[F...]}        search your deck for up to N cards matching ANY F, put into D ("hand"|"bench"); F = {"category":"Pokemon"|"Energy","stage":"Basic"|omit,"type":"Fire"|omit}. Energy = Basic Energy only. Deck is shuffled after.
  {"kind":"shuffle_deck"}                               shuffle your own deck

Any op may carry "flip":true meaning it only happens on a heads coin flip.

Rules:
- An effect is expressible ONLY if EVERY clause of the text maps exactly to the ops above. Partial coverage is forbidden.
- "search" only for deck→hand/bench with filters expressible as category/stage/type. Searches with HP conditions, name conditions, evolution/ex filters, or other zones → {"manual":true}.
- Effects with durations ("during your next turn...", "until end of your next turn"), conditions on game state, choices of damage amounts, or anything not listed → {"manual":true}.
- Output ONLY a JSON object mapping each input text verbatim to either {"ops":[...]} or {"manual":true}. No prose, no markdown fences.`

type apiRequest struct {
	Model     string       `json:"model"`
	MaxTokens int          `json:"max_tokens"`
	System    string       `json:"system"`
	Messages  []apiMessage `json:"messages"`
}

type apiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type apiResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Error      *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// compileBatch envia um lote de textos ao Claude e devolve texto → entrada.
func compileBatch(key string, texts []string) (map[string]game.EffectEntry, error) {
	input, _ := json.MarshalIndent(texts, "", "  ")
	reqBody, err := json.Marshal(apiRequest{
		Model:     model,
		MaxTokens: 16000,
		System:    systemPrompt,
		Messages:  []apiMessage{{Role: "user", Content: string(input)}},
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", key)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chamando Claude API: %w", err)
	}
	defer resp.Body.Close()

	var out apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decodificando resposta: %w", err)
	}
	if out.Error != nil {
		return nil, fmt.Errorf("API %s: %s", out.Error.Type, out.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API HTTP %d", resp.StatusCode)
	}
	if out.StopReason == "refusal" {
		return nil, fmt.Errorf("API recusou a requisição (stop_reason: refusal)")
	}
	var text strings.Builder
	for _, block := range out.Content {
		if block.Type == "text" {
			text.WriteString(block.Text)
		}
	}

	results := map[string]game.EffectEntry{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(text.String())), &results); err != nil {
		return nil, fmt.Errorf("resposta não é o JSON esperado: %w", err)
	}
	for t := range results {
		if !slices.Contains(texts, t) {
			return nil, fmt.Errorf("resposta contém texto não solicitado: %q", t)
		}
	}
	return results, nil
}

// saveDB grava o banco com chaves ordenadas (diff estável no git).
func saveDB(path string, db map[string]game.EffectEntry) error {
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
