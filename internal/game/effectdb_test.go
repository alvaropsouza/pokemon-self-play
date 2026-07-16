package game

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEffectDBOverlay(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "effects.json")
	if err := os.WriteFile(path, []byte(`{
		"Search your deck for weirdness.": {"ops":[{"kind":"draw","n":2}], "source":"llm"},
		"Truly manual effect.": {"manual":true, "source":"llm"},
		"Broken entry.": {"ops":[{"kind":"nonsense"}], "source":"llm"}
	}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := LoadEffectDB(path); err != nil {
		t.Fatal(err)
	}
	defer func() {
		effectDB = map[string]EffectEntry{}
		effectCache = map[string]CompiledEffect{}
	}()

	// Entrada válida do banco resolve texto que o regex não cobre.
	ce := CompileEffect("Search your deck for weirdness.")
	if ce.Manual || len(ce.Ops) != 1 || ce.Ops[0].Kind != OpDraw || ce.Ops[0].N != 2 {
		t.Errorf("entrada do banco não aplicada: %+v", ce)
	}
	// Entrada manual continua manual.
	if !CompileEffect("Truly manual effect.").Manual {
		t.Error("entrada manual do banco deveria continuar Manual")
	}
	// Entrada com op inválida é rejeitada → Manual.
	if !CompileEffect("Broken entry.").Manual {
		t.Error("entrada inválida do banco deveria cair em Manual")
	}
	// Regex continua tendo prioridade sobre o banco.
	if CompileEffect("Draw 3 cards.").Manual {
		t.Error("regex deveria resolver independente do banco")
	}
}

func TestLoadEffectDBMissingFile(t *testing.T) {
	if err := LoadEffectDB(filepath.Join(t.TempDir(), "nope.json")); err != nil {
		t.Fatalf("arquivo ausente não deveria ser erro: %v", err)
	}
}

func TestValidateOps(t *testing.T) {
	if err := ValidateOps([]Op{{Kind: OpDraw, N: 3}}); err != nil {
		t.Errorf("ops válidas rejeitadas: %v", err)
	}
	for name, ops := range map[string][]Op{
		"vazia":             {},
		"kind desconhecido": {{Kind: "explode"}},
		"cond inválida":     {{Kind: OpStatus, Cond: "dizzy"}},
		"n fora de faixa":   {{Kind: OpDraw, N: 9999}},
	} {
		if ValidateOps(ops) == nil {
			t.Errorf("%s: deveria falhar", name)
		}
	}
}
