package game

import "fmt"

func (g *Game) ApplyDamage(p, slot, amount int) error {
	t, err := g.target(p, slot)
	if err != nil {
		return err
	}
	t.Damage += amount
	g.logf("arbitragem: %d de dano em %s (jogador %d)", amount, g.Card(t.TopID()).Name.EN, p+1)
	g.resolveKnockouts()
	return nil
}

func (g *Game) Heal(p, slot, amount int) error {
	t, err := g.target(p, slot)
	if err != nil {
		return err
	}
	t.Damage -= amount
	if t.Damage < 0 {
		t.Damage = 0
	}
	g.logf("arbitragem: cura %d em %s (jogador %d)", amount, g.Card(t.TopID()).Name.EN, p+1)
	return nil
}

func (g *Game) SetCondition(p int, cond string) error {
	ps := g.Players[p]
	if ps.Active == nil {
		return fmt.Errorf("jogador %d sem Ativo", p+1)
	}
	switch Condition(cond) {
	case CondAsleep, CondConfused, CondParalyzed:
		ps.Active.Rot = Condition(cond)
	default:
		switch cond {
		case "poisoned":
			ps.Active.Poisoned = true
		case "burned":
			ps.Active.Burned = true
		default:
			return fmt.Errorf("condição desconhecida: %q", cond)
		}
	}
	g.logf("arbitragem: %s (jogador %d) está %s", g.Card(ps.Active.TopID()).Name.EN, p+1, cond)
	return nil
}

func (g *Game) DrawCards(p, n int) {
	drawn := 0
	for i := 0; i < n && g.drawCard(p); i++ {
		drawn++
	}
	g.logf("arbitragem: jogador %d compra %d carta(s)", p+1, drawn)
}

func (g *Game) ShuffleDeck(p int) {
	g.shuffle(g.Players[p].Deck)
	g.logf("arbitragem: deck do jogador %d embaralhado", p+1)
}

func (g *Game) SwitchActive(p, benchIdx int) error {
	ps := g.Players[p]
	if ps.Active == nil {
		return fmt.Errorf("jogador %d sem Ativo", p+1)
	}
	if benchIdx < 0 || benchIdx >= len(ps.Bench) {
		return fmt.Errorf("posição de banco inválida: %d", benchIdx)
	}
	old := ps.Active
	old.clearConditions()
	ps.Active = ps.Bench[benchIdx]
	ps.Bench[benchIdx] = old
	g.logf("arbitragem: jogador %d troca Ativo para %s", p+1, g.Card(ps.Active.TopID()).Name.EN)
	return nil
}
