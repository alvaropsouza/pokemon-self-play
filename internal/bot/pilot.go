package bot

import (
	"github.com/alvaropsouza/pokemon-self-play/internal/cards"
	"github.com/alvaropsouza/pokemon-self-play/internal/game"
)

func Setup(g *game.Game, player int) error {
	if idx := bestBasicInHand(g, player); idx >= 0 {
		cmd := game.PlaceActiveCmd{Player: player, HandIdx: idx}
		if err := cmd.Execute(g); err != nil {
			return err
		}
	}
	for i := 0; i < 2; i++ {
		idx := bestBasicInHand(g, player)
		if idx < 0 {
			break
		}
		if err := (game.PlaceBenchCmd{Player: player, HandIdx: idx}).Execute(g); err != nil {
			break
		}
	}
	return game.FinishSetupCmd{Player: player}.Execute(g)
}

func PromoteIfNeeded(g *game.Game, player int) {
	if !g.NeedPromote[player] {
		return
	}
	ps := g.Players[player]
	best, bestHP := 0, -1
	for i, pk := range ps.Bench {
		hp := g.Card(pk.TopID()).HP - pk.Damage
		if hp > bestHP {
			best, bestHP = i, hp
		}
	}
	_ = game.PromoteCmd{Player: player, BenchIdx: best}.Execute(g)
}

func ResolvePending(g *game.Game, player int) {
	pc := g.Pending
	if pc == nil || pc.Player != player {
		return
	}
	n := min(pc.Max, len(pc.Candidates))
	picks := make([]int, n)
	for i := range picks {
		picks[i] = i
	}
	_ = game.ResolveChoiceCmd{Player: player, Picks: picks}.Execute(g)
}

func TakeTurn(g *game.Game, player int) {
	PromoteIfNeeded(g, player)
	ResolvePending(g, player)
	if g.Phase != game.PhaseTurn || g.Current != player {
		return
	}
	ps := g.Players[player]

	for len(ps.Bench) < 3 {
		idx := bestBasicInHand(g, player)
		if idx < 0 {
			break
		}
		if (game.PlaceBenchCmd{Player: player, HandIdx: idx}).Execute(g) != nil {
			break
		}
	}

	for _, slot := range slots(g, player) {
		for i := 0; i < len(ps.Hand); i++ {
			if (game.EvolveCmd{Player: player, HandIdx: i, Slot: slot}).Execute(g) == nil {
				break
			}
		}
	}

	if !ps.SupporterPlayed {
		playBestSupporter(g, player)
		ResolvePending(g, player)
	}

	playUsefulItems(g, player)

	if idx := energyInHand(g, player); idx >= 0 {
		target := game.ActiveSlot
		if ps.Active != nil && bestAttack(g, player, ps.Active) >= 0 && len(ps.Bench) > 0 {
			target = 0
		}
		_ = game.AttachEnergyCmd{Player: player, HandIdx: idx, Slot: target}.Execute(g)
	}

	if ps.Active != nil {
		atk := MCPickAttack(g, player)
		if atk >= 0 {
			if (game.AttackCmd{Player: player, AtkIdx: atk}).Execute(g) == nil {
				return
			}
		}
	}
	_ = game.EndTurnCmd{Player: player}.Execute(g)
}

func slots(g *game.Game, player int) []int {
	out := []int{game.ActiveSlot}
	for i := range g.Players[player].Bench {
		out = append(out, i)
	}
	return out
}

func bestBasicInHand(g *game.Game, player int) int {
	best, bestHP := -1, -1
	for i, id := range g.Players[player].Hand {
		c := g.Card(id)
		if c.IsBasicPokemon() && c.HP > bestHP {
			best, bestHP = i, c.HP
		}
	}
	return best
}

func energyInHand(g *game.Game, player int) int {
	for i, id := range g.Players[player].Hand {
		if g.Card(id).Category == cards.CategoryEnergy {
			return i
		}
	}
	return -1
}

func supporterScore(g *game.Game, player int, ops []game.Op) int {
	ps := g.Players[player]
	score := 0
	for _, op := range ops {
		switch op.Kind {
		case game.OpDraw, game.OpDrawUntil, game.OpDrawBoth, game.OpDrawPerPrizeBoth,
			game.OpShuffleHandBoth, game.OpShuffleHandSelf:
			score += 30
		case game.OpSearch:
			score += 20
		case game.OpSwitchOpp:
			opp := g.Players[1-player]
			if opp.Active != nil {
				c := g.Card(opp.Active.TopID())
				remaining := c.HP - opp.Active.Damage
				if remaining <= 60 {
					score += 25
				} else {
					score += 5
				}
			}
		case game.OpStatus:
			if !op.OnSelf {
				score += 10
			}
		default:
		}
	}
	if len(ps.Hand) > 5 {
		hasRefresh := false
		for _, op := range ops {
			if op.Kind == game.OpShuffleHandBoth || op.Kind == game.OpShuffleHandSelf {
				hasRefresh = true
			}
		}
		if !hasRefresh {
			return 0
		}
	}
	return score
}

func playBestSupporter(g *game.Game, player int) {
	ps := g.Players[player]
	bestIdx, bestScore := -1, 0
	for i, id := range ps.Hand {
		c := g.Card(id)
		if c.Category != cards.CategoryTrainer || c.TrainerType != "Supporter" {
			continue
		}
		ce := game.CompileEffect(c.Effect.EN)
		if ce.Manual {
			continue
		}
		if s := supporterScore(g, player, ce.Ops); s > bestScore {
			bestScore, bestIdx = s, i
		}
	}
	if bestIdx >= 0 {
		_ = game.PlaySupporterCmd{Player: player, HandIdx: bestIdx}.Execute(g)
	}
}

func playUsefulItems(g *game.Game, player int) {
	ps := g.Players[player]
	if len(ps.Hand) > 4 {
		return
	}
	for i := 0; i < len(ps.Hand); i++ {
		c := g.Card(ps.Hand[i])
		if c.Category != cards.CategoryTrainer || c.TrainerType != "Item" {
			continue
		}
		ce := game.CompileEffect(c.Effect.EN)
		if ce.Manual {
			continue
		}
		for _, op := range ce.Ops {
			if op.Kind == game.OpSearch {
				if (game.PlayItemCmd{Player: player, HandIdx: i}).Execute(g) == nil {
					ResolvePending(g, player)
					i--
				}
				break
			}
		}
	}
}

func attackScore(g *game.Game, player int, pk *game.PokemonInPlay, atkIdx int) int {
	c := g.Card(pk.TopID())
	atk := c.Attacks[atkIdx]
	if !g.CostPaid(pk, atk.Cost) {
		return -1
	}
	score := game.BaseDamage(atk.Damage) + game.ExtraAttackDamage(g, player, atk, pk)
	ce := game.CompileEffect(atk.Effect.EN)
	for _, op := range ce.Ops {
		switch op.Kind {
		case game.OpStatus:
			if !op.OnSelf {
				score += 15
			}
		case game.OpDamageOppBench:
			score += op.N * len(g.Players[1-player].Bench) / 2
		case game.OpSwitchOpp:
			opp := g.Players[1-player]
			if opp.Active != nil {
				remaining := g.Card(opp.Active.TopID()).HP - opp.Active.Damage
				if remaining > 60 {
					score += 10
				}
			}
		case game.OpHealSelf:
			score += op.N / 4
		default:
		}
	}
	return score
}

func bestAttack(g *game.Game, player int, pk *game.PokemonInPlay) int {
	best, bestScore := -1, -1
	c := g.Card(pk.TopID())
	for i := range c.Attacks {
		if s := attackScore(g, player, pk, i); s > bestScore {
			bestScore, best = s, i
		}
	}
	return best
}
