package bot

import (
	"github.com/alvaropsouza/pokemon-self-play/internal/cards"
	"github.com/alvaropsouza/pokemon-self-play/internal/game"
)

// Pilot joga o turno do bot com heurística simples (PLANO.md etapa 2):
// promover se pendente, baixar básicos, evoluir, ligar energia no Ativo,
// atacar com o maior dano pago; senão passa o turno.
type Pilot struct {
	Player int
}

// Setup faz a colocação inicial do bot: melhor básico como Ativo (maior HP),
// até 2 básicos no banco.
func (b *Pilot) Setup(g *game.Game) error {
	if idx := b.bestBasicInHand(g); idx >= 0 {
		if err := g.PlaceActive(b.Player, idx); err != nil {
			return err
		}
	}
	for i := 0; i < 2; i++ {
		idx := b.bestBasicInHand(g)
		if idx < 0 {
			break
		}
		if err := g.PlaceBench(b.Player, idx); err != nil {
			break
		}
	}
	return g.FinishSetup(b.Player)
}

// PromoteIfNeeded resolve promoção pendente: Pokémon com mais HP restante.
func (b *Pilot) PromoteIfNeeded(g *game.Game) {
	if !g.NeedPromote[b.Player] {
		return
	}
	ps := g.Players[b.Player]
	best, bestHP := 0, -1
	for i, pk := range ps.Bench {
		hp := g.Card(pk.TopID()).HP - pk.Damage
		if hp > bestHP {
			best, bestHP = i, hp
		}
	}
	_ = g.Promote(b.Player, best)
}

// TakeTurn executa o turno completo do bot (após a compra automática do motor).
// Termina com ataque ou passagem de turno.
func (b *Pilot) TakeTurn(g *game.Game) {
	b.PromoteIfNeeded(g)
	if g.Phase != game.PhaseTurn || g.Current != b.Player {
		return
	}
	ps := g.Players[b.Player]

	// Banco: baixa básicos até ter 3 Pokémon de reserva.
	for len(ps.Bench) < 3 {
		idx := b.bestBasicInHand(g)
		if idx < 0 || g.PlaceBench(b.Player, idx) != nil {
			break
		}
	}

	// Evolui tudo que puder (Ativo primeiro).
	for _, slot := range b.slots(g) {
		for i := 0; i < len(ps.Hand); i++ {
			if g.Evolve(b.Player, i, slot) == nil {
				break
			}
		}
	}

	// Energia: no Ativo se o melhor ataque dele ainda não está pago; senão no banco.
	if idx := b.energyInHand(g); idx >= 0 {
		target := game.ActiveSlot
		if ps.Active != nil && b.bestPaidAttack(g, ps.Active) >= 0 && len(ps.Bench) > 0 {
			target = 0
		}
		_ = g.AttachEnergy(b.Player, idx, target)
	}

	// Ataca com o maior dano pago; senão passa.
	if ps.Active != nil {
		if atk := b.bestPaidAttack(g, ps.Active); atk >= 0 {
			if g.Attack(b.Player, atk) == nil {
				return
			}
		}
	}
	_ = g.EndTurn(b.Player)
}

// slots lista Ativo + banco do bot.
func (b *Pilot) slots(g *game.Game) []int {
	out := []int{game.ActiveSlot}
	for i := range g.Players[b.Player].Bench {
		out = append(out, i)
	}
	return out
}

// bestBasicInHand devolve o índice do básico de maior HP na mão (-1 se nenhum).
func (b *Pilot) bestBasicInHand(g *game.Game) int {
	best, bestHP := -1, -1
	for i, id := range g.Players[b.Player].Hand {
		c := g.Card(id)
		if c.Category == cards.CategoryPokemon && c.Stage == "Basic" && c.HP > bestHP {
			best, bestHP = i, c.HP
		}
	}
	return best
}

func (b *Pilot) energyInHand(g *game.Game) int {
	for i, id := range g.Players[b.Player].Hand {
		if g.Card(id).Category == cards.CategoryEnergy {
			return i
		}
	}
	return -1
}

// bestPaidAttack devolve o índice do ataque pago de maior dano (-1 se nenhum).
// Usa uma tentativa a seco: valida custo pelas energias ligadas.
func (b *Pilot) bestPaidAttack(g *game.Game, pk *game.PokemonInPlay) int {
	c := g.Card(pk.TopID())
	best, bestDmg := -1, -1
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
