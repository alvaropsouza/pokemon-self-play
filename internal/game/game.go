// Package game implementa o motor de regras do Pokémon TCG (formato Standard).
//
// O motor é o árbitro da partida: mantém o estado completo (incluindo
// informação oculta como deck e prêmios) e valida/aplica ações. Camadas acima
// (CLI, bot, visão computacional) consomem o estado e nunca alteram zonas
// diretamente. Determinístico: mesma seed + mesmas ações → mesma partida.
//
// Efeitos de texto de cartas (ataques com efeito, Treinadores, Habilidades)
// não são interpretados automaticamente nesta versão: o motor aplica limites e
// mecânica base (dano, custos, condições) e expõe helpers de arbitragem
// (ApplyDamage, Heal, MoveToBench...) para o efeito ser resolvido manualmente.
package game

import (
	"fmt"
	"math/rand"

	"github.com/alvaropsouza/pokemon-self-play/internal/cards"
)

// Phase é a fase macro da partida.
type Phase string

const (
	PhaseSetup    Phase = "setup"    // colocação de Ativo/Banco antes do turno 1
	PhaseTurn     Phase = "turn"     // partida em andamento
	PhaseFinished Phase = "finished" // partida encerrada
)

// Condition é uma Condição Especial aplicável ao Pokémon Ativo.
type Condition string

const (
	CondNone      Condition = ""
	CondAsleep    Condition = "asleep"
	CondConfused  Condition = "confused"
	CondParalyzed Condition = "paralyzed"
)

// condPT nomeia em PT as condições que bloqueiam atacar/recuar (mensagens de erro).
var condPT = map[Condition]string{CondAsleep: "Adormecido", CondParalyzed: "Paralisado"}

// ActiveSlot referencia o Pokémon Ativo em parâmetros de alvo; índices ≥ 0
// referenciam posições do Banco.
const ActiveSlot = -1

// PokemonInPlay é um Pokémon em jogo com tudo que está fisicamente ligado a ele.
type PokemonInPlay struct {
	// Stack são os IDs das cartas da pilha, topo primeiro (carta atual;
	// evoluções anteriores por baixo).
	Stack []string
	// Energies são os IDs das cartas de Energia ligadas.
	Energies []string
	// Tool é o ID da Ferramenta ligada ("" = nenhuma).
	Tool string
	// Damage é o dano acumulado (múltiplos de 10).
	Damage int

	// Rot é a condição rotacional (Adormecido/Confuso/Paralisado) — mutuamente
	// exclusivas, a mais nova substitui. Poisoned/Burned coexistem.
	Rot      Condition
	Poisoned bool
	Burned   bool

	// EnteredTurn é o turno global em que entrou em jogo (0 = setup).
	EnteredTurn int
	// EvolvedTurn é o turno global da última evolução (0 = nunca).
	EvolvedTurn int
}

// TopID devolve o ID da carta do topo da pilha (a carta "atual" do Pokémon).
func (p *PokemonInPlay) TopID() string { return p.Stack[0] }

// clearConditions remove todas as Condições Especiais (recuo, evolução, ir ao Banco).
func (p *PokemonInPlay) clearConditions() {
	p.Rot = CondNone
	p.Poisoned = false
	p.Burned = false
}

// allCardIDs devolve todas as cartas físicas da pilha (stack + energias + ferramenta).
func (p *PokemonInPlay) allCardIDs() []string {
	ids := append([]string{}, p.Stack...)
	ids = append(ids, p.Energies...)
	if p.Tool != "" {
		ids = append(ids, p.Tool)
	}
	return ids
}

// PlayerState é o estado de um lado da mesa.
type PlayerState struct {
	Deck     []string // topo = índice 0
	Hand     []string
	Discard  []string
	Prizes   []string
	LostZone []string
	Active   *PokemonInPlay
	Bench    []*PokemonInPlay // máx. 5

	PrizesTaken int
	TurnsTaken  int
	Mulligans   int

	// Limites do turno corrente (zerados a cada início de turno).
	EnergyAttached  bool
	SupporterPlayed bool
	StadiumPlayed   bool
	Retreated       bool

	// setupReady indica que o jogador concluiu a colocação inicial.
	setupReady bool
}

// Game é a partida completa. Zonas e contadores são exportados para leitura;
// mutações passam pelos métodos de ação.
type Game struct {
	store *cards.Store
	rng   *rand.Rand

	Players [2]*PlayerState
	// Current é o índice do jogador da vez.
	Current int
	// TurnNumber é o turno global, começando em 1 no primeiro turno.
	TurnNumber int
	Phase      Phase
	// Winner é o índice do vencedor; -1 enquanto em andamento, -2 = Sudden Death
	// (condições simultâneas equivalentes).
	Winner int

	// Stadium em jogo ("" = nenhum) e quem o jogou.
	Stadium      string
	StadiumOwner int

	// NeedPromote[p] indica que o jogador p precisa promover um Pokémon do
	// Banco antes de qualquer outra ação.
	NeedPromote [2]bool

	// Pending é uma escolha de busca aguardando o jogador (nil = nenhuma).
	// Bloqueia as demais ações até ResolveChoice.
	Pending *PendingChoice

	// Log é o histórico legível de eventos da partida.
	Log []string
}

// New cria a partida: embaralha os decks, compra as mãos iniciais resolvendo
// mulligans (com compra extra automática por mulligan do oponente) e deixa a
// partida em PhaseSetup aguardando a colocação de Ativo/Banco.
// firstPlayer escolhe quem começa; passe -1 para sortear pela seed.
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

	// Mãos iniciais com mulligan: recompra até ter Pokémon Básico.
	for p := 0; p < 2; p++ {
		ps := g.Players[p]
		for {
			// Cópia obrigatória: fatiar o deck faria a mão compartilhar o array
			// e appends futuros na mão (prêmio, busca) corromperiam o deck.
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
	// Compra extra por mulligan do adversário (aplicada automaticamente).
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

// Card devolve a carta canônica de um ID (nil se desconhecido).
func (g *Game) Card(id string) *cards.Card { return g.store.Cards[id] }

// CloneWithSeed cria uma cópia profunda do estado de jogo com um novo RNG.
// O store de cartas é compartilhado (imutável). Log é descartado (irrelevante
// para simulações). Usado por MCTS para explorar ramos sem afetar o estado real.
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
		a := ps.Active.clonePkm()
		c.Active = a
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

func (g *Game) logf(format string, args ...any) {
	g.Log = append(g.Log, fmt.Sprintf(format, args...))
}

func (g *Game) shuffle(pile []string) {
	g.rng.Shuffle(len(pile), func(i, j int) { pile[i], pile[j] = pile[j], pile[i] })
}

// flip joga uma moeda: true = cara.
func (g *Game) flip() bool { return g.rng.Intn(2) == 0 }

func (g *Game) handHasBasic(hand []string) bool {
	for _, id := range hand {
		if c := g.store.Cards[id]; c != nil && c.IsBasicPokemon() {
			return true
		}
	}
	return false
}

// drawCard move 1 carta do topo do deck para a mão. Devolve false se o deck
// estava vazio (não decide derrota — quem chama trata deck-out).
func (g *Game) drawCard(p int) bool {
	ps := g.Players[p]
	if len(ps.Deck) == 0 {
		return false
	}
	ps.Hand = append(ps.Hand, ps.Deck[0])
	ps.Deck = ps.Deck[1:]
	return true
}

// ---- Setup ----

// PlaceActive coloca um Pokémon Básico da mão como Ativo (fase de setup).
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

// PlaceBench coloca um Pokémon Básico da mão no Banco (setup ou turno).
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
	return nil
}

// FinishSetup marca o jogador como pronto. Quando ambos terminam: separa os 6
// Prêmios, inicia o turno 1 e faz a compra obrigatória do primeiro jogador.
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
		// Cópia obrigatória (mesmo motivo da mão inicial: sem aliasing com o deck).
		s.Prizes = append([]string(nil), s.Deck[:6]...)
		s.Deck = s.Deck[6:]
	}
	g.Phase = PhaseTurn
	g.TurnNumber = 1
	g.logf("turno 1: jogador %d", g.Current+1)
	g.mandatoryDraw(g.Current)
	return nil
}

// ---- helpers de mão/validação ----

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

// requireTurn valida que é o turno do jogador p, a partida está em andamento e
// não há promoção pendente.
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

// target resolve uma referência de alvo (ActiveSlot ou índice de Banco) do jogador p.
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
