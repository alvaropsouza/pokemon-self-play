package game

import "fmt"

func (g *Game) PlaceActive(p, handIdx int) error {
	if g.Phase != PhaseSetup {
		return fmt.Errorf("fora da fase de setup")
	}
	ps := g.Players[p]
	if ps.Active != nil {
		return fmt.Errorf("jogador %d já tem Ativo", p+1)
	}
	c, err := g.handCard(p, handIdx)
	if err != nil {
		return err
	}
	if !c.IsBasicPokemon() {
		return fmt.Errorf("%s não é Pokémon Básico", c.Name.EN)
	}
	ps.Active = &PokemonInPlay{Stack: []string{c.ID}}
	g.removeFromHand(p, handIdx)
	return nil
}

func (g *Game) PlaceBench(p, handIdx int) error {
	ps := g.Players[p]
	switch g.Phase {
	case PhaseSetup:
		if ps.Active == nil {
			return fmt.Errorf("coloque o Ativo antes do Banco")
		}
	case PhaseTurn:
		if err := g.requireTurn(p); err != nil {
			return err
		}
	default:
		return fmt.Errorf("partida encerrada")
	}
	if len(ps.Bench) >= 5 {
		return fmt.Errorf("banco cheio (5)")
	}
	c, err := g.handCard(p, handIdx)
	if err != nil {
		return err
	}
	if !c.IsBasicPokemon() {
		return fmt.Errorf("%s não é Pokémon Básico", c.Name.EN)
	}
	ps.Bench = append(ps.Bench, &PokemonInPlay{Stack: []string{c.ID}, EnteredTurn: g.TurnNumber})
	g.removeFromHand(p, handIdx)
	g.logf("jogador %d: %s no Banco", p+1, c.Name.EN)
	if g.Phase == PhaseTurn {
		g.emit(Trigger{Kind: TrigBenchPlaced, Player: p, Slot: len(ps.Bench) - 1})
	}
	return nil
}

func (g *Game) FinishSetup(p int) error {
	if g.Phase != PhaseSetup {
		return fmt.Errorf("fora da fase de setup")
	}
	ps := g.Players[p]
	if ps.Active == nil {
		return fmt.Errorf("jogador %d precisa de um Ativo", p+1)
	}
	ps.setupReady = true
	if !g.Players[0].setupReady || !g.Players[1].setupReady {
		return nil
	}
	for i := 0; i < 2; i++ {
		s := g.Players[i]
		// copying avoids aliasing between prizes and deck slices
		s.Prizes = append([]string(nil), s.Deck[:6]...)
		s.Deck = s.Deck[6:]
	}
	g.Phase = PhaseTurn
	g.TurnNumber = 1
	g.logf("turno 1: jogador %d", g.Current+1)
	g.mandatoryDraw(g.Current)
	return nil
}
