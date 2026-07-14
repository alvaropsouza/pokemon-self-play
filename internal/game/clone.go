package game

import "math/rand"

// CloneWithSeed creates a deep copy of the game state with a new RNG seed.
// The card store is shared (immutable). Log is discarded. Used by MCTS.
func (g *Game) CloneWithSeed(seed int64) *Game {
	c := &Game{
		store:        g.store,
		rng:          rand.New(rand.NewSource(seed)),
		Current:      g.Current,
		TurnNumber:   g.TurnNumber,
		Phase:        g.Phase,
		Winner:       g.Winner,
		Stadium:      g.Stadium,
		StadiumOwner: g.StadiumOwner,
		NeedPromote:  g.NeedPromote,
	}
	for p := 0; p < 2; p++ {
		c.Players[p] = g.Players[p].cloneState()
	}
	if g.Pending != nil {
		pc := *g.Pending
		pc.Candidates = append([]int{}, g.Pending.Candidates...)
		pc.rest = append([]Op{}, g.Pending.rest...)
		c.Pending = &pc
	}
	return c
}

func (ps *PlayerState) cloneState() *PlayerState {
	c := *ps
	c.Deck = append([]string{}, ps.Deck...)
	c.Hand = append([]string{}, ps.Hand...)
	c.Discard = append([]string{}, ps.Discard...)
	c.Prizes = append([]string{}, ps.Prizes...)
	c.LostZone = append([]string{}, ps.LostZone...)
	if ps.Active != nil {
		c.Active = ps.Active.clonePkm()
	}
	c.Bench = make([]*PokemonInPlay, len(ps.Bench))
	for i, pk := range ps.Bench {
		c.Bench[i] = pk.clonePkm()
	}
	return &c
}

func (p *PokemonInPlay) clonePkm() *PokemonInPlay {
	c := *p
	c.Stack = append([]string{}, p.Stack...)
	c.Energies = append([]string{}, p.Energies...)
	return &c
}
