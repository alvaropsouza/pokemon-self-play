package game

import (
	"fmt"
	"math/rand"

	"github.com/alvaropsouza/pokemon-self-play/internal/cards"
)

// New creates a match: shuffles decks, draws initial hands resolving mulligans
// (with automatic extra draws for opponent mulligans), and leaves the game in
// PhaseSetup waiting for Active/Bench placement.
// Pass firstPlayer 0 or 1 to force; any other value triggers a random draw.
func New(store *cards.Store, decks [2][]string, seed int64, firstPlayer int) (*Game, error) {
	g := &Game{
		store:  store,
		rng:    rand.New(rand.NewSource(seed)),
		Phase:  PhaseSetup,
		Winner: -1,
	}
	for p := 0; p < 2; p++ {
		for _, id := range decks[p] {
			if store.Cards[id] == nil {
				return nil, fmt.Errorf("jogador %d: carta %q não existe na base", p+1, id)
			}
		}
		if len(decks[p]) != 60 {
			return nil, fmt.Errorf("jogador %d: deck tem %d cartas (esperado 60)", p+1, len(decks[p]))
		}
		ps := &PlayerState{Deck: append([]string{}, decks[p]...)}
		g.shuffle(ps.Deck)
		g.Players[p] = ps
	}

	if firstPlayer != 0 && firstPlayer != 1 {
		firstPlayer = g.rng.Intn(2)
		g.logf("sorteio: jogador %d começa", firstPlayer+1)
	}
	g.Current = firstPlayer

	for p := 0; p < 2; p++ {
		ps := g.Players[p]
		for {
			// copying avoids aliasing: slicing the deck would share the backing array
			// and future appends to the hand (prizes, search) would corrupt the deck
			ps.Hand = append([]string(nil), ps.Deck[:7]...)
			ps.Deck = ps.Deck[7:]
			if g.handHasBasic(ps.Hand) {
				break
			}
			ps.Mulligans++
			g.logf("jogador %d: mulligan (%d)", p+1, ps.Mulligans)
			ps.Deck = append(ps.Deck, ps.Hand...)
			ps.Hand = nil
			g.shuffle(ps.Deck)
		}
	}
	for p := 0; p < 2; p++ {
		if n := g.Players[1-p].Mulligans; n > 0 {
			for i := 0; i < n; i++ {
				g.drawCard(p)
			}
			g.logf("jogador %d: +%d carta(s) por mulligan do oponente (mão: %d)", p+1, n, len(g.Players[p].Hand))
		}
	}
	return g, nil
}

func (g *Game) Card(id string) *cards.Card { return g.store.Cards[id] }

func (g *Game) logf(format string, args ...any) {
	g.Log = append(g.Log, fmt.Sprintf(format, args...))
}

func (g *Game) event(kind string, p int) {
	g.Events = append(g.Events, Event{Kind: kind, Player: p})
}

func (g *Game) shuffle(pile []string) {
	g.rng.Shuffle(len(pile), func(i, j int) { pile[i], pile[j] = pile[j], pile[i] })
}

func (g *Game) flip() bool { return g.rng.Intn(2) == 0 }

func (g *Game) handHasBasic(hand []string) bool {
	for _, id := range hand {
		if c := g.store.Cards[id]; c != nil && c.IsBasicPokemon() {
			return true
		}
	}
	return false
}

func (g *Game) drawCard(p int) bool {
	ps := g.Players[p]
	if len(ps.Deck) == 0 {
		return false
	}
	ps.Hand = append(ps.Hand, ps.Deck[0])
	ps.Deck = ps.Deck[1:]
	return true
}

func (g *Game) handCard(p, handIdx int) (*cards.Card, error) {
	ps := g.Players[p]
	if handIdx < 0 || handIdx >= len(ps.Hand) {
		return nil, fmt.Errorf("índice de mão inválido: %d", handIdx)
	}
	return g.store.Cards[ps.Hand[handIdx]], nil
}

func (g *Game) removeFromHand(p, handIdx int) string {
	ps := g.Players[p]
	id := ps.Hand[handIdx]
	ps.Hand = append(ps.Hand[:handIdx], ps.Hand[handIdx+1:]...)
	return id
}

func (g *Game) requireTurn(p int) error {
	if g.Phase != PhaseTurn {
		return fmt.Errorf("partida não está em andamento")
	}
	if g.Current != p {
		return fmt.Errorf("não é o turno do jogador %d", p+1)
	}
	if g.NeedPromote[0] || g.NeedPromote[1] {
		return fmt.Errorf("promoção pendente antes de continuar")
	}
	if g.Pending != nil {
		return fmt.Errorf("escolha de busca pendente antes de continuar")
	}
	return nil
}

func (g *Game) target(p, slot int) (*PokemonInPlay, error) {
	ps := g.Players[p]
	if slot == ActiveSlot {
		if ps.Active == nil {
			return nil, fmt.Errorf("jogador %d sem Ativo", p+1)
		}
		return ps.Active, nil
	}
	if slot < 0 || slot >= len(ps.Bench) {
		return nil, fmt.Errorf("posição de banco inválida: %d", slot)
	}
	return ps.Bench[slot], nil
}
