package bot

// Flat Monte Carlo search para decisão de ataque.
//
// Em vez de jogar o maior dano imediato (pilot anterior), o bot agora:
//  1. Enumera os ataques pagos + "passar o turno".
//  2. Para cada opção: clona o estado, aplica, roda MCBudget jogos aleatórios até o fim.
//  3. Retorna o índice do ataque com maior taxa de vitória simulada (-1 = passar).
//
// Custo por turno do bot: O(MCBudget × profundidade_média_do_jogo).
// MCBudget=100 e profundidade≈20 turnos → ~2000 ações por decisão (rápido).
// ponytail: sem árvore UCB1 — flat MC suficiente; add tree quando budget>500 measurably helps.

import (
	"math/rand"

	"github.com/alvaropsouza/pokemon-self-play/internal/game"
)

// MCBudget é o número de simulações por opção de ataque.
// Valores maiores = decisão melhor, mas mais lenta.
const MCBudget = 100

// MCPickAttack retorna o índice do ataque com maior win rate simulada, ou -1
// se passar o turno for melhor. Usa MCBudget simulações por opção.
func MCPickAttack(g *game.Game, player int) int {
	pk := g.Players[player].Active
	if pk == nil {
		return -1
	}
	c := g.Card(pk.TopID())

	type option struct {
		atk   int // -1 = end turn
		score int
	}

	var opts []option
	for i := range c.Attacks {
		if g.CostPaid(pk, c.Attacks[i].Cost) {
			opts = append(opts, option{atk: i})
		}
	}
	if len(opts) == 0 {
		return -1 // nenhum ataque pago: passar
	}
	// "Passar" entra por último: em empate de score (comum quando os rollouts
	// não alcançam um vencedor), o desempate estrito fica com um ataque.
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

// applyOption aplica a opção (ataque ou endTurn) ao clone.
func applyOption(g *game.Game, player, atk int) {
	if atk >= 0 {
		_ = g.Attack(player, atk)
	} else {
		_ = g.EndTurn(player)
	}
}

// rollout joga o jogo do clone até maxTurns com heurística mínima e devolve
// um score para player: 100 = vitória, 0 = derrota; jogo inacabado vale 50
// ± vantagem de prêmios (sinal parcial — evita que rollouts longos demais
// zerem todas as opções e o bot nunca ataque).
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
	return 50 + 8*(g.Players[player].PrizesTaken-g.Players[1-player].PrizesTaken)
}

// randomTurn executa um turno com heurística mínima (sem recursão MC).
func randomTurn(g *game.Game, player int, rng *rand.Rand) {
	ps := g.Players[player]

	// Banco: básico disponível.
	if idx := bestBasicInHand(g, player); idx >= 0 {
		_ = g.PlaceBench(player, idx)
	}

	// Energia no Ativo.
	if idx := energyInHand(g, player); idx >= 0 {
		_ = g.AttachEnergy(player, idx, game.ActiveSlot)
	}

	// Ataca com maior dano pago; senão passa.
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
		_ = rng.Intn(2) // consume rng for variety
		if best >= 0 {
			if g.Attack(player, best) == nil {
				return
			}
		}
	}
	_ = g.EndTurn(player)
}
