---
target: tabuleiro de jogo
total_score: 28
p0_count: 0
p1_count: 1
timestamp: 2026-07-14T02-52-44Z
slug: web-src-components-mat-tsx
---
# Critique: tabuleiro de jogo (web/src/components/Mat.tsx)

## Design Health Score

| # | Heuristic | Score | Key Issue |
|---|-----------|-------|-----------|
| 1 | Visibility of System Status | 3 | HP, contagens, condições e vez sempre visíveis; falta indicação de fase do turno no tabuleiro em si |
| 2 | Match System / Real World | 4 | Metáfora de playmat fiel: zonas nomeadas em PT, posições físicas, cartas do bot viradas |
| 3 | User Control and Freedom | 3 | Seleção por clique com feedback; sem "cancelar" visível durante pick inline |
| 4 | Consistency and Standards | 3 | Vocabulário coerente (dourado=você, azul=bot, verde=alvo); Estádio dentro do tapete do bot destoa (zona compartilhada) |
| 5 | Error Prevention | 3 | Só alvos válidos destacados (picking/over); drop aceita qualquer data sem validação visual prévia |
| 6 | Recognition Rather Than Recall | 3 | Zonas rotuladas; chip de Ferramenta "F" é críptico (só title no hover) |
| 7 | Flexibility and Efficiency | 2 | Drag E clique — bom; zero teclado (cartas são divs sem tabindex) |
| 8 | Aesthetic and Minimalist Design | 3 | Feltro escuro comprometido, motion só de estado; rótulos/placeholder 8–9px no limite |
| 9 | Error Recovery | 3 | err-banner persistente até próxima ação (fora do tapete, mas serve o tabuleiro) |
| 10 | Help and Documentation | 1 | Nenhuma ajuda contextual de regras; gaveta é log/arbitragem, não ajuda |
| **Total** | | **28/40** | **Good — base sólida, atacar pontos fracos** |

## Anti-Patterns Verdict

**LLM**: não parece gerado por IA. Playmat escuro com costura, Pokébola como motivo, identidade por lado via brilho de borda — decisões próprias do domínio, não gramática de IA. Sem eyebrows, sem gradient text, sem card grids.

**Deterministic scan**: `detect.mjs` em Mat.tsx, Card.tsx, App.tsx → **0 achados**.

**Visual overlays**: não executado — política do projeto proíbe validação via browser pelo agente; usuário valida em localhost:5173.

## Overall Impression

Tabuleiro lê como mesa de verdade: hierarquia espacial correta, estado legível, cor comprometida sem teatro. Maior oportunidade: robustez de layout (clipping da coluna de pilhas em telas altas) e acessibilidade mínima de teclado que o próprio PRODUCT.md pede.

## What's Working

1. **Metáfora física consistente** — zonas nomeadas com posição fixa, tapete do bot espelhado com cartas a 180°, pilhas com profundidade física (box-shadow escalonado). Princípio "a mesa é a interface" cumprido.
2. **Estado sem cliques** — HP bar com faixas de cor + texto, contadores nas pilhas, badges de condição com sigla, glow de vez. Princípio "estado sempre visível" cumprido.
3. **Feedback de ação contextual** — verde pulsante em alvos de pick, verde em drag-over, azul hover / dourado seleção. Vocabulário de cor estável.

## Priority Issues

- **[P1] Clipping da coluna de pilhas em telas altas**: `.mat.bot` / `.mat.you` fixam coluna de pilhas em 96px, mas `.pile` mede `calc(var(--cw) * .92)` — com `--cw` no teto (150px) a pilha tem 138px e é cortada por `overflow:hidden`. **Why**: em monitor grande (o caso de uso primário — monitor ao lado da mesa) contador e borda da pilha somem. **Fix**: coluna em `calc(var(--cw) * .92 + 8px)` ou `min-content`. **Command**: $impeccable adapt.
- **[P2] Tabuleiro 100% mouse**: cartas clicáveis são `div onClick` sem `tabindex`/`role="button"`/Enter. PRODUCT.md pede "foco visível em controles" — no tapete não há foco possível. **Why**: quebra o básico de a11y auto-declarado. **Fix**: `role="button" tabIndex={0}` + onKeyDown em `Card`/`PokemonSlot` clicáveis + `:focus-visible`. **Command**: $impeccable harden.
- **[P2] Estádio preso ao tapete do bot**: zona compartilhada renderizada dentro de `.mat.bot`, com brilho azul do bot em volta. **Why**: leitura errada de posse; no jogo físico fica no centro. **Fix**: faixa central própria entre os dois tapetes (ou neutralizar o tint na apoiocol). **Command**: $impeccable layout.
- **[P3] Microtipografia no limite**: placeholder do slot 9px, badges de condição 8px, hptxt 9px. Em `--cw` mínimo (64px, laptop) fica ilegível a 50cm. **Fix**: piso 10px, abreviar textos. **Command**: $impeccable typeset.
- **[P3] Chip de Ferramenta "F" críptico**: só hover revela nome. **Fix**: mini-ícone ou sigla da carta; manter title. **Command**: $impeccable clarify.

## Persona Red Flags

**Alex (power user)**: fluxo primário (ligar energia → atacar) exige mouse preciso em alvos pequenos no `--cw` mínimo; sem atalhos (ex.: Enter = ataque selecionado, E = terminar turno). Aceitável para uso pessoal, mas o dono do projeto É o power user.

**Sam (a11y)**: nenhuma carta focável; estados de HP têm texto além da cor (ok); badges de condição dependem de sigla 8px; drag-and-drop sem alternativa de teclado (clique existe — mitiga).

**Riley (stress)**: 5+ energias ligadas + Ferramenta no `.sub` de um slot de banco estreito — edots quebram linha e empurram o layout do slot; banco com 5 Pokémon em `--cw` máximo pode estourar `.benchrow` (gap 10px fixo). Descarte com carta sem imagem cai no fallback `.card.txt` dentro de `.pile`? Não — usa `Card` direto, ok.

## Minor Observations

- `.mat::after` (costura) sobrepõe cantos do conteúdo em tapetes muito cheios — inset 7px cruza a coluna de pilhas quando clipada (mesma causa do P1).
- Rotação 180° das cartas do bot: badges de dano/condição ficam de cabeça para baixo junto — dano do bot lê invertido; overlay podia contra-rotacionar.
- `justify-content:safe center` no handtray: bom detalhe, raro ver certo.

## Questions to Consider

- E se a faixa do Estádio fosse o divisor visual entre os dois tapetes — resolvendo posse e dando respiro ao centro?
- O tabuleiro precisa dos rótulos de zona depois da 3ª partida, ou um modo "limpo" (rótulos só em hover) deixaria o feltro respirar?
- Dano/condição do bot mereciam leitura na orientação do jogador?
