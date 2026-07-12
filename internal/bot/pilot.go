package bot

import (
	"github.com/alvaropsouza/pokemon-self-play/internal/cards"
	"github.com/alvaropsouza/pokemon-self-play/internal/game"
)

// Setup faz a colocação inicial do bot: melhor básico como Ativo (maior HP),
// até 2 básicos no banco.
func Setup(g *game.Game, player int) error {
	if idx := bestBasicInHand(g, player); idx >= 0 {
		if err := g.PlaceActive(player, idx); err != nil {
			return err
		}
	}
	for i := 0; i < 2; i++ {
		idx := bestBasicInHand(g, player)
		if idx < 0 {
			break
		}
		if err := g.PlaceBench(player, idx); err != nil {
			break
		}
	}
	return g.FinishSetup(player)
}

// PromoteIfNeeded resolve promoção pendente: Pokémon com mais HP restante.
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
	_ = g.Promote(player, best)
}

// TakeTurn executa o turno completo do bot (após a compra automática do motor).
func TakeTurn(g *game.Game, player int) {
	PromoteIfNeeded(g, player)
	if g.Phase != game.PhaseTurn || g.Current != player {
		return
	}
	ps := g.Players[player]

	// Banco: baixa básicos até ter 3 Pokémon de reserva.
	for len(ps.Bench) < 3 {
		idx := bestBasicInHand(g, player)
		if idx < 0 || g.PlaceBench(player, idx) != nil {
			break
		}
	}

	// Evolui tudo que puder (Ativo primeiro).
	for _, slot := range slots(g, player) {
		for i := 0; i < len(ps.Hand); i++ {
			if g.Evolve(player, i, slot) == nil {
				break
			}
		}
	}

	// Energia: no Ativo se o melhor ataque dele ainda não está pago; senão no banco.
	if idx := energyInHand(g, player); idx >= 0 {
		target := game.ActiveSlot
		if ps.Active != nil && bestPaidAttack(g, ps.Active) >= 0 && len(ps.Bench) > 0 {
			target = 0
		}
		_ = g.AttachEnergy(player, idx, target)
	}

	// Ataca com o maior dano pago; senão passa.
	if ps.Active != nil {
		if atk := bestPaidAttack(g, ps.Active); atk >= 0 {
			if g.Attack(player, atk) == nil {
				return
			}
		}
	}
	_ = g.EndTurn(player)
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
		if c.Category == cards.CategoryPokemon && c.Stage == "Basic" && c.HP > bestHP {
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

// bestPaidAttack devolve o índice do ataque pago de maior dano (≥1), ou -1.
// Ataques de custo vazio com dano 0 (efeito puro) são ignorados.
func bestPaidAttack(g *game.Game, pk *game.PokemonInPlay) int {
	c := g.Card(pk.TopID())
	best, bestDmg := -1, 0
	for i, atk := range c.Attacks {
		if !g.CostPaid(pk, atk.Cost) {
			continue
		}
		dmg := game.BaseDamage(atk.Damage)
		if dmg > bestDmg {
			best, bestDmg = i, dmg
		}
	}
	return best
}
