package game

import (
	"regexp"
	"strconv"
	"strings"
)

// OpKind identifies a primitive operation. String for stable serialization in data/effects.json.
type OpKind string

const (
	OpDraw               OpKind = "draw"
	OpDrawUntil          OpKind = "draw_until"
	OpDrawOrMore         OpKind = "draw_or_more"
	OpDiscardHand        OpKind = "discard_hand"
	OpShuffleHandBoth    OpKind = "shuffle_hand_both"
	OpShuffleHandSelf    OpKind = "shuffle_hand_self"
	OpDrawBoth           OpKind = "draw_both"
	OpDrawPerPrizeBoth   OpKind = "draw_per_prize_both"
	OpDamageOppBench     OpKind = "damage_opp_bench"
	OpDamageSelfBench    OpKind = "damage_self_bench"
	OpHealSelf           OpKind = "heal_self"
	OpDiscardSelfEnergy  OpKind = "discard_self_energy"
	OpScalePerEnergySelf OpKind = "scale_per_energy_self"
	OpScalePerEnergyOpp  OpKind = "scale_per_energy_opp"
	OpStatus             OpKind = "status"
	OpSearch             OpKind = "search"
	OpShuffleDeck        OpKind = "shuffle_deck"
	OpSwitchSelf         OpKind = "switch_self"
	OpSwitchOpp          OpKind = "switch_opp"
	OpDiscardOppEnergy   OpKind = "discard_opp_energy"
	OpDamageSelf         OpKind = "damage_self"
	OpDiscardFromHand    OpKind = "discard_from_hand"
	OpFlipCoinsScale     OpKind = "flip_coins_scale"
	OpScalePerEnergyAll  OpKind = "scale_per_energy_all"
	OpScalePerPrizeTaken OpKind = "scale_per_prize_taken"
	OpScalePerDamageOpp  OpKind = "scale_per_damage_opp"
	OpScaleIfStatusOpp      OpKind = "scale_if_status_opp"
	OpDamageCountersPerHand OpKind = "damage_counters_per_hand"
)

// Op is a compiled primitive operation from effect text.
type Op struct {
	Kind        OpKind `json:"kind"`
	N           int    `json:"n,omitempty"`
	Alt         int    `json:"alt,omitempty"`
	ExactPrizes int    `json:"exactPrizes,omitempty"`
	Cond        string `json:"cond,omitempty"`
	OnSelf      bool   `json:"onSelf,omitempty"`
	Flip        bool   `json:"flip,omitempty"`
	Tools       bool   `json:"tools,omitempty"`
	Cost        bool   `json:"cost,omitempty"`
	Dest        string `json:"dest,omitempty"`
	Find        []Find `json:"find,omitempty"`
}

// CompiledEffect is the result of compiling effect text.
type CompiledEffect struct {
	Ops    []Op `json:"ops,omitempty"`
	Manual bool `json:"manual,omitempty"`
}

type pattern struct {
	re    *regexp.Regexp
	build func(m []string) []Op
}

func atoi(s string) int { n, _ := strconv.Atoi(s); return n }

var condAlt = `(asleep|confused|paralyzed|poisoned|burned)`

// patterns must be ordered most-specific first; matched spans are removed before subsequent patterns run.
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
	{regexp.MustCompile(`draws a cards? for each of their remaining prize cards?`), func(m []string) []Op {
		return []Op{{Kind: OpDrawPerPrizeBoth}}
	}},
	{regexp.MustCompile(`discard your hand`), func(m []string) []Op {
		return []Op{{Kind: OpDiscardHand}}
	}},
	{regexp.MustCompile(`you can use this card only if you discard (\d+) other cards? from your hand`), func(m []string) []Op {
		return []Op{{Kind: OpDiscardFromHand, N: atoi(m[1]), Cost: true}}
	}},
	{regexp.MustCompile(`discard (\d+) (?:other )?cards? from your hand`), func(m []string) []Op {
		return []Op{{Kind: OpDiscardFromHand, N: atoi(m[1])}}
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
	{regexp.MustCompile(`(?:this attack )?does (\d+) less damage for each[\w ]{0,30}energy attached to your opponent's active pokemon`), func(m []string) []Op {
		return []Op{{Kind: OpScalePerEnergyOpp, N: -atoi(m[1])}}
	}},
	{regexp.MustCompile(`(?:this attack )?does (\d+) damage for each[\w\s{}.]{0,50}energy attached to all of your pokemon`), func(m []string) []Op {
		return []Op{{Kind: OpScalePerEnergyAll, N: atoi(m[1])}}
	}},
	{regexp.MustCompile(`(?:this attack )?does (\d+) damage for each prize card you have taken`), func(m []string) []Op {
		return []Op{{Kind: OpScalePerPrizeTaken, N: atoi(m[1])}}
	}},
	{regexp.MustCompile(`(?:this attack )?does (\d+) damage for each damage counter on your opponent's active pokemon`), func(m []string) []Op {
		return []Op{{Kind: OpScalePerDamageOpp, N: atoi(m[1])}}
	}},
	{regexp.MustCompile(`if your opponent's active pokemon is now ` + condAlt + `, (?:this attack )?does (\d+) more damage`), func(m []string) []Op {
		return []Op{{Kind: OpScaleIfStatusOpp, Cond: m[1], N: atoi(m[2])}}
	}},
	{regexp.MustCompile(`if your opponent's active pokemon is ` + condAlt + `, (?:this attack )?does (\d+) more damage`), func(m []string) []Op {
		return []Op{{Kind: OpScaleIfStatusOpp, Cond: m[1], N: atoi(m[2])}}
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
	{regexp.MustCompile(`once during your turn[^.]*?you may use this ability`), func(m []string) []Op { return []Op{} }},
	{regexp.MustCompile(`as often as you like during your turn[^.]*?you may use this ability`), func(m []string) []Op { return []Op{} }},
	{regexp.MustCompile(`you can.t use more than 1[^.]*?ability each turn`), func(m []string) []Op { return []Op{} }},
	{regexp.MustCompile(`(?:you may )?switch this pokemon with (?:1 of )?your benched pokemon`), func(m []string) []Op {
		return []Op{{Kind: OpSwitchSelf}}
	}},
	{regexp.MustCompile(`switch your active pokemon with (?:1 of )?your benched pokemon`), func(m []string) []Op {
		return []Op{{Kind: OpSwitchSelf}}
	}},
	{regexp.MustCompile(`switch out your opponent.s active pokemon(?: to the bench)?`), func(m []string) []Op {
		return []Op{{Kind: OpSwitchOpp}}
	}},
	{regexp.MustCompile(`switch (?:1 of )?your opponent's benched pokemon with their active pokemon`), func(m []string) []Op {
		return []Op{{Kind: OpSwitchOpp}}
	}},
	{regexp.MustCompile(`switch in (?:1 of )?your opponent's benched pokemon(?: to the active spot)?`), func(m []string) []Op {
		return []Op{{Kind: OpSwitchOpp}}
	}},
	{regexp.MustCompile(`discard (\d+)[\w ]{0,30}energy (?:cards? )?from your opponent's active pokemon`), func(m []string) []Op {
		return []Op{{Kind: OpDiscardOppEnergy, N: atoi(m[1])}}
	}},
	{regexp.MustCompile(`discard an?[\w ]{0,20}energy(?:\scard)? from your opponent's active pokemon`), func(m []string) []Op {
		return []Op{{Kind: OpDiscardOppEnergy, N: 1}}
	}},
	{regexp.MustCompile(`discard all[\w ]{0,20}energy from your opponent's active pokemon`), func(m []string) []Op {
		return []Op{{Kind: OpDiscardOppEnergy, N: -1}}
	}},
	{regexp.MustCompile(`place (\d+) damage counters? on your opponent's active pokemon for each card in your hand`), func(m []string) []Op {
		return []Op{{Kind: OpDamageCountersPerHand, N: atoi(m[1])}}
	}},
	{regexp.MustCompile(`this pokemon (?:also )?does (\d+) damage to itself`), func(m []string) []Op {
		return []Op{{Kind: OpDamageSelf, N: atoi(m[1])}}
	}},
	{regexp.MustCompile(`put (\d+) damage counters? on this pokemon`), func(m []string) []Op {
		return []Op{{Kind: OpDamageSelf, N: atoi(m[1]) * 10}}
	}},
}

var reConditionalInstead = regexp.MustCompile(
	`if you have exactly (\d+) prize cards? remaining,\s*draw (\d+) cards? instead`,
)

var reFlipNCoins = regexp.MustCompile(`flip (\d+) coins?`)
var reFlipUntilTails = regexp.MustCompile(`flip a coin until you get tails`)
var reFlipScaleDamage = regexp.MustCompile(`(?:this attack )?does? (\d+) (?:more )?damage for each heads?`)
var reFlipAllHeadsDamage = regexp.MustCompile(`if both of them are heads, (?:this attack )?does (\d+) more damage`)

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

var symbolType = map[string]string{
	"g": "Grass", "r": "Fire", "w": "Water", "l": "Lightning", "p": "Psychic",
	"f": "Fighting", "d": "Darkness", "m": "Metal", "n": "Dragon", "c": "Colorless",
}

var reFindAlt = regexp.MustCompile(`^(?:an? )?(basic )?(?:\{(\w)\} )?(pokemon|energy)(?: cards?)?$`)
var reHPSuffix = regexp.MustCompile(` with (\d+) hp or less$`)

func parseFinds(list string) []Find {
	maxHP := 0
	if m := reHPSuffix.FindStringSubmatch(list); m != nil {
		maxHP = atoi(m[1])
		list = strings.TrimSuffix(list, m[0])
	}
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
			f.MaxHP = maxHP
		case "energy":
			if m[1] == "" || maxHP > 0 {
				return nil
			}
			f.Category = "Energy"
		}
		finds = append(finds, f)
	}
	return finds
}

var reFiller = regexp.MustCompile(`^[\s,]*(?:and|then|also|afterwards?)?[\s,]*$`)
var reParen = regexp.MustCompile(`\([^)]*\)`)

var effectCache = map[string]CompiledEffect{}

// CompileEffect compiles EN effect text into ops.
// Order: regex compiler → effects.json DB → Manual. Cached per text (single-threaded engine, no lock).
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
	low = strings.ReplaceAll(low, "é", "e")
	low = reParen.ReplaceAllString(low, "")

	var ops []Op
	flipNext := false
	flipCount := -1 // -1: none; 0: until tails; N>0: flip N coins
	for _, clause := range strings.Split(low, ".") {
		clause = strings.TrimSpace(clause)
		if clause == "" {
			continue
		}
		if clause == "flip a coin" {
			flipNext = true
			continue
		}
		if reFlipUntilTails.MatchString(clause) {
			clause = reFlipUntilTails.ReplaceAllString(clause, " ")
			if reFiller.MatchString(clause) {
				flipCount = 0
				continue
			}
		}
		if m := reFlipNCoins.FindStringSubmatch(clause); m != nil {
			clause = strings.Replace(clause, m[0], " ", 1)
			if reFiller.MatchString(clause) {
				flipCount = atoi(m[1])
				continue
			}
		}
		if flipCount >= 0 {
			if m := reFlipScaleDamage.FindStringSubmatch(clause); m != nil {
				ops = append(ops, Op{Kind: OpFlipCoinsScale, N: flipCount, Alt: atoi(m[1])})
				clause = strings.Replace(clause, m[0], " ", 1)
				flipCount = -1
			} else if m := reFlipAllHeadsDamage.FindStringSubmatch(clause); m != nil {
				ops = append(ops, Op{Kind: OpFlipCoinsScale, N: flipCount, Alt: atoi(m[1]), OnSelf: true})
				clause = strings.Replace(clause, m[0], " ", 1)
				flipCount = -1
			}
		}
		flip := false
		if rest, ok := strings.CutPrefix(clause, "if heads,"); ok && flipNext {
			clause = strings.TrimSpace(rest)
			flip = true
			flipNext = false
		}

		// check "instead" conditional before pattern loop — prevents draw(\d+) consuming
		// the inner "draw M" before reConditionalInstead sees the full clause
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
				continue
			}
			ops = append(ops, built...)
			clause = strings.Replace(clause, m[0], " ", 1)
		}
		if !reFiller.MatchString(clause) {
			return CompiledEffect{Manual: true}
		}
		if flip {
			for i := start; i < len(ops); i++ {
				ops[i].Flip = true
			}
		}
	}
	if flipNext || flipCount >= 0 {
		return CompiledEffect{Manual: true}
	}
	return CompiledEffect{Ops: ops}
}
