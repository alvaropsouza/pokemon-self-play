---
target: premios dos jogadores
total_score: 31
p0_count: 0
p1_count: 1
timestamp: 2026-07-16T01-43-19Z
slug: premios-dos-jogadores
---
## Design Health Score

| # | Heuristic | Score | Key Issue |
|---|-----------|-------|-----------|
| 1 | Visibility of System Status | 3 | No momentary feedback when prize is collected |
| 2 | Match System / Real World | 4 | Pokéball = exact physical object; "Prêmios" = correct TCG term |
| 3 | User Control and Freedom | 3 | Read-only widget — n/a |
| 4 | Consistency and Standards | 3 | prize-label 9px vs mlabel 11px — minor system drift |
| 5 | Error Prevention | 3 | Read-only; no motor errors possible |
| 6 | Recognition Rather Than Recall | 4 | Pokéball + label + count — nothing to memorize |
| 7 | Flexibility and Efficiency | 3 | Count scannable instantly |
| 8 | Aesthetic and Minimalist Design | 3 | 20px count louder than 13px player name — resolved by intent |
| 9 | Error Recovery | 3 | N/a |
| 10 | Help and Documentation | 2 | "4/6" direction ambiguous without domain knowledge |
| **Total** | | **31/40** | **Good** |

## Anti-Patterns Verdict

LLM: Clean. Pokéball gradient is thematic, not decorative. detect.mjs: 0 findings.

## Overall Impression

Structurally sound and thematically correct. Prize collection — the most significant game event — passes emotionally silent. Two clear improvement targets: animate the collection moment, and make the ARIA group label player-specific.

## What's Working

1. Color identity locked in — blue bot, gold you, matching .pp panel identity.
2. Triple-signal taken prizes: opacity + scale + grayscale. Works for color-blind users.
3. Count + dots redundancy is deliberate and correct for two scanning distances.

## Priority Issues

**[P1] Prize collection is emotionally flat** — No peak-moment feedback on KO/prize event. Fix: @keyframes prizeout pop on the leaving prize; key the count span to trigger a dmgpop-style number flash. Command: $impeccable animate

**[P2] Count direction ambiguous** — "4/6" could mean remaining OR collected. Fix: tooltip, or change prize-count color to var(--bad) when count ≤ 2. Command: $impeccable clarify

**[P2] Screen reader group label missing player name** — Two groups both announce "N prêmios restantes" with no player attribution. Fix: aria-label="Seus prêmios: N restantes" vs "Prêmios do bot: N restantes". Command: $impeccable harden

**[P3] Label system drift** — .prize-label 9px/0.8px vs .mlabel 11px/1px. Command: $impeccable polish

## Persona Red Flags

**Sam (Screen Reader):** Tabs into two groups with identical labels — cannot tell which is theirs without counting previous elements.

**Marco (Casual TCG Player):** Prize collected, nothing flickers on sidebar. Checks log to confirm KO was registered.

## Minor Observations

- border-top inside .pp makes prize track feel bolted on; margin-top alone may suffice.
- box-shadow 4px blur on .prize: within ≤8px rule. OK.
- text-transform:uppercase on "Prêmios" correctly renders "PRÊMIOS". OK.

## Questions to Consider

- When a prize is collected, what should the player feel? Relief or tension? That drives the animation choice.
- Should count turn red at 2/6 and 1/6 for endgame legibility?
- Is the prize track the right home, or would a single shared scoreboard panel tell the story better?
