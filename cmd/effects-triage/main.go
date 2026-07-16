package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/alvaropsouza/pokemon-self-play/internal/cards"
	"github.com/alvaropsouza/pokemon-self-play/internal/game"

	_ "modernc.org/sqlite"
)

const schemaEffects = `CREATE TABLE IF NOT EXISTS pending_effects (
	effect_text TEXT PRIMARY KEY,
	context     TEXT NOT NULL,
	card_ids    TEXT NOT NULL,
	card_count  INTEGER NOT NULL,
	sample_name TEXT NOT NULL,
	status      TEXT NOT NULL DEFAULT 'todo' CHECK (status IN ('todo','done','manual')),
	ops         TEXT,
	note        TEXT
)`

const schemaUI = `CREATE TABLE IF NOT EXISTS pending_ui (
	op_kind     TEXT PRIMARY KEY,
	description TEXT NOT NULL,
	suggestion  TEXT NOT NULL,
	files       TEXT NOT NULL,
	status      TEXT NOT NULL DEFAULT 'todo' CHECK (status IN ('todo','done')),
	note        TEXT
)`

type uiEntry struct {
	description string
	suggestion  string
	files       string
}

var uiRegistry = map[string]uiEntry{
	string(game.OpDraw):              {description: "draw N cards", suggestion: "covered: flyFromDeck in HandTray", files: "web/src/drawfx.ts", },
	string(game.OpSearch):            {description: "search deck, put into hand/bench", suggestion: "covered: PendingChoice overlay", files: "web/src/App.tsx"},
	string(game.OpSwitchSelf):        {description: "switch own active with bench", suggestion: "covered: PendingChoice overlay", files: "web/src/App.tsx"},
	string(game.OpSwitchOpp):         {description: "force opponent bench to active", suggestion: "covered: PendingChoice overlay", files: "web/src/App.tsx"},
	string(game.OpDiscardHand):       {description: "discard full hand", suggestion: "covered: PendingChoice overlay", files: "web/src/App.tsx"},
	string(game.OpDiscardFromHand):   {description: "discard N cards from hand", suggestion: "covered: PendingChoice overlay", files: "web/src/App.tsx"},
	string(game.OpStatus):            {description: "apply special condition badge", suggestion: "covered (static badge); missing: pop-in animation when condition is applied", files: "web/src/components/Card.tsx"},
	string(game.OpShuffleDeck):       {description: "shuffle own deck", suggestion: "covered: shakeDeck via GameState.events in effectsfx.ts", files: "web/src/effectsfx.ts"},
	string(game.OpDrawUntil):         {description: "draw until hand has N cards", suggestion: "same flyFromDeck as OpDraw, fired multiple times until hand count reaches N", files: "web/src/drawfx.ts,web/src/App.tsx"},
	string(game.OpDrawOrMore):        {description: "draw N, or Alt if player has exactly M prizes", suggestion: "same as OpDraw; toast should clarify which variant was used", files: "web/src/drawfx.ts,web/src/App.tsx"},
	string(game.OpDrawBoth):          {description: "both players draw N cards", suggestion: "flyFromDeck fired for both sides in sequence", files: "web/src/drawfx.ts,web/src/App.tsx"},
	string(game.OpDrawPerPrizeBoth):  {description: "both players draw 1 card per remaining prize", suggestion: "same as OpDrawBoth, N derived from state.players[i].prizeCount", files: "web/src/drawfx.ts,web/src/App.tsx"},
	string(game.OpShuffleHandBoth):   {description: "both players shuffle hand into deck", suggestion: "fly hand cards back to deck (reverse of flyFromDeck) for both sides + shakeDeck", files: "web/src/drawfx.ts,web/src/effectsfx.ts"},
	string(game.OpShuffleHandSelf):   {description: "own player shuffles hand into deck", suggestion: "fly hand cards back to deck (reverse of flyFromDeck) + shakeDeck", files: "web/src/drawfx.ts,web/src/effectsfx.ts"},
	string(game.OpDamageOppBench):    {description: "place N damage on each opponent bench slot", suggestion: "red flash on each affected bench Card (no shake; bench doesn't attack)", files: "web/src/components/Card.tsx,web/src/components/Mat.tsx"},
	string(game.OpDamageSelfBench):   {description: "place N damage on each own bench slot", suggestion: "same red flash as OpDamageOppBench but on own bench", files: "web/src/components/Card.tsx,web/src/components/Mat.tsx"},
	string(game.OpHealSelf):          {description: "remove N damage from own active", suggestion: "green flash + HP counter rising on own active Card", files: "web/src/components/Card.tsx"},
	string(game.OpDiscardSelfEnergy): {description: "discard N energies from own active (n=-1: all)", suggestion: "energy chip fade-out on own active Card", files: "web/src/components/Card.tsx"},
	string(game.OpDiscardOppEnergy):  {description: "discard N energies from opponent active (n=-1: all)", suggestion: "energy chip fade-out on opponent active Card", files: "web/src/components/Card.tsx"},
	string(game.OpScalePerEnergySelf): {description: "attack does +N per energy on own active", suggestion: "no dedicated UI needed (final damage already animates); optional: scale badge on attack card", files: "web/src/components/ActionBar.tsx"},
	string(game.OpScalePerEnergyOpp): {description: "attack does +N per energy on opponent active", suggestion: "no dedicated UI needed (final damage already animates)", files: "web/src/components/ActionBar.tsx"},
	string(game.OpDamageSelf):        {description: "place N damage on own active (recoil/confusion)", suggestion: "red flash + shake on own active Card (same as damage received, own side)", files: "web/src/components/Card.tsx"},
}

var coveredOps = map[string]bool{
	string(game.OpDraw):              true,
	string(game.OpDrawUntil):         true,
	string(game.OpDrawOrMore):        true,
	string(game.OpDrawBoth):          true,
	string(game.OpDrawPerPrizeBoth):  true,
	string(game.OpSearch):            true,
	string(game.OpSwitchSelf):        true,
	string(game.OpSwitchOpp):         true,
	string(game.OpDiscardHand):       true,
	string(game.OpDiscardFromHand):   true,
	string(game.OpDiscardSelfEnergy): true,
	string(game.OpDiscardOppEnergy):  true,
	string(game.OpStatus):            true,
	string(game.OpShuffleDeck):       true,
	string(game.OpShuffleHandSelf):   true,
	string(game.OpShuffleHandBoth):   true,
	string(game.OpDamageSelf):        true,
	string(game.OpDamageOppBench):    true,
	string(game.OpDamageSelfBench):   true,
	string(game.OpHealSelf):          true,
	string(game.OpScalePerEnergySelf): true,
	string(game.OpScalePerEnergyOpp):  true,
}

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
	for _, s := range []string{schemaEffects, schemaUI} {
		if _, err := db.Exec(s); err != nil {
			log.Fatal(err)
		}
	}

	switch flag.Arg(0) {
	case "", "scan":
		err = scan(db, store)
	case "export":
		err = export(db, *effectsPath)
	case "create-issues":
		err = createIssues(db)
	case "mark-manual":
		err = markManual(db)
	default:
		fmt.Fprintln(os.Stderr, "uso: effects-triage [scan|export|create-issues|mark-manual]")
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
	if err := scanEffects(db, store); err != nil {
		return err
	}
	return scanUI(db)
}

func scanEffects(db *sql.DB, store *cards.Store) error {
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
	_ = rows.Close()
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
	fmt.Printf("[motor] %d efeitos pendentes | %d todo / %d done / %d manual\n", len(pending), todo, done, manual)
	return nil
}

func scanUI(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var ops []string
	for op := range uiRegistry {
		ops = append(ops, op)
	}
	sort.Strings(ops)

	for _, op := range ops {
		e := uiRegistry[op]
		initialStatus := "todo"
		if coveredOps[op] {
			initialStatus = "done"
		}
		_, err := tx.Exec(`INSERT INTO pending_ui (op_kind, description, suggestion, files, status)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(op_kind) DO UPDATE SET
				description = excluded.description,
				suggestion  = excluded.suggestion,
				files       = excluded.files,
				status      = CASE WHEN excluded.status = 'done' THEN 'done' ELSE pending_ui.status END`,
			op, e.description, e.suggestion, e.files, initialStatus)
		if err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	var todo, done int
	if err := db.QueryRow(`SELECT
		COUNT(*) FILTER (WHERE status = 'todo'),
		COUNT(*) FILTER (WHERE status = 'done')
		FROM pending_ui`).Scan(&todo, &done); err != nil {
		return err
	}
	fmt.Printf("[ui]    %d OpKinds pendentes de animação | %d done / %d todo\n", todo, done, todo)
	fmt.Println()
	if todo > 0 {
		rows, err := db.Query(`SELECT op_kind, description FROM pending_ui WHERE status = 'todo' ORDER BY op_kind`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var op, desc string
			if err := rows.Scan(&op, &desc); err != nil {
				return err
			}
			fmt.Printf("  %-28s %s\n", op, desc)
		}
		return rows.Err()
	}
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
	fmt.Printf("gravado %s: %d entradas (+%d da triagem) — rode scan para limpar o banco\n", effectsPath, len(effects), applied)
	return nil
}

func createIssues(db *sql.DB) error {
	created := 0

	rows, err := db.Query(`
		SELECT effect_text, context, sample_name, card_count, card_ids
		FROM pending_effects
		WHERE status = 'todo'
		ORDER BY card_count DESC`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var text, ctx, sample, cardIDs string
		var count int
		if err := rows.Scan(&text, &ctx, &sample, &count, &cardIDs); err != nil {
			return err
		}
		title := fmt.Sprintf("engine: implement effect (%s) — %d card(s) — sample: %s", ctx, count, sample)
		body := fmt.Sprintf("## Effect text\n\n```\n%s\n```\n\n## Context\n\n`%s` — %d card(s) affected\n\n## Cards\n\n`%s`\n\n## How to fix\n\n1. Open `data/triage.db` (DB Browser for SQLite)\n2. Find row with this `effect_text`\n3. Set `status = 'done'` and fill `ops` with JSON array of ops (see `internal/game/effects.go` for OpKind list), or `status = 'manual'` if not expressible\n4. Run `task triage -- export`\n\n<!-- auto-generated by cmd/effects-triage -->",
			text, ctx, count, cardIDs)
		if err := ghCreateIssue(title, body, "engine-effect,triage"); err != nil {
			return fmt.Errorf("criando issue para %q: %w", sample, err)
		}
		created++
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	uiRows, err := db.Query(`
		SELECT op_kind, description, suggestion, files
		FROM pending_ui
		WHERE status = 'todo'
		ORDER BY op_kind`)
	if err != nil {
		return err
	}
	for uiRows.Next() {
		var op, desc, suggestion, files string
		if err := uiRows.Scan(&op, &desc, &suggestion, &files); err != nil {
			return err
		}
		title := fmt.Sprintf("ui: animate `%s` — %s", op, desc)
		fileList := strings.Join(strings.Split(files, ","), "\n- ")
		body := fmt.Sprintf("## OpKind\n\n`%s`\n\n## What it does\n\n%s\n\n## Suggested UI\n\n%s\n\n## Files to edit\n\n- %s\n\n## How to close\n\n1. Implement the animation/interaction in the files above\n2. Update `coveredOps` map in `cmd/effects-triage/main.go` to mark `\"%s\"` as covered\n3. Run `task triage -- scan` to sync the db\n\n<!-- auto-generated by cmd/effects-triage -->",
			op, desc, suggestion, fileList, op)
		if err := ghCreateIssue(title, body, "ui,triage"); err != nil {
			return fmt.Errorf("criando issue para %q: %w", op, err)
		}
		created++
	}
	_ = uiRows.Close()
	if err := uiRows.Err(); err != nil {
		return err
	}

	fmt.Printf("%d issues criadas no GitHub\n", created)
	return nil
}

func markManual(db *sql.DB) error {
	res, err := db.Exec(`UPDATE pending_effects SET status = 'manual' WHERE status = 'todo'`)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	fmt.Printf("%d efeitos marcados como manual\n", n)
	return nil
}

func ghCreateIssue(title, body, labels string) error {
	cmd := exec.Command("gh", "issue", "create",
		"--title", title,
		"--body", body,
		"--label", labels,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
