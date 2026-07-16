package game

import (
	"math/rand"

	"github.com/alvaropsouza/pokemon-self-play/internal/cards"
)

type Phase string

const (
	PhaseSetup    Phase = "setup"
	PhaseTurn     Phase = "turn"
	PhaseFinished Phase = "finished"
)

type Condition string

const (
	CondNone      Condition = ""
	CondAsleep    Condition = "asleep"
	CondConfused  Condition = "confused"
	CondParalyzed Condition = "paralyzed"
)

var condPT = map[Condition]string{CondAsleep: "Adormecido", CondParalyzed: "Paralisado"}

const ActiveSlot = -1

// PokemonInPlay is a Pokémon on the field with all attached cards.
type PokemonInPlay struct {
	Stack    []string  // current card first; previous evolutions beneath
	Energies []string
	Tool     string
	Damage   int

	Rot      Condition // mutually exclusive rotational conditions; newer replaces older
	Poisoned bool
	Burned   bool

	EnteredTurn int
	EvolvedTurn int
}

func (p *PokemonInPlay) TopID() string { return p.Stack[0] }

func (p *PokemonInPlay) clearConditions() {
	p.Rot = CondNone
	p.Poisoned = false
	p.Burned = false
}

func (p *PokemonInPlay) allCardIDs() []string {
	ids := append([]string{}, p.Stack...)
	ids = append(ids, p.Energies...)
	if p.Tool != "" {
		ids = append(ids, p.Tool)
	}
	return ids
}

// PlayerState holds one side of the table.
type PlayerState struct {
	Deck     []string // draw from index 0
	Hand     []string
	Discard  []string
	Prizes   []string
	LostZone []string
	Active   *PokemonInPlay
	Bench    []*PokemonInPlay

	PrizesTaken int
	TurnsTaken  int
	Mulligans   int

	EnergyAttached  bool
	SupporterPlayed bool
	StadiumPlayed   bool
	Retreated       bool
	AbilitiesUsed   map[int]bool

	setupReady bool
}

// Game is the full match state. Zones and counters are exported for reading;
// mutations go through action methods.
type Game struct {
	store *cards.Store
	rng   *rand.Rand

	Players    [2]*PlayerState
	Current    int
	TurnNumber int
	Phase      Phase
	Winner     int // -1 = in progress, -2 = Sudden Death

	Stadium      string
	StadiumOwner int

	NeedPromote [2]bool
	Pending     *PendingChoice

	Log    []string
	Events []Event
}

// Event notifica a UI de um efeito com semântica que o diff de estado não
// captura (ex.: embaralhar não muda contagens). Drenado pelo servidor a cada resposta.
type Event struct {
	Kind   string `json:"kind"`
	Player int    `json:"player"`
}
