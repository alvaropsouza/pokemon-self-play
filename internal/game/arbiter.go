package game

import "fmt"

// Helpers de arbitragem manual: aplicam efeitos de texto de cartas (ataques,
// Treinadores, Habilidades) que o motor não interpreta automaticamente.
// Todos registram no log e resolvem nocautes quando aplicável.

// ApplyDamage aplica dano/contadores a um Pokémon do jogador p (efeitos de
// carta). Não aplica Fraqueza/Resistência — "colocar contadores" as ignora;
// para dano de ataque com F/R o cálculo já ocorre em Attack.
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

// Heal remove dano de um Pokémon do jogador p.
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

// SetCondition aplica uma Condição Especial ao Ativo do jogador p.
// Adormecido/Confuso/Paralisado substituem a rotacional anterior;
// Envenenado/Queimado coexistem.
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

// DrawCards compra n cartas para o jogador p (efeito de carta; não é a compra
// obrigatória — deck vazio aqui só interrompe, não perde).
func (g *Game) DrawCards(p, n int) {
	drawn := 0
	for i := 0; i < n && g.drawCard(p); i++ {
		drawn++
	}
	g.logf("arbitragem: jogador %d compra %d carta(s)", p+1, drawn)
}

// ShuffleDeck embaralha o deck do jogador p (após busca, por exemplo).
func (g *Game) ShuffleDeck(p int) {
	g.shuffle(g.Players[p].Deck)
	g.logf("arbitragem: deck do jogador %d embaralhado", p+1)
}

// SwitchActive troca o Ativo do jogador p com o Banco (efeito de carta como
// Switch/Boss's Orders — sem custo de recuo). Remove Condições Especiais do
// que sai.
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
