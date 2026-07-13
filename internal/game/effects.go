package game

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/alvaropsouza/pokemon-self-play/internal/cards"
)

// Efeitos de texto: o texto EN da carta é compilado uma única vez em uma lista
// de operações primitivas (Op). Cobertura total obrigatória: se qualquer
// cláusula do texto não casar com um padrão conhecido, o efeito inteiro é
// marcado Manual e cai na arbitragem manual — nunca há resolução parcial.

// OpKind identifica a operação primitiva. String para serialização estável
// em data/effects.json (banco de efeitos preenchido por LLM/humano).
type OpKind string

const (
	OpDraw               OpKind = "draw"                  // compra N cartas (jogador do efeito)
	OpDrawUntil          OpKind = "draw_until"            // compra até ter N na mão
	OpDrawOrMore         OpKind = "draw_or_more"          // compra N, mas se exatamente ExactPrizes prêmios → compra Alt
	OpDiscardHand        OpKind = "discard_hand"          // descarta a mão inteira
	OpShuffleHandBoth    OpKind = "shuffle_hand_both"     // ambos embaralham a mão no deck (Tools inclui ferramentas)
	OpShuffleHandSelf    OpKind = "shuffle_hand_self"     // embaralha a própria mão no deck
	OpDrawBoth           OpKind = "draw_both"             // ambos compram N
	OpDrawPerPrizeBoth   OpKind = "draw_per_prize_both"   // ambos compram = prêmios restantes
	OpDamageOppBench     OpKind = "damage_opp_bench"      // N de dano em cada Pokémon do Banco do oponente
	OpDamageSelfBench    OpKind = "damage_self_bench"     // N de dano em cada Pokémon do Banco próprio
	OpHealSelf           OpKind = "heal_self"             // cura N do atacante
	OpDiscardSelfEnergy  OpKind = "discard_self_energy"   // descarta N Energias do atacante (N=-1 → todas)
	OpScalePerEnergySelf OpKind = "scale_per_energy_self" // +N de dano por Energia no atacante (pré-dano)
	OpScalePerEnergyOpp  OpKind = "scale_per_energy_opp"  // +N de dano por Energia no Ativo do oponente
	OpStatus             OpKind = "status"                // Condição Especial (Cond) no Ativo do alvo
	OpSearch             OpKind = "search"                // busca no deck (Find/Dest/N) — escolha do jogador
	OpShuffleDeck        OpKind = "shuffle_deck"          // embaralha o próprio deck
)

// Op é uma operação primitiva compilada do texto do efeito.
type Op struct {
	Kind        OpKind `json:"kind"`
	N           int    `json:"n,omitempty"`
	Alt         int    `json:"alt,omitempty"`          // OpDrawOrMore: compras alternativas
	ExactPrizes int    `json:"exactPrizes,omitempty"` // OpDrawOrMore: condição de prêmios
	Cond        string `json:"cond,omitempty"`         // condição especial (OpStatus)
	OnSelf      bool   `json:"onSelf,omitempty"`       // OpStatus: alvo é o próprio Ativo (senão, o do oponente)
	Flip        bool   `json:"flip,omitempty"`         // executa só se a moeda der cara
	Tools       bool   `json:"tools,omitempty"`        // OpShuffleHandBoth: ferramentas voltam ao deck também
	Dest        string `json:"dest,omitempty"`         // OpSearch: "hand" | "bench"
	Find        []Find `json:"find,omitempty"`         // OpSearch: critérios (casa qualquer um)
}

// CompiledEffect é o resultado da compilação do texto de um efeito.
type CompiledEffect struct {
	Ops    []Op `json:"ops,omitempty"`
	Manual bool `json:"manual,omitempty"` // cláusula não coberta → arbitragem manual integral
}

// pattern casa uma parte de cláusula e produz zero ou mais Ops.
type pattern struct {
	re    *regexp.Regexp
	build func(m []string) []Op
}

func atoi(s string) int { n, _ := strconv.Atoi(s); return n }

var condAlt = `(asleep|confused|paralyzed|poisoned|burned)`

// Ordem importa: padrões mais específicos primeiro (spans casados são removidos
// da cláusula antes dos padrões seguintes).
var patterns = []pattern{
	{regexp.MustCompile(`each player shuffles their hand (?:and shuffles it )?into (?:the bottom of )?their deck and draws (\d+) cards?`), func(m []string) []Op {
		return []Op{{Kind: OpShuffleHandBoth}, {Kind: OpDrawBoth, N: atoi(m[1])}}
	}},
	{regexp.MustCompile(`each player shuffles their hand[\w' ,]{0,40}into (?:the bottom of )?their deck`), func(m []string) []Op {
		return []Op{{Kind: OpShuffleHandBoth, Tools: strings.Contains(m[0], "tool")}}
	}},
	{regexp.MustCompile(`each player draws a card for each of their remaining prize cards?`), func(m []string) []Op {
		return []Op{{Kind: OpDrawPerPrizeBoth}}
	}},
	// "draws a card for each of their remaining prize cards" — continuação inline
	// de "each player shuffles … and draws …" (ex: Iono).
	{regexp.MustCompile(`draws a cards? for each of their remaining prize cards?`), func(m []string) []Op {
		return []Op{{Kind: OpDrawPerPrizeBoth}}
	}},
	{regexp.MustCompile(`discard your hand`), func(m []string) []Op {
		return []Op{{Kind: OpDiscardHand}}
	}},
	{regexp.MustCompile(`draw cards? until you have (\d+) cards? in your hand`), func(m []string) []Op {
		return []Op{{Kind: OpDrawUntil, N: atoi(m[1])}}
	}},
	{regexp.MustCompile(`draw (\d+) cards?`), func(m []string) []Op {
		return []Op{{Kind: OpDraw, N: atoi(m[1])}}
	}},
	{regexp.MustCompile(`(?:this attack (?:also )?)?does (\d+) damage to each of your opponent's benched pokemon`), func(m []string) []Op {
		return []Op{{Kind: OpDamageOppBench, N: atoi(m[1])}}
	}},
	{regexp.MustCompile(`(?:this attack (?:also )?)?does (\d+) damage to each of your benched pokemon`), func(m []string) []Op {
		return []Op{{Kind: OpDamageSelfBench, N: atoi(m[1])}}
	}},
	{regexp.MustCompile(`heal (\d+) damage from this pokemon`), func(m []string) []Op {
		return []Op{{Kind: OpHealSelf, N: atoi(m[1])}}
	}},
	{regexp.MustCompile(`discard all[\w ]{0,20}energy from this pokemon`), func(m []string) []Op {
		return []Op{{Kind: OpDiscardSelfEnergy, N: -1}}
	}},
	{regexp.MustCompile(`discard (\d+)[\w ]{0,20}energy (?:cards? )?from this pokemon`), func(m []string) []Op {
		return []Op{{Kind: OpDiscardSelfEnergy, N: atoi(m[1])}}
	}},
	{regexp.MustCompile(`(?:this attack )?does (\d+) more damage for each[\w ]{0,30}energy attached to this pokemon`), func(m []string) []Op {
		return []Op{{Kind: OpScalePerEnergySelf, N: atoi(m[1])}}
	}},
	{regexp.MustCompile(`(?:this attack )?does (\d+) more damage for each[\w ]{0,30}energy attached to your opponent's active pokemon`), func(m []string) []Op {
		return []Op{{Kind: OpScalePerEnergyOpp, N: atoi(m[1])}}
	}},
	{regexp.MustCompile(`this pokemon is now ` + condAlt + `(?: and ` + condAlt + `)?`), func(m []string) []Op {
		ops := []Op{{Kind: OpStatus, Cond: m[1], OnSelf: true}}
		if m[2] != "" {
			ops = append(ops, Op{Kind: OpStatus, Cond: m[2], OnSelf: true})
		}
		return ops
	}},
	{regexp.MustCompile(`(?:the defending pokemon|your opponent's active pokemon) is now ` + condAlt + `(?: and ` + condAlt + `)?`), func(m []string) []Op {
		ops := []Op{{Kind: OpStatus, Cond: m[1]}}
		if m[2] != "" {
			ops = append(ops, Op{Kind: OpStatus, Cond: m[2]})
		}
		return ops
	}},
	{regexp.MustCompile(`search your deck for (?:up to (\d+) )?(.+?), reveal (?:it|them), and put (?:it|them) into your hand`), func(m []string) []Op {
		return buildSearch(m[1], m[2], "hand")
	}},
	{regexp.MustCompile(`search your deck for (?:up to (\d+) )?(.+?) and put (?:it|them) onto your bench`), func(m []string) []Op {
		return buildSearch(m[1], m[2], "bench")
	}},
	{regexp.MustCompile(`shuffle your deck`), func(m []string) []Op {
		return []Op{{Kind: OpShuffleDeck}}
	}},
	{regexp.MustCompile(`shuffle your hand.*?into (?:the bottom of )?your deck`), func(m []string) []Op {
		return []Op{{Kind: OpShuffleHandSelf}}
	}},
}

// reConditionalInstead casa "if you have exactly N prize cards remaining, draw M cards instead"
// (cláusula condicional que sobrescreve o draw base da cláusula anterior).
var reConditionalInstead = regexp.MustCompile(
	`if you have exactly (\d+) prize cards? remaining,\s*draw (\d+) cards? instead`,
)

// buildSearch monta a op de busca; alternativas não reconhecidas → nil (a
// cláusula fica sem cobertura e o efeito cai em Manual).
func buildSearch(count, list, dest string) []Op {
	finds := parseFinds(list)
	if finds == nil {
		return nil
	}
	n := atoi(count)
	if n == 0 {
		n = 1
	}
	return []Op{{Kind: OpSearch, N: n, Dest: dest, Find: finds}}
}

// símbolo de energia do TCGdex ({f}, {w}...) → nome do tipo.
var symbolType = map[string]string{
	"g": "Grass", "r": "Fire", "w": "Water", "l": "Lightning", "p": "Psychic",
	"f": "Fighting", "d": "Darkness", "m": "Metal", "n": "Dragon", "c": "Colorless",
}

var reFindAlt = regexp.MustCompile(`^(?:an? )?(basic )?(?:\{(\w)\} )?(pokemon|energy)(?: cards?)?$`)

// parseFinds interpreta a lista de alternativas ("a basic {f} energy card or a
// basic {f} pokemon"). Qualquer alternativa não reconhecida → nil.
func parseFinds(list string) []Find {
	var finds []Find
	for _, alt := range strings.Split(list, " or ") {
		m := reFindAlt.FindStringSubmatch(strings.TrimSpace(alt))
		if m == nil {
			return nil
		}
		f := Find{Type: symbolType[m[2]]}
		switch m[3] {
		case "pokemon":
			f.Category = "Pokemon"
			if m[1] != "" {
				f.Stage = "Basic"
			}
		case "energy":
			// Só Energia Básica é suportada ("basic ... energy").
			if m[1] == "" {
				return nil
			}
			f.Category = "Energy"
		}
		finds = append(finds, f)
	}
	return finds
}

// reFiller: sobras aceitáveis entre spans casados (conectivos e pontuação).
var reFiller = regexp.MustCompile(`^[\s,]*(?:and|then|also|afterwards?)?[\s,]*$`)

// reParen remove texto entre parênteses (lembretes de regra, não mecânica).
var reParen = regexp.MustCompile(`\([^)]*\)`)

var effectCache = map[string]CompiledEffect{}

// CompileEffect compila texto EN de efeito em ops. Ordem: compilador regex
// (offline puro) → banco de efeitos (data/effects.json, preenchido por
// LLM/humano via cmd/compile-effects) → Manual. Cache por texto — motor
// single-threaded, sem lock.
func CompileEffect(text string) CompiledEffect {
	if text == "" {
		return CompiledEffect{}
	}
	if ce, ok := effectCache[text]; ok {
		return ce
	}
	ce := compile(text)
	if ce.Manual {
		if entry, ok := effectDB[text]; ok && !entry.Manual && ValidateOps(entry.Ops) == nil {
			ce = CompiledEffect{Ops: entry.Ops}
		}
	}
	effectCache[text] = ce
	return ce
}

func compile(text string) CompiledEffect {
	low := strings.ToLower(text)
	low = strings.ReplaceAll(low, "é", "e") // pokémon → pokemon
	low = reParen.ReplaceAllString(low, "")

	var ops []Op
	flipNext := false
	for _, clause := range strings.Split(low, ".") {
		clause = strings.TrimSpace(clause)
		if clause == "" {
			continue
		}
		if clause == "flip a coin" {
			flipNext = true
			continue
		}
		flip := false
		if rest, ok := strings.CutPrefix(clause, "if heads,"); ok && flipNext {
			clause = strings.TrimSpace(rest)
			flip = true
			flipNext = false
		}

		// Cláusula condicional "instead": checar ANTES do loop de padrões para
		// evitar que draw (\d+) consuma o "draw M" interno antes da regex ver a
		// cláusula inteira.
		if m := reConditionalInstead.FindStringSubmatch(clause); m != nil {
			for i := len(ops) - 1; i >= 0; i-- {
				if ops[i].Kind == OpDraw {
					ops[i] = Op{
						Kind:        OpDrawOrMore,
						N:           ops[i].N,
						Alt:         atoi(m[2]),
						ExactPrizes: atoi(m[1]),
					}
					break
				}
			}
			continue
		}

		start := len(ops)
		for _, pat := range patterns {
			m := pat.re.FindStringSubmatch(clause)
			if m == nil {
				continue
			}
			built := pat.build(m)
			if built == nil {
				continue // padrão casou mas conteúdo não é suportado — não consome
			}
			ops = append(ops, built...)
			clause = strings.Replace(clause, m[0], " ", 1)
		}
		// Cobertura total: sobra não trivial na cláusula → efeito manual.
		if !reFiller.MatchString(clause) {
			return CompiledEffect{Manual: true}
		}
		if flip {
			for i := start; i < len(ops); i++ {
				ops[i].Flip = true
			}
		}
	}
	if flipNext {
		// "flip a coin" sem "if heads" coberto → manual.
		return CompiledEffect{Manual: true}
	}
	return CompiledEffect{Ops: ops}
}

// ExtraAttackDamage soma modificadores de dano por contagem de Energia.
func ExtraAttackDamage(g *Game, p int, atk cards.Attack, attacker *PokemonInPlay) int {
	ce := CompileEffect(atk.Effect.EN)
	extra := 0
	for _, op := range ce.Ops {
		switch op.Kind {
		case OpScalePerEnergySelf:
			extra += op.N * len(attacker.Energies)
		case OpScalePerEnergyOpp:
			if opp := g.Players[1-p].Active; opp != nil {
				extra += op.N * len(opp.Energies)
			}
		}
	}
	return extra
}

// applyTrainerEffect resolve efeitos de Treinador compilados.
// Retorna true se totalmente resolvido; false = arbitragem manual.
func (g *Game) applyTrainerEffect(p int, c *cards.Card) bool {
	ce := CompileEffect(c.Effect.EN)
	if ce.Manual {
		return false
	}
	g.runOps(p, ce.Ops, nil)
	return true
}

// applyAttackEffect resolve efeitos de ataque após o dano principal.
// attacker = Ativo do atacante capturado antes de qualquer nocaute.
// Não chama resolveKnockouts — o chamador resolve uma vez ao final.
func (g *Game) applyAttackEffect(p int, atk cards.Attack, attacker *PokemonInPlay) {
	if atk.Effect.EN == "" {
		return
	}
	ce := CompileEffect(atk.Effect.EN)
	// ponytail: busca em ataque exigiria escolha pendente ANTES do fim do
	// turno (finishTurn) — fica manual até o motor suportar; Treinadores cobrem.
	for _, op := range ce.Ops {
		if op.Kind == OpSearch {
			ce = CompiledEffect{Manual: true}
			break
		}
	}
	if ce.Manual {
		g.logf("efeito de %s (arbitragem manual): %s", atk.Name.EN, atk.Effect.EN)
		return
	}
	g.runOps(p, ce.Ops, attacker)
}

// runOps executa ops compiladas para o jogador p. attacker é o Ativo do
// atacante em efeitos de ataque (nil em Treinadores — textos de Treinador não
// produzem ops que dependem do atacante). Uma op de busca pode interromper a
// execução (escolha pendente); as ops restantes continuam em ResolveChoice.
func (g *Game) runOps(p int, ops []Op, attacker *PokemonInPlay) {
	for i, op := range ops {
		if op.Flip {
			if !g.flip() {
				g.logf("coroa: efeito não acontece")
				continue
			}
			g.logf("cara: efeito acontece")
		}
		switch op.Kind {
		case OpDraw:
			g.DrawCards(p, op.N)
		case OpDrawOrMore:
			n := op.N
			if op.ExactPrizes > 0 && len(g.Players[p].Prizes) == op.ExactPrizes {
				n = op.Alt
				g.logf("jogador %d: condição (%d prêmios) → compra %d", p+1, op.ExactPrizes, n)
			}
			g.DrawCards(p, n)
		case OpShuffleHandSelf:
			g.shuffleHandIntoDeck(p, false)
			g.logf("jogador %d: embaralha a mão no deck", p+1)
		case OpDrawUntil:
			for len(g.Players[p].Hand) < op.N && g.drawCard(p) {
			}
			g.logf("jogador %d: compra até ter %d na mão", p+1, op.N)
		case OpDiscardHand:
			ps := g.Players[p]
			ps.Discard = append(ps.Discard, ps.Hand...)
			ps.Hand = nil
			g.logf("jogador %d: descarta a mão", p+1)
		case OpShuffleHandBoth:
			for i := 0; i < 2; i++ {
				g.shuffleHandIntoDeck(i, op.Tools)
			}
			g.logf("ambos embaralham a mão no deck")
		case OpDrawBoth:
			for i := 0; i < 2; i++ {
				g.DrawCards(i, op.N)
			}
		case OpDrawPerPrizeBoth:
			for i := 0; i < 2; i++ {
				g.DrawCards(i, len(g.Players[i].Prizes))
			}
		case OpDamageOppBench:
			for _, b := range g.Players[1-p].Bench {
				b.Damage += op.N
			}
			g.logf("dano de %d em cada Pokémon do Banco do oponente", op.N)
		case OpDamageSelfBench:
			for _, b := range g.Players[p].Bench {
				b.Damage += op.N
			}
			g.logf("dano de %d em cada Pokémon do Banco próprio", op.N)
		case OpHealSelf:
			if attacker == nil {
				break
			}
			attacker.Damage -= op.N
			if attacker.Damage < 0 {
				attacker.Damage = 0
			}
			g.logf("cura %d de %s", op.N, g.Card(attacker.TopID()).Name.EN)
		case OpDiscardSelfEnergy:
			if attacker == nil {
				break
			}
			n := op.N
			if n < 0 || n > len(attacker.Energies) {
				n = len(attacker.Energies)
			}
			ids := make([]int, n)
			for i := range ids {
				ids[i] = i
			}
			_ = g.discardEnergies(p, attacker, ids)
			g.logf("descarta %d Energia(s) de %s", n, g.Card(attacker.TopID()).Name.EN)
		case OpScalePerEnergySelf, OpScalePerEnergyOpp:
			// Modificadores de dano — aplicados antes, em ExtraAttackDamage.
		case OpSearch:
			if g.startSearch(p, op, ops[i+1:]) {
				return // pendente: restante roda em ResolveChoice
			}
			return // auto-resolvida: startSearch já rodou o restante
		case OpShuffleDeck:
			g.shuffle(g.Players[p].Deck)
			g.logf("jogador %d: embaralha o deck", p+1)
		case OpStatus:
			target := 1 - p
			if op.OnSelf {
				target = p
			}
			g.applyStatusToActive(target, op.Cond)
		}
	}
}

// shuffleHandIntoDeck devolve a mão (e opcionalmente ferramentas) ao deck e embaralha.
func (g *Game) shuffleHandIntoDeck(p int, includeTools bool) {
	ps := g.Players[p]
	ps.Deck = append(ps.Hand, ps.Deck...)
	ps.Hand = nil
	if includeTools {
		all := append([]*PokemonInPlay{ps.Active}, ps.Bench...)
		for _, pk := range all {
			if pk != nil && pk.Tool != "" {
				ps.Deck = append(ps.Deck, pk.Tool)
				pk.Tool = ""
			}
		}
	}
	g.shuffle(ps.Deck)
}

func (g *Game) applyStatusToActive(p int, status string) {
	a := g.Players[p].Active
	if a == nil {
		return
	}
	name := g.Card(a.TopID()).Name.EN
	switch status {
	case "asleep":
		a.Rot = CondAsleep
	case "confused":
		a.Rot = CondConfused
	case "paralyzed":
		a.Rot = CondParalyzed
	case "poisoned":
		a.Poisoned = true
	case "burned":
		a.Burned = true
	}
	g.logf("%s está %s", name, status)
}
