package game

// checkup é o Pokémon Checkup entre turnos (seção 8 do CLAUDE.md).
// Ordem: Envenenado (1 contador) → Queimado (2 contadores + moeda) →
// Adormecido (moeda) → Paralisado (remove no fim do turno do dono).
// Processa os Ativos dos dois jogadores; nocautes por veneno/queimadura são
// resolvidos ao final.
func (g *Game) checkup() {
	for p := 0; p < 2; p++ {
		a := g.Players[p].Active
		if a == nil {
			continue
		}
		name := g.Card(a.TopID()).Name.EN
		if a.Poisoned {
			a.Damage += 10
			g.logf("checkup: %s Envenenado, +10", name)
		}
		if a.Burned {
			a.Damage += 20
			if g.flip() {
				a.Burned = false
				g.logf("checkup: %s Queimado, +20 — cara, queimadura removida", name)
			} else {
				g.logf("checkup: %s Queimado, +20 — coroa, continua", name)
			}
		}
		if a.Rot == CondAsleep {
			if g.flip() {
				a.Rot = CondNone
				g.logf("checkup: %s acorda (cara)", name)
			} else {
				g.logf("checkup: %s continua Adormecido (coroa)", name)
			}
		}
		// Paralisia sai no checkup ao final do turno do próprio dono.
		if a.Rot == CondParalyzed && p == g.Current {
			a.Rot = CondNone
			g.logf("checkup: %s deixa de estar Paralisado", name)
		}
	}
	g.resolveKnockouts()
}
