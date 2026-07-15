package game

// TriggerKind identifica um momento do jogo que habilidades reativas observam.
type TriggerKind string

const (
	TrigEnergyAttached TriggerKind = "energy-attached"
	TrigEvolved        TriggerKind = "evolved"
	TrigKnockOut       TriggerKind = "knockout"
	TrigTurnEnded      TriggerKind = "turn-ended"
)

// Trigger descreve o momento ocorrido e o Pokémon envolvido (Player + Slot,
// com ActiveSlot para o Ativo).
type Trigger struct {
	Kind   TriggerKind
	Player int
	Slot   int
}

type triggerHandler func(g *Game, owner, slot int, t Trigger)

// registro em nível de pacote (não por Game) para que CloneWithSeed continue
// trivial: nenhum estado de listener vive dentro do Game
var triggerDB = map[string]triggerHandler{}

func (g *Game) emit(t Trigger) {
	for p := 0; p < 2; p++ {
		ps := g.Players[p]
		if ps.Active != nil {
			g.fireTrigger(p, ActiveSlot, ps.Active, t)
		}
		for i, pk := range ps.Bench {
			g.fireTrigger(p, i, pk, t)
		}
	}
}

func (g *Game) fireTrigger(owner, slot int, pk *PokemonInPlay, t Trigger) {
	if h := triggerDB[pk.TopID()]; h != nil {
		h(g, owner, slot, t)
	}
}
