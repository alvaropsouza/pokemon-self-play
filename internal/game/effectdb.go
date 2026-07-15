package game

import (
	"encoding/json"
	"fmt"
	"os"
)

// Banco de efeitos: base de conhecimento persistente que mapeia texto EN de
// efeito → ops. Cresce via cmd/compile-effects (LLM valida offline, 1× por
// texto) ou edição humana. Em jogo é só leitura — zero rede, determinístico.

// EffectEntry é uma entrada do banco (data/effects.json).
type EffectEntry struct {
	Ops    []Op   `json:"ops,omitempty"`
	Manual bool   `json:"manual,omitempty"` // texto avaliado e inexprimível em ops → arbitragem manual
	Source string `json:"source,omitempty"` // "llm" ou "human"
	// Verified: entrada revisada por humano — cmd/compile-effects nunca sobrescreve.
	Verified bool `json:"verified,omitempty"`
}

// effectDB é o overlay consultado por CompileEffect. Chave: texto EN exato.
var effectDB = map[string]EffectEntry{}

var validKinds = map[OpKind]bool{
	OpDraw: true, OpDrawUntil: true, OpDrawOrMore: true, OpDiscardHand: true,
	OpShuffleHandBoth: true, OpShuffleHandSelf: true,
	OpDrawBoth: true, OpDrawPerPrizeBoth: true, OpDamageOppBench: true,
	OpDamageSelfBench: true, OpHealSelf: true, OpDiscardSelfEnergy: true,
	OpScalePerEnergySelf: true, OpScalePerEnergyOpp: true, OpStatus: true,
	OpSearch: true, OpShuffleDeck: true,
	OpSwitchSelf: true, OpSwitchOpp: true, OpDiscardOppEnergy: true, OpDamageSelf: true,
	OpDiscardFromHand: true, OpFlipCoinsScale: true,
	OpScalePerEnergyAll: true, OpScalePerPrizeTaken: true,
	OpScalePerDamageOpp: true, OpScaleIfStatusOpp: true,
}

var validConds = map[string]bool{"asleep": true, "confused": true, "paralyzed": true, "poisoned": true, "burned": true}

// ValidateOps checa estruturalmente ops vindas de fora do compilador (LLM,
// edição humana). Op inválida → a entrada inteira é rejeitada.
func ValidateOps(ops []Op) error {
	if len(ops) == 0 {
		return fmt.Errorf("lista de ops vazia")
	}
	for i, op := range ops {
		if !validKinds[op.Kind] {
			return fmt.Errorf("op %d: kind desconhecido %q", i, op.Kind)
		}
		if op.Kind == OpStatus && !validConds[op.Cond] {
			return fmt.Errorf("op %d: condição desconhecida %q", i, op.Cond)
		}
		if op.Kind == OpSearch {
			if op.Dest != "hand" && op.Dest != "bench" {
				return fmt.Errorf("op %d: dest inválido %q", i, op.Dest)
			}
			if len(op.Find) == 0 {
				return fmt.Errorf("op %d: busca sem critérios", i)
			}
			for _, f := range op.Find {
				if f.Category != "Pokemon" && f.Category != "Energy" {
					return fmt.Errorf("op %d: categoria de busca inválida %q", i, f.Category)
				}
			}
		}
		if op.N < -1 || op.N > 500 {
			return fmt.Errorf("op %d: n fora de faixa: %d", i, op.N)
		}
	}
	return nil
}

// LoadEffectDB carrega data/effects.json no overlay do CompileEffect.
// Arquivo ausente não é erro (banco começa vazio).
func LoadEffectDB(path string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("lendo banco de efeitos: %w", err)
	}
	db := map[string]EffectEntry{}
	if err := json.Unmarshal(data, &db); err != nil {
		return fmt.Errorf("decodificando %s: %w", path, err)
	}
	effectDB = db
	effectCache = map[string]CompiledEffect{} // invalida cache compilado
	return nil
}

// EffectDB expõe o banco carregado (leitura; usado por cmd/compile-effects).
func EffectDB() map[string]EffectEntry { return effectDB }
