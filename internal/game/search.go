package game

import (
	"fmt"
	"strings"

	"github.com/alvaropsouza/pokemon-self-play/internal/cards"
)

// Busca no deck (op "search"): efeito com escolha do jogador. O motor entra em
// estado pendente (Game.Pending), expõe os candidatos e bloqueia as demais
// ações até ResolveChoice. Buscar pode falhar de propósito (pegar menos que o
// máximo, inclusive zero) — regra de informação oculta do TCG. O deck é sempre
// embaralhado ao final.

// Find é um critério de busca; a carta é candidata se casar QUALQUER Find da op.
type Find struct {
	Category string `json:"category"`        // "Pokemon" | "Energy"
	Stage    string `json:"stage,omitempty"` // "Basic" | "" (qualquer)
	Type     string `json:"type,omitempty"`  // tipo de energia/Pokémon ("" = qualquer)
}

// PendingChoice é uma busca aguardando a escolha do jogador.
type PendingChoice struct {
	Player     int
	Dest       string // "hand" | "bench"
	Max        int    // máximo de cartas a pegar (mínimo é sempre 0)
	Candidates []int  // índices no deck do jogador
	Reveal     bool   // cartas pegas são reveladas no log
	rest       []Op   // ops restantes do efeito, executadas após a resolução
}

// matchFind informa se a carta satisfaz o critério.
func matchFind(c *cards.Card, f Find) bool {
	if c == nil || string(c.Category) != f.Category {
		return false
	}
	switch c.Category {
	case cards.CategoryPokemon:
		if f.Stage != "" && c.Stage != f.Stage {
			return false
		}
		if f.Type != "" {
			for _, t := range c.Types {
				if t == f.Type {
					return true
				}
			}
			return false
		}
		return true
	case cards.CategoryEnergy:
		// Energia básica: TCGdex usa energyType "Normal"; nome "<Tipo> Energy".
		if c.EnergyType != "Normal" && c.EnergyType != "Basic" {
			return false
		}
		return f.Type == "" || strings.HasPrefix(c.Name.EN, f.Type+" ")
	}
	return false
}

// startSearch executa uma op de busca. Devolve true se ficou pendente
// (aguardando escolha); false se auto-resolveu (sem candidatos/espaço).
func (g *Game) startSearch(p int, op Op, rest []Op) bool {
	ps := g.Players[p]
	max := op.N
	if max <= 0 {
		max = 1
	}
	if op.Dest == "bench" {
		if space := 5 - len(ps.Bench); space < max {
			max = space
		}
	}

	var cand []int
	for i, id := range ps.Deck {
		for _, f := range op.Find {
			if matchFind(g.Card(id), f) {
				cand = append(cand, i)
				break
			}
		}
	}

	if max <= 0 || len(cand) == 0 {
		g.logf("jogador %d: busca no deck sem resultado", p+1)
		g.shuffle(ps.Deck)
		g.runOps(p, rest, nil)
		return false
	}

	g.Pending = &PendingChoice{
		Player: p, Dest: op.Dest, Max: max,
		Candidates: cand, Reveal: true, rest: rest,
	}
	return true
}

// ResolveChoice conclui a busca pendente: picks são posições na lista
// Candidates (0..len-1), até Max, podendo ser vazio (busca "falha").
func (g *Game) ResolveChoice(p int, picks []int) error {
	pc := g.Pending
	if pc == nil || pc.Player != p {
		return fmt.Errorf("jogador %d não tem escolha pendente", p+1)
	}
	if len(picks) > pc.Max {
		return fmt.Errorf("máximo de %d carta(s), %d escolhida(s)", pc.Max, len(picks))
	}
	seen := map[int]bool{}
	var deckIdxs []int
	for _, pk := range picks {
		if pk < 0 || pk >= len(pc.Candidates) || seen[pk] {
			return fmt.Errorf("escolha inválida/repetida: %d", pk)
		}
		seen[pk] = true
		deckIdxs = append(deckIdxs, pc.Candidates[pk])
	}

	ps := g.Players[p]
	// Remove do deck em ordem decrescente de índice (não desloca os demais).
	for i := 0; i < len(deckIdxs); i++ {
		for j := i + 1; j < len(deckIdxs); j++ {
			if deckIdxs[j] > deckIdxs[i] {
				deckIdxs[i], deckIdxs[j] = deckIdxs[j], deckIdxs[i]
			}
		}
	}
	var names []string
	for _, di := range deckIdxs {
		id := ps.Deck[di]
		ps.Deck = append(ps.Deck[:di], ps.Deck[di+1:]...)
		names = append(names, g.Card(id).Name.EN)
		if pc.Dest == "bench" {
			ps.Bench = append(ps.Bench, &PokemonInPlay{Stack: []string{id}, EnteredTurn: g.TurnNumber})
		} else {
			ps.Hand = append(ps.Hand, id)
		}
	}

	if len(names) == 0 {
		g.logf("jogador %d: busca no deck sem pegar nada", p+1)
	} else if pc.Reveal {
		g.logf("jogador %d: busca revela %s → %s", p+1, strings.Join(names, ", "), destPT(pc.Dest))
	} else {
		g.logf("jogador %d: busca pega %d carta(s) → %s", p+1, len(names), destPT(pc.Dest))
	}
	g.shuffle(ps.Deck)

	rest := pc.rest
	g.Pending = nil
	g.runOps(p, rest, nil)
	return nil
}

func destPT(dest string) string {
	if dest == "bench" {
		return "Banco"
	}
	return "mão"
}
