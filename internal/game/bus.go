package game

type TriggerKind string

const (
	TrigEnergyAttached TriggerKind = "energy-attached"
	TrigEvolved        TriggerKind = "evolved"
	TrigKnockOut       TriggerKind = "knockout"
	TrigTurnEnded      TriggerKind = "turn-ended"
	TrigBenchPlaced    TriggerKind = "bench-placed"
)

type Trigger struct {
	Kind   TriggerKind
	Player int
	Slot   int
}

type triggerHandler func(g *Game, owner, slot int, t Trigger)

var triggerDB = map[string]triggerHandler{
	"me01-127": riskyRuins,
}

func riskyRuins(g *Game, _, _ int, t Trigger) {
	if t.Kind != TrigBenchPlaced {
		return
	}
	pk, err := g.target(t.Player, t.Slot)
	if err != nil {
		return
	}
	c := g.Card(pk.TopID())
	for _, ty := range c.Types {
		if ty == "Darkness" {
			return
		}
	}
	pk.Damage += 20
	g.logf("Risky Ruins: 2 contadores de dano em %s", c.Name.EN)
}

func HasTrigger(cardID string) bool {
	_, ok := triggerDB[cardID]
	return ok
}

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
	if g.Stadium != "" {
		if h := triggerDB[g.Stadium]; h != nil {
			h(g, g.StadiumOwner, ActiveSlot, t)
		}
	}
}

func (g *Game) fireTrigger(owner, slot int, pk *PokemonInPlay, t Trigger) {
	if h := triggerDB[pk.TopID()]; h != nil {
		h(g, owner, slot, t)
	}
}
