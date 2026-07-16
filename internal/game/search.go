package game

import (
	"fmt"
	"sort"
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
	MaxHP    int    `json:"maxHp,omitempty"` // Pokémon com "N HP or less" (0 = sem filtro)
}

// ChoiceKind diferencia o tipo de escolha pendente.
type ChoiceKind string

const (
	ChoiceSearch      ChoiceKind = "search"       // busca no deck
	ChoiceSwitchSelf  ChoiceKind = "switch_self"  // troca Ativo↔Banco próprio
	ChoiceSwitchOpp   ChoiceKind = "switch_opp"   // atacante escolhe Banco do oponente para virar Ativo
	ChoiceDiscardHand ChoiceKind = "discard_hand" // descarta N cartas da mão (custo/efeito)
)

// PendingChoice é uma escolha aguardando o jogador.
type PendingChoice struct {
	Kind       ChoiceKind
	Player     int
	Dest       string // "hand" | "bench" (busca no deck)
	Max        int    // máximo de itens a escolher
	Min        int    // mínimo obrigatório (0 = escolha opcional; >0 em descarte de custo)
	Candidates []int  // índices: deck (search), Bench (switch) ou mão (discard_hand)
	Reveal     bool   // itens escolhidos são revelados no log (busca)
	rest       []Op   // ops restantes executadas após a resolução
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
		if f.MaxHP > 0 && c.HP > f.MaxHP {
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
		if c.EnergyType != "Normal" && c.EnergyType != "Basic" {
			return false
		}
		return f.Type == "" || strings.HasPrefix(c.Name.EN, f.Type+" ")
	default:
		return false
	}
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
		g.event("shuffle_deck", p)
		g.runOps(p, rest, nil)
		return false
	}

	g.Pending = &PendingChoice{
		Kind: ChoiceSearch, Player: p, Dest: op.Dest, Max: max,
		Candidates: cand, Reveal: true, rest: rest,
	}
	return true
}

// startDiscardHand cria escolha pendente de descarte de N cartas da mão.
// Retorna true se ficou pendente; false se a mão está vazia (nada a descartar).
func (g *Game) startDiscardHand(p int, op Op, rest []Op) bool {
	ps := g.Players[p]
	n := op.N
	if n > len(ps.Hand) {
		n = len(ps.Hand)
	}
	if n <= 0 {
		g.runOps(p, rest, nil)
		return false
	}
	cand := make([]int, len(ps.Hand))
	for i := range cand {
		cand[i] = i
	}
	g.Pending = &PendingChoice{
		Kind: ChoiceDiscardHand, Player: p, Max: n, Min: n, Candidates: cand, rest: rest,
	}
	return true
}

// startSwitchPending cria escolha pendente de troca de Ativo↔Banco.
// forOpp=true: atacante (p) escolhe qual Pokémon do Banco do oponente vira Ativo.
// Retorna true se ficou pendente; false se não há Banco (nada a fazer).
func (g *Game) startSwitchPending(p int, forOpp bool, rest []Op) bool {
	target := p
	kind := ChoiceSwitchSelf
	if forOpp {
		target = 1 - p
		kind = ChoiceSwitchOpp
	}
	bench := g.Players[target].Bench
	if len(bench) == 0 {
		g.runOps(p, rest, nil)
		return false
	}
	cand := make([]int, len(bench))
	for i := range cand {
		cand[i] = i
	}
	g.Pending = &PendingChoice{
		Kind: kind, Player: p, Max: 1, Candidates: cand, rest: rest,
	}
	return true
}

// ResolveChoice conclui uma escolha pendente (busca no deck ou troca de Ativo).
// picks são posições na lista Candidates (0..len-1), até Max, podendo ser vazio.
func (g *Game) ResolveChoice(p int, picks []int) error {
	pc := g.Pending
	if pc == nil || pc.Player != p {
		return fmt.Errorf("jogador %d não tem escolha pendente", p+1)
	}
	if len(picks) > pc.Max {
		return fmt.Errorf("máximo de %d escolha(s), %d fornecida(s)", pc.Max, len(picks))
	}
	if len(picks) < pc.Min {
		return fmt.Errorf("mínimo de %d escolha(s), %d fornecida(s)", pc.Min, len(picks))
	}
	seen := map[int]bool{}
	for _, pk := range picks {
		if pk < 0 || pk >= len(pc.Candidates) || seen[pk] {
			return fmt.Errorf("escolha inválida/repetida: %d", pk)
		}
		seen[pk] = true
	}

	switch pc.Kind {
	case ChoiceDiscardHand:
		ps := g.Players[p]
		var handIdxs []int
		for _, pk := range picks {
			handIdxs = append(handIdxs, pc.Candidates[pk])
		}
		// Remove da mão em ordem decrescente de índice.
		sortDesc(handIdxs)
		var names []string
		for _, hi := range handIdxs {
			id := ps.Hand[hi]
			ps.Hand = append(ps.Hand[:hi], ps.Hand[hi+1:]...)
			ps.Discard = append(ps.Discard, id)
			names = append(names, g.Card(id).Name.EN)
		}
		g.logf("jogador %d: descarta %s da mão", p+1, strings.Join(names, ", "))
	case ChoiceSwitchSelf, ChoiceSwitchOpp:
		if len(picks) > 0 {
			benchIdx := pc.Candidates[picks[0]]
			owner := p
			if pc.Kind == ChoiceSwitchOpp {
				owner = 1 - p
			}
			g.performSwitch(owner, benchIdx)
		}
	default: // ChoiceSearch
		ps := g.Players[p]
		var deckIdxs []int
		for _, pk := range picks {
			deckIdxs = append(deckIdxs, pc.Candidates[pk])
		}
		// Remove do deck em ordem decrescente de índice.
		sortDesc(deckIdxs)
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
		switch {
		case len(names) == 0:
			g.logf("jogador %d: busca no deck sem pegar nada", p+1)
		case pc.Reveal:
			g.logf("jogador %d: busca revela %s → %s", p+1, strings.Join(names, ", "), destPT(pc.Dest))
		default:
			g.logf("jogador %d: busca pega %d carta(s) → %s", p+1, len(names), destPT(pc.Dest))
		}
		g.shuffle(ps.Deck)
		g.event("shuffle_deck", p)
	}

	rest := pc.rest
	g.Pending = nil
	g.runOps(p, rest, nil)
	return nil
}

// performSwitch troca o Ativo do jogador p com Bench[benchIdx], removendo condições do que saiu.
func (g *Game) performSwitch(p, benchIdx int) {
	ps := g.Players[p]
	if ps.Active == nil || benchIdx < 0 || benchIdx >= len(ps.Bench) {
		return
	}
	old := ps.Active
	old.clearConditions()
	ps.Active = ps.Bench[benchIdx]
	ps.Bench[benchIdx] = old
	g.logf("jogador %d: %s → Banco, %s → Ativo", p+1,
		g.Card(old.TopID()).Name.EN, g.Card(ps.Active.TopID()).Name.EN)
}

func sortDesc(idxs []int) {
	sort.Sort(sort.Reverse(sort.IntSlice(idxs)))
}

func destPT(dest string) string {
	if dest == "bench" {
		return "Banco"
	}
	return "mão"
}
