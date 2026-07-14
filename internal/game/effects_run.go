package game

import "github.com/alvaropsouza/pokemon-self-play/internal/cards"

// ExtraAttackDamage sums damage scaling modifiers from energy counts.
func ExtraAttackDamage(g *Game, p int, atk cards.Attack, attacker *PokemonInPlay) int {
	ce := CompileEffect(atk.Effect.EN)
	extra := 0
	for _, op := range ce.Ops {
		if op.Flip {
			continue
		}
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

func (g *Game) applyTrainerEffect(p int, c *cards.Card) bool {
	ce := CompileEffect(c.Effect.EN)
	if ce.Manual {
		return false
	}
	g.runOps(p, ce.Ops, nil)
	return true
}

func (g *Game) applyAttackEffect(p int, atk cards.Attack, attacker *PokemonInPlay) {
	if atk.Effect.EN == "" {
		return
	}
	ce := CompileEffect(atk.Effect.EN)
	// ops that create pending choices during an attack can't be resolved before finishTurn;
	// Trainers handle these cases instead
	for _, op := range ce.Ops {
		if op.Kind == OpSearch || op.Kind == OpSwitchSelf || op.Kind == OpSwitchOpp || op.Kind == OpDiscardFromHand {
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
			// applied before combat in ExtraAttackDamage
		case OpSearch:
			if g.startSearch(p, op, ops[i+1:]) {
				return
			}
			return
		case OpSwitchSelf:
			g.startSwitchPending(p, false, ops[i+1:])
			return
		case OpSwitchOpp:
			g.startSwitchPending(p, true, ops[i+1:])
			return
		case OpDiscardFromHand:
			g.startDiscardHand(p, op, ops[i+1:])
			return
		case OpDiscardOppEnergy:
			opp := 1 - p
			if a := g.Players[opp].Active; a != nil {
				n := op.N
				if n < 0 || n > len(a.Energies) {
					n = len(a.Energies)
				}
				idxs := make([]int, n)
				for j := range idxs {
					idxs[j] = j
				}
				_ = g.discardEnergies(opp, a, idxs)
				g.logf("descarta %d Energia(s) do Ativo do oponente", n)
			}
		case OpDamageSelf:
			if attacker != nil {
				attacker.Damage += op.N
				g.logf("%s recebe %d de dano (recoil)", g.Card(attacker.TopID()).Name.EN, op.N)
			}
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

func (g *Game) shuffleHandIntoDeck(p int, includeTools bool) {
	ps := g.Players[p]
	// new slice avoids writing into the hand's backing array
	ps.Deck = append(append([]string(nil), ps.Hand...), ps.Deck...)
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
