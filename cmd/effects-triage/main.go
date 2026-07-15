package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/alvaropsouza/pokemon-self-play/internal/cards"
	"github.com/alvaropsouza/pokemon-self-play/internal/game"

	_ "modernc.org/sqlite"
)

const schema = `CREATE TABLE IF NOT EXISTS pending_effects (
	effect_text TEXT PRIMARY KEY,
	context     TEXT NOT NULL,
	card_ids    TEXT NOT NULL,
	card_count  INTEGER NOT NULL,
	sample_name TEXT NOT NULL,
	status      TEXT NOT NULL DEFAULT 'todo' CHECK (status IN ('todo','done','manual')),
	ops         TEXT,
	note        TEXT
)`

func main() {
	cardsPath := flag.String("cards", "data/cards.json", "base de cartas")
	effectsPath := flag.String("effects", "data/effects.json", "banco de efeitos do jogo")
	dbPath := flag.String("db", "data/triage.db", "banco SQLite de triagem")
	flag.Parse()

	store, err := cards.Load(*cardsPath)
	if err != nil {
		log.Fatal(err)
	}
	if err := game.LoadEffectDB(*effectsPath); err != nil {
		log.Fatal(err)
	}
	db, err := sql.Open("sqlite", *dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(schema); err != nil {
		log.Fatal(err)
	}

	switch flag.Arg(0) {
	case "scan":
		err = scan(db, store)
	case "export":
		err = export(db, *effectsPath)
	default:
		fmt.Fprintln(os.Stderr, "uso: effects-triage <scan|export>")
		os.Exit(2)
	}
	if err != nil {
		log.Fatal(err)
	}
}

type pendingRow struct {
	context string
	ids     []string
	sample  string
}

func scan(db *sql.DB, store *cards.Store) error {
	pending := map[string]*pendingRow{}
	add := func(text, context string, c *cards.Card) {
		if text == "" || !game.CompileEffect(text).Manual || game.HasTrigger(c.ID) {
			return
		}
		row := pending[text]
		if row == nil {
			row = &pendingRow{context: context, sample: c.Name.EN}
			pending[text] = row
		}
		row.ids = append(row.ids, c.ID)
	}
	for _, c := range store.Cards {
		if !c.StandardLegal() {
			continue
		}
		if c.Category == cards.CategoryTrainer || c.Category == cards.CategoryEnergy {
			add(c.Effect.EN, strings.ToLower(string(c.Category)), c)
		}
		for _, atk := range c.Attacks {
			add(atk.Effect.EN, "attack", c)
		}
		for _, ab := range c.Abilities {
			add(ab.Effect.EN, "ability", c)
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for text, row := range pending {
		sort.Strings(row.ids)
		_, err := tx.Exec(`INSERT INTO pending_effects (effect_text, context, card_ids, card_count, sample_name)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(effect_text) DO UPDATE SET
				context = excluded.context, card_ids = excluded.card_ids,
				card_count = excluded.card_count, sample_name = excluded.sample_name`,
			text, row.context, strings.Join(row.ids, ","), len(row.ids), row.sample)
		if err != nil {
			return err
		}
	}

	rows, err := tx.Query(`SELECT effect_text FROM pending_effects`)
	if err != nil {
		return err
	}
	var stale []string
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			return err
		}
		if pending[text] == nil {
			stale = append(stale, text)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, text := range stale {
		if _, err := tx.Exec(`DELETE FROM pending_effects WHERE effect_text = ?`, text); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	var todo, done, manual int
	if err := db.QueryRow(`SELECT
		COUNT(*) FILTER (WHERE status = 'todo'),
		COUNT(*) FILTER (WHERE status = 'done'),
		COUNT(*) FILTER (WHERE status = 'manual')
		FROM pending_effects`).Scan(&todo, &done, &manual); err != nil {
		return err
	}
	fmt.Printf("pendentes: %d | resolvidos desde o último scan: %d\n", len(pending), len(stale))
	fmt.Printf("status: %d todo, %d done (exportar), %d manual (exportar)\n", todo, done, manual)
	return nil
}

func export(db *sql.DB, effectsPath string) error {
	rows, err := db.Query(`SELECT effect_text, status, ops FROM pending_effects WHERE status IN ('done', 'manual')`)
	if err != nil {
		return err
	}
	defer rows.Close()

	effects := map[string]game.EffectEntry{}
	for k, v := range game.EffectDB() {
		effects[k] = v
	}

	applied := 0
	for rows.Next() {
		var text, status string
		var opsJSON sql.NullString
		if err := rows.Scan(&text, &status, &opsJSON); err != nil {
			return err
		}
		entry := game.EffectEntry{Source: "human", Verified: true}
		if status == "manual" {
			entry.Manual = true
		} else {
			if !opsJSON.Valid || strings.TrimSpace(opsJSON.String) == "" {
				return fmt.Errorf("status done sem ops: %q", text)
			}
			if err := json.Unmarshal([]byte(opsJSON.String), &entry.Ops); err != nil {
				return fmt.Errorf("ops inválido em %q: %w", text, err)
			}
			if err := game.ValidateOps(entry.Ops); err != nil {
				return fmt.Errorf("ops rejeitado em %q: %w", text, err)
			}
		}
		effects[text] = entry
		applied++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if applied == 0 {
		fmt.Println("nada a exportar (nenhuma linha done/manual)")
		return nil
	}

	data, err := json.MarshalIndent(effects, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(effectsPath, append(data, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Printf("gravado %s: %d entradas (+%d da triagem) — rode `scan` para limpar o banco\n", effectsPath, len(effects), applied)
	return nil
}
