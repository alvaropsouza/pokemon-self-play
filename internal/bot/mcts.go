package bot

// ponytail: flat MC sem árvore UCB1 — suficiente para MCBudget≤500; add tree se budget measurably helps.

import (
	"math/rand"

	"github.com/alvaropsouza/pokemon-self-play/internal/game"
)

const MCBudget = 100

// MCPickAttack returns the attack index with the highest simulated win rate, or -1 to pass.
func MCPickAttack(g *game.Game, player int) int {
	pk := g.Players[player].Active
	if pk == nil {
		return -1
	}
	c := g.Card(pk.TopID())

	type option struct {
		atk   int
		score int
	}

	var opts []option
	for i := range c.Attacks {
		if g.CostPaid(pk, c.Attacks[i].Cost) {
			opts = append(opts, option{atk: i})
		}
	}
	if len(opts) == 0 {
		return -1
	}
	// pass enters last so score ties favor attacking
	opts = append(opts, option{atk: -1})

	seed := int64(g.TurnNumber)*1000 + int64(player)
	rng := rand.New(rand.NewSource(seed))

	for i := range opts {
		for sim := 0; sim < MCBudget; sim++ {
			clone := g.CloneWithSeed(rng.Int63())
			applyOption(clone, player, opts[i].atk)
			opts[i].score += rollout(clone, player, rng.Int63())
		}
	}

	best, bestScore := -1, -1
	for _, o := range opts {
		if o.score > bestScore {
			bestScore, best = o.score, o.atk
		}
	}
	return best
}

func applyOption(g *game.Game, player, atk int) {
	if atk >= 0 {
		_ = game.AttackCmd{Player: player, AtkIdx: atk}.Execute(g)
	} else {
		_ = game.EndTurnCmd{Player: player}.Execute(g)
	}
}

func rollout(g *game.Game, player int, seed int64) int {
	rng := rand.New(rand.NewSource(seed))
	const maxTurns = 60
	for i := 0; i < maxTurns && g.Phase == game.PhaseTurn && g.Winner < 0; i++ {
		p := g.Current
		PromoteIfNeeded(g, p)
		if pc := g.Pending; pc != nil {
			ResolvePending(g, p)
			continue
		}
		if g.Phase != game.PhaseTurn {
			break
		}
		randomTurn(g, p, rng)
	}
	switch g.Winner {
	case player:
		return 100
	case 1 - player:
		return 0
	}
	// partial signal: prizes delta prevents all options scoring 0 when no winner reached
	return 50 + 8*(g.Players[player].PrizesTaken-g.Players[1-player].PrizesTaken)
}

func randomTurn(g *game.Game, player int, rng *rand.Rand) {
	ps := g.Players[player]

	if idx := bestBasicInHand(g, player); idx >= 0 {
		_ = game.PlaceBenchCmd{Player: player, HandIdx: idx}.Execute(g)
	}

	if idx := energyInHand(g, player); idx >= 0 {
		_ = game.AttachEnergyCmd{Player: player, HandIdx: idx, Slot: game.ActiveSlot}.Execute(g)
	}

	if ps.Active != nil {
		best, bestDmg := -1, -1
		c := g.Card(ps.Active.TopID())
		for i, atk := range c.Attacks {
			if g.CostPaid(ps.Active, atk.Cost) {
				dmg := game.BaseDamage(atk.Damage)
				if dmg > bestDmg {
					bestDmg, best = dmg, i
				}
			}
		}
		_ = rng.Intn(2)
		if best >= 0 {
			if (game.AttackCmd{Player: player, AtkIdx: best}).Execute(g) == nil {
				return
			}
		}
	}
	_ = game.EndTurnCmd{Player: player}.Execute(g)
}
