package game

import (
	"fmt"
	"strings"

	"github.com/alvaropsouza/pokemon-self-play/internal/cards"
)

// Attack declara o ataque attackIdx do Pokémon Ativo e encerra o turno.
// Fluxo (seção 7 do CLAUDE.md): validações → moeda de Confusão → dano base com
// Fraqueza (×2) e Resistência (−30) no Ativo defensor → nocautes/Prêmios →
// checkup e passagem de turno. Ataques com texto de efeito têm o efeito
// registrado no log para arbitragem manual (aplicar via helpers antes/depois).
func (g *Game) Attack(p, attackIdx int) error {
	if err := g.requireTurn(p); err != nil {
		return err
	}
	if g.TurnNumber == 1 {
		return fmt.Errorf("quem começa não ataca no primeiro turno")
	}
	ps := g.Players[p]
	if ps.Active == nil {
		return fmt.Errorf("sem Ativo")
	}
	if ps.Active.Rot == CondAsleep || ps.Active.Rot == CondParalyzed {
		return fmt.Errorf("Pokémon %s não pode atacar", condPT[ps.Active.Rot])
	}
	atkCard := g.Card(ps.Active.TopID())
	if attackIdx < 0 || attackIdx >= len(atkCard.Attacks) {
		return fmt.Errorf("ataque inválido: %d", attackIdx)
	}
	atk := atkCard.Attacks[attackIdx]
	if !g.CostPaid(ps.Active, atk.Cost) {
		return fmt.Errorf("custo de %s não está pago", atk.Name.EN)
	}

	// Confusão: moeda ao declarar; coroa = falha e 3 contadores em si mesmo.
	if ps.Active.Rot == CondConfused {
		if !g.flip() {
			g.logf("jogador %d: Confuso, coroa — ataque falha, 30 em si mesmo", p+1)
			ps.Active.Damage += 30
			g.resolveKnockouts()
			if g.Phase == PhaseTurn {
				g.finishTurn()
			}
			return nil
		}
		g.logf("jogador %d: Confuso, cara — ataca normalmente", p+1)
	}

	def := g.Players[1-p]
	activeSnapshot := ps.Active // capture before any KO resolution
	dmg := g.attackDamage(atkCard, atk, def.Active)
	dmg += ExtraAttackDamage(g, p, atk, ps.Active)
	g.logf("jogador %d: %s usa %s → %d de dano", p+1, atkCard.Name.EN, atk.Name.EN, dmg)
	if def.Active != nil && dmg > 0 {
		def.Active.Damage += dmg
	}
	g.applyAttackEffect(p, atk, activeSnapshot)
	g.resolveKnockouts()
	if g.Phase == PhaseTurn {
		g.finishTurn()
	}
	return nil
}

// attackDamage calcula o dano no Ativo defensor: base impresso, Fraqueza ×2,
// Resistência −30. Modificadores de efeitos são arbitragem manual.
func (g *Game) attackDamage(attacker *cards.Card, atk cards.Attack, defender *PokemonInPlay) int {
	dmg := BaseDamage(atk.Damage)
	if dmg == 0 || defender == nil {
		return dmg
	}
	defCard := g.Card(defender.TopID())
	atkType := ""
	if len(attacker.Types) > 0 {
		atkType = attacker.Types[0]
	}
	for _, w := range defCard.Weaknesses {
		if w.Type == atkType {
			dmg *= 2
			g.logf("Fraqueza %s: dano ×2", w.Type)
		}
	}
	for _, r := range defCard.Resistances {
		if r.Type == atkType {
			dmg -= 30
			g.logf("Resistência %s: dano −30", r.Type)
		}
	}
	if dmg < 0 {
		dmg = 0
	}
	return dmg
}

// BaseDamage extrai o número do dano impresso ("30", "30+", "20×" → 30, 30, 20).
func BaseDamage(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	return n
}

// resolveKnockouts trata nocautes dos dois Ativos (simultâneos inclusive):
// descarte da pilha, Prêmios (sempre para o oponente do nocauteado), promoção
// pendente e condições de vitória.
func (g *Game) resolveKnockouts() {
	koed := [2]bool{}
	for p := 0; p < 2; p++ {
		a := g.Players[p].Active
		if a != nil && a.Damage >= g.Card(a.TopID()).HP {
			koed[p] = true
		}
	}
	for p := 0; p < 2; p++ {
		if !koed[p] {
			continue
		}
		ps := g.Players[p]
		name := g.Card(ps.Active.TopID()).Name.EN
		prizes := PrizeValue(g.Card(ps.Active.TopID()))
		ps.Discard = append(ps.Discard, ps.Active.allCardIDs()...)
		ps.Active = nil
		g.logf("%s (jogador %d) é Nocauteado — jogador %d pega %d Prêmio(s)", name, p+1, 2-p, prizes)
		g.takePrizes(1-p, prizes)
		g.emit(Trigger{Kind: TrigKnockOut, Player: p, Slot: ActiveSlot})
	}
	// Verifica vitórias após todos os nocautes (simultâneos resolvem juntos).
	var winners []int
	for p := 0; p < 2; p++ {
		win := len(g.Players[p].Prizes) == 0
		opp := g.Players[1-p]
		if opp.Active == nil && len(opp.Bench) == 0 {
			win = true
		}
		if win {
			winners = append(winners, p)
		}
	}
	switch len(winners) {
	case 1:
		g.declareWinner(winners[0])
		return
	case 2:
		g.declareWinner(-2) // Sudden Death
		return
	}
	for p := 0; p < 2; p++ {
		if koed[p] {
			g.NeedPromote[p] = true
		}
	}
}

// takePrizes move n cartas de Prêmio para a mão do jogador p.
func (g *Game) takePrizes(p, n int) {
	ps := g.Players[p]
	for i := 0; i < n && len(ps.Prizes) > 0; i++ {
		ps.Hand = append(ps.Hand, ps.Prizes[0])
		ps.Prizes = ps.Prizes[1:]
		ps.PrizesTaken++
	}
}

// PrizeValue devolve quantos Prêmios o nocaute desta carta concede.
// Heurística pelo nome (a base TCGdex não expõe a Rule Box): Mega ex = 3,
// ex/V/VSTAR = 2, VMAX = 3, demais = 1. Casos exóticos: conferir a carta.
func PrizeValue(c *cards.Card) int {
	name := c.Name.EN
	switch {
	case strings.HasPrefix(name, "Mega ") && strings.HasSuffix(name, " ex"):
		return 3
	case strings.HasSuffix(name, " VMAX"):
		return 3
	case strings.HasSuffix(name, " ex"), strings.HasSuffix(name, " EX"),
		strings.HasSuffix(name, " V"), strings.HasSuffix(name, " VSTAR"):
		return 2
	}
	return 1
}
