package game

import (
	"fmt"

	"github.com/alvaropsouza/pokemon-self-play/internal/cards"
)

func (g *Game) AttachEnergy(p, handIdx, slot int) error {
	if err := g.requireTurn(p); err != nil {
		return err
	}
	ps := g.Players[p]
	if ps.EnergyAttached {
		return fmt.Errorf("já ligou Energia neste turno")
	}
	c, err := g.handCard(p, handIdx)
	if err != nil {
		return err
	}
	if c.Category != cards.CategoryEnergy {
		return fmt.Errorf("%s não é Energia", c.Name.EN)
	}
	t, err := g.target(p, slot)
	if err != nil {
		return err
	}
	t.Energies = append(t.Energies, c.ID)
	g.removeFromHand(p, handIdx)
	ps.EnergyAttached = true
	g.logf("jogador %d: liga %s em %s", p+1, c.Name.EN, g.Card(t.TopID()).Name.EN)
	g.emit(Trigger{Kind: TrigEnergyAttached, Player: p, Slot: slot})
	return nil
}

func (g *Game) Evolve(p, handIdx, slot int) error {
	if err := g.requireTurn(p); err != nil {
		return err
	}
	ps := g.Players[p]
	if ps.TurnsTaken == 0 {
		return fmt.Errorf("não pode evoluir no seu primeiro turno")
	}
	c, err := g.handCard(p, handIdx)
	if err != nil {
		return err
	}
	if c.Category != cards.CategoryPokemon || c.EvolveFrom.EN == "" {
		return fmt.Errorf("%s não é carta de evolução", c.Name.EN)
	}
	t, err := g.target(p, slot)
	if err != nil {
		return err
	}
	cur := g.Card(t.TopID())
	if c.EvolveFrom.EN != cur.Name.EN {
		return fmt.Errorf("%s não evolui de %s", c.Name.EN, cur.Name.EN)
	}
	if t.EnteredTurn == g.TurnNumber {
		return fmt.Errorf("%s entrou em jogo neste turno", cur.Name.EN)
	}
	if t.EvolvedTurn == g.TurnNumber {
		return fmt.Errorf("%s já evoluiu neste turno", cur.Name.EN)
	}
	t.Stack = append([]string{c.ID}, t.Stack...)
	t.EvolvedTurn = g.TurnNumber
	t.clearConditions()
	g.removeFromHand(p, handIdx)
	g.logf("jogador %d: %s evolui para %s", p+1, cur.Name.EN, c.Name.EN)
	g.emit(Trigger{Kind: TrigEvolved, Player: p, Slot: slot})
	return nil
}

func (g *Game) AttachTool(p, handIdx, slot int) error {
	if err := g.requireTurn(p); err != nil {
		return err
	}
	c, err := g.handCard(p, handIdx)
	if err != nil {
		return err
	}
	if c.Category != cards.CategoryTrainer || c.TrainerType != "Tool" {
		return fmt.Errorf("%s não é Ferramenta", c.Name.EN)
	}
	t, err := g.target(p, slot)
	if err != nil {
		return err
	}
	if t.Tool != "" {
		return fmt.Errorf("%s já tem Ferramenta", g.Card(t.TopID()).Name.EN)
	}
	t.Tool = c.ID
	g.removeFromHand(p, handIdx)
	g.logf("jogador %d: liga Ferramenta %s em %s (efeito: arbitragem manual)", p+1, c.Name.EN, g.Card(t.TopID()).Name.EN)
	return nil
}

func (g *Game) checkDiscardCost(p int, c *cards.Card) error {
	for _, op := range CompileEffect(c.Effect.EN).Ops {
		if op.Kind == OpDiscardFromHand && op.Cost && len(g.Players[p].Hand)-1 < op.N {
			return fmt.Errorf("%s exige descartar %d outras cartas da mão", c.Name.EN, op.N)
		}
	}
	return nil
}

func (g *Game) PlayItem(p, handIdx int) error {
	if err := g.requireTurn(p); err != nil {
		return err
	}
	c, err := g.handCard(p, handIdx)
	if err != nil {
		return err
	}
	if c.Category != cards.CategoryTrainer || c.TrainerType != "Item" {
		return fmt.Errorf("%s não é Item", c.Name.EN)
	}
	if err := g.checkDiscardCost(p, c); err != nil {
		return err
	}
	g.removeFromHand(p, handIdx)
	g.Players[p].Discard = append(g.Players[p].Discard, c.ID)
	if !g.applyTrainerEffect(p, c) {
		g.logf("jogador %d: joga Item %s → efeito manual: %s", p+1, c.Name.EN, c.Effect.EN)
	} else {
		g.logf("jogador %d: joga Item %s", p+1, c.Name.EN)
	}
	return nil
}

func (g *Game) PlaySupporter(p, handIdx int) error {
	if err := g.requireTurn(p); err != nil {
		return err
	}
	ps := g.Players[p]
	if ps.SupporterPlayed {
		return fmt.Errorf("já jogou Suporte neste turno")
	}
	if g.TurnNumber == 1 && ps.TurnsTaken == 0 {
		return fmt.Errorf("quem começa não joga Suporte no primeiro turno")
	}
	c, err := g.handCard(p, handIdx)
	if err != nil {
		return err
	}
	if c.Category != cards.CategoryTrainer || c.TrainerType != "Supporter" {
		return fmt.Errorf("%s não é Suporte", c.Name.EN)
	}
	if err := g.checkDiscardCost(p, c); err != nil {
		return err
	}
	g.removeFromHand(p, handIdx)
	ps.Discard = append(ps.Discard, c.ID)
	ps.SupporterPlayed = true
	if !g.applyTrainerEffect(p, c) {
		g.logf("jogador %d: joga Suporte %s → efeito manual: %s", p+1, c.Name.EN, c.Effect.EN)
	} else {
		g.logf("jogador %d: joga Suporte %s", p+1, c.Name.EN)
	}
	return nil
}

func (g *Game) PlayStadium(p, handIdx int) error {
	if err := g.requireTurn(p); err != nil {
		return err
	}
	ps := g.Players[p]
	if ps.StadiumPlayed {
		return fmt.Errorf("já jogou Estádio neste turno")
	}
	c, err := g.handCard(p, handIdx)
	if err != nil {
		return err
	}
	if c.Category != cards.CategoryTrainer || c.TrainerType != "Stadium" {
		return fmt.Errorf("%s não é Estádio", c.Name.EN)
	}
	if g.Stadium != "" && g.Card(g.Stadium).Name.EN == c.Name.EN {
		return fmt.Errorf("%s já está em jogo", c.Name.EN)
	}
	if g.Stadium != "" {
		g.Players[g.StadiumOwner].Discard = append(g.Players[g.StadiumOwner].Discard, g.Stadium)
	}
	g.removeFromHand(p, handIdx)
	g.Stadium = c.ID
	g.StadiumOwner = p
	ps.StadiumPlayed = true
	g.logf("jogador %d: joga Estádio %s", p+1, c.Name.EN)
	return nil
}

func (g *Game) UseAbility(player, abilitySlot, targetSlot int) error {
	if err := g.requireTurn(player); err != nil {
		return err
	}
	pk, err := g.target(player, abilitySlot)
	if err != nil {
		return err
	}
	cardID := pk.TopID()
	h, ok := abilityDB[cardID]
	if !ok {
		return fmt.Errorf("%s não tem Habilidade implementada", g.Card(cardID).Name.EN)
	}
	ps := g.Players[player]
	if ps.AbilitiesUsed == nil {
		ps.AbilitiesUsed = map[int]bool{}
	}
	if ps.AbilitiesUsed[abilitySlot] {
		return fmt.Errorf("Habilidade já usada neste turno")
	}
	if err := h(g, player, abilitySlot, targetSlot); err != nil {
		return err
	}
	ps.AbilitiesUsed[abilitySlot] = true
	g.logf("jogador %d: usa Habilidade de %s", player+1, g.Card(cardID).Name.EN)
	return nil
}

func (g *Game) Retreat(p, benchIdx int, energyIdxs []int) error {
	if err := g.requireTurn(p); err != nil {
		return err
	}
	ps := g.Players[p]
	if ps.Retreated {
		return fmt.Errorf("já recuou neste turno")
	}
	if ps.Active == nil {
		return fmt.Errorf("sem Ativo")
	}
	if ps.Active.Rot == CondAsleep || ps.Active.Rot == CondParalyzed {
		return fmt.Errorf("Pokémon %s não pode recuar", condPT[ps.Active.Rot])
	}
	if benchIdx < 0 || benchIdx >= len(ps.Bench) {
		return fmt.Errorf("posição de banco inválida: %d", benchIdx)
	}
	cost := g.Card(ps.Active.TopID()).Retreat
	if len(energyIdxs) != cost {
		return fmt.Errorf("custo de Recuo é %d Energia(s), %d indicada(s)", cost, len(energyIdxs))
	}
	if err := g.discardEnergies(p, ps.Active, energyIdxs); err != nil {
		return err
	}
	old := ps.Active
	old.clearConditions()
	ps.Active = ps.Bench[benchIdx]
	ps.Bench[benchIdx] = old
	ps.Retreated = true
	g.logf("jogador %d: recua %s, promove %s", p+1, g.Card(old.TopID()).Name.EN, g.Card(ps.Active.TopID()).Name.EN)
	return nil
}

func (g *Game) discardEnergies(p int, t *PokemonInPlay, idxs []int) error {
	seen := map[int]bool{}
	for _, i := range idxs {
		if i < 0 || i >= len(t.Energies) || seen[i] {
			return fmt.Errorf("índice de Energia inválido/repetido: %d", i)
		}
		seen[i] = true
	}
	var kept []string
	for i, id := range t.Energies {
		if seen[i] {
			g.Players[p].Discard = append(g.Players[p].Discard, id)
		} else {
			kept = append(kept, id)
		}
	}
	t.Energies = kept
	return nil
}

func (g *Game) Promote(p, benchIdx int) error {
	if g.Phase != PhaseTurn || !g.NeedPromote[p] {
		return fmt.Errorf("jogador %d não tem promoção pendente", p+1)
	}
	ps := g.Players[p]
	if benchIdx < 0 || benchIdx >= len(ps.Bench) {
		return fmt.Errorf("posição de banco inválida: %d", benchIdx)
	}
	ps.Active = ps.Bench[benchIdx]
	ps.Bench = append(ps.Bench[:benchIdx], ps.Bench[benchIdx+1:]...)
	g.NeedPromote[p] = false
	g.logf("jogador %d: promove %s", p+1, g.Card(ps.Active.TopID()).Name.EN)
	return nil
}

func (g *Game) EndTurn(p int) error {
	if err := g.requireTurn(p); err != nil {
		return err
	}
	g.finishTurn()
	return nil
}

func (g *Game) finishTurn() {
	ps := g.Players[g.Current]
	ps.TurnsTaken++
	ps.EnergyAttached = false
	ps.SupporterPlayed = false
	ps.StadiumPlayed = false
	ps.Retreated = false
	ps.AbilitiesUsed = nil

	g.emit(Trigger{Kind: TrigTurnEnded, Player: g.Current, Slot: ActiveSlot})
	g.checkup()
	if g.Phase == PhaseFinished {
		return
	}

	g.Current = 1 - g.Current
	g.TurnNumber++
	g.logf("turno %d: jogador %d", g.TurnNumber, g.Current+1)
	g.mandatoryDraw(g.Current)
}

func (g *Game) mandatoryDraw(p int) {
	if !g.drawCard(p) {
		g.logf("jogador %d não pode comprar: deck-out", p+1)
		g.declareWinner(1 - p)
	}
}

func (g *Game) declareWinner(p int) {
	if g.Phase == PhaseFinished {
		return
	}
	g.Phase = PhaseFinished
	g.Winner = p
	if p == -2 {
		g.logf("condições de vitória simultâneas: Sudden Death (nova partida com 1 Prêmio)")
	} else {
		g.logf("jogador %d vence", p+1)
	}
}
