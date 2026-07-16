package game

// Command is a single game action that can be executed against a Game.
type Command interface {
	Execute(g *Game) error
}

type PlaceActiveCmd   struct{ Player, HandIdx int }
type PlaceBenchCmd    struct{ Player, HandIdx int }
type FinishSetupCmd   struct{ Player int }
type AttachEnergyCmd  struct{ Player, HandIdx, Slot int }
type EvolveCmd        struct{ Player, HandIdx, Slot int }
type AttachToolCmd    struct{ Player, HandIdx, Slot int }
type PlayItemCmd      struct{ Player, HandIdx int }
type PlaySupporterCmd struct{ Player, HandIdx int }
type PlayStadiumCmd   struct{ Player, HandIdx int }
type RetreatCmd       struct{ Player, BenchIdx int; Energies []int }
type AttackCmd        struct{ Player, AtkIdx int }
type PromoteCmd       struct{ Player, BenchIdx int }
type EndTurnCmd       struct{ Player int }
type ResolveChoiceCmd struct{ Player int; Picks []int }
type UseAbilityCmd    struct{ Player, AbilitySlot, Target int }
type ArbDamageCmd     struct{ Player, Slot, Amount int }
type ArbHealCmd       struct{ Player, Slot, Amount int }
type ArbConditionCmd  struct{ Player int; Condition string }
type ArbDrawCmd       struct{ Player, Amount int }
type ArbSwitchCmd     struct{ Player, BenchIdx int }
type ArbShuffleCmd    struct{ Player int }

func (c PlaceActiveCmd)   Execute(g *Game) error { return g.PlaceActive(c.Player, c.HandIdx) }
func (c PlaceBenchCmd)    Execute(g *Game) error { return g.PlaceBench(c.Player, c.HandIdx) }
func (c FinishSetupCmd)   Execute(g *Game) error { return g.FinishSetup(c.Player) }
func (c AttachEnergyCmd)  Execute(g *Game) error { return g.AttachEnergy(c.Player, c.HandIdx, c.Slot) }
func (c EvolveCmd)        Execute(g *Game) error { return g.Evolve(c.Player, c.HandIdx, c.Slot) }
func (c AttachToolCmd)    Execute(g *Game) error { return g.AttachTool(c.Player, c.HandIdx, c.Slot) }
func (c PlayItemCmd)      Execute(g *Game) error { return g.PlayItem(c.Player, c.HandIdx) }
func (c PlaySupporterCmd) Execute(g *Game) error { return g.PlaySupporter(c.Player, c.HandIdx) }
func (c PlayStadiumCmd)   Execute(g *Game) error { return g.PlayStadium(c.Player, c.HandIdx) }
func (c RetreatCmd)       Execute(g *Game) error { return g.Retreat(c.Player, c.BenchIdx, c.Energies) }
func (c AttackCmd)        Execute(g *Game) error { return g.Attack(c.Player, c.AtkIdx) }
func (c PromoteCmd)       Execute(g *Game) error { return g.Promote(c.Player, c.BenchIdx) }
func (c EndTurnCmd)       Execute(g *Game) error { return g.EndTurn(c.Player) }
func (c ResolveChoiceCmd) Execute(g *Game) error { return g.ResolveChoice(c.Player, c.Picks) }
func (c UseAbilityCmd)   Execute(g *Game) error { return g.UseAbility(c.Player, c.AbilitySlot, c.Target) }
func (c ArbDamageCmd)    Execute(g *Game) error { return g.ApplyDamage(c.Player, c.Slot, c.Amount) }
func (c ArbHealCmd)      Execute(g *Game) error { return g.Heal(c.Player, c.Slot, c.Amount) }
func (c ArbConditionCmd) Execute(g *Game) error { return g.SetCondition(c.Player, c.Condition) }
func (c ArbDrawCmd)      Execute(g *Game) error { g.DrawCards(c.Player, c.Amount); return nil }
func (c ArbSwitchCmd)    Execute(g *Game) error { return g.SwitchActive(c.Player, c.BenchIdx) }
func (c ArbShuffleCmd)   Execute(g *Game) error { g.ShuffleDeck(c.Player); return nil }
