# Efeitos sem ação de UI definida

Referência para implementação futura. Cada `OpKind` de `internal/game/effects.go` que o motor já executa mas o frontend ainda não representa visualmente.

> **Status:** implementado. Estratégia final:
> - **Diff de estado no componente** (`Card.tsx`): dano sobe → flash vermelho + tremor; dano desce → flash verde. Cobre ataque, banco, recoil, checkup e arbitragem sem evento do servidor.
> - **Remontagem por key + CSS**: badge de condição entra com pop (`cpop`); topo do descarte assenta com `discpop` — cobre descarte de mão, energia e nocaute.
> - **Eventos estruturados** (`GameState.events`, drenados por resposta): só para o que o diff não vê — `shuffle_deck` e `shuffle_hand` → tremor no baralho (`effectsfx.ts`). Emitidos em `effects_run.go`, `search.go`, `arbiter.go`.
> - Compras múltiplas (`draw_until`, `draw_both`, etc.) já eram cobertas pela detecção de crescimento da mão em `HandTray`.
> - `scale_per_energy_*`: sem UI própria (dano final já anima), conforme sugerido abaixo.

## Cobertura atual do frontend

| Coberto | Como |
|---|---|
| `OpDraw` | Detecção de encolhimento do deck → voo `flyFromDeck` |
| `OpSearch` | `PendingChoice` overlay (kind `search`) |
| `OpSwitchSelf` | `PendingChoice` overlay (kind `switch_self`) |
| `OpSwitchOpp` | `PendingChoice` overlay (kind `switch_opp`) |
| `OpDiscardHand` / `OpDiscardFromHand` | `PendingChoice` overlay (kind `discard_hand`) |
| `OpStatus` (badge) | Badge estático na carta (componente `Card.tsx`) |

---

## Efeitos sem UI

### `draw_until`
**O quê:** compra cartas até ter N na mão.  
**UI sugerida:** mesma animação de compra do `OpDraw`, mas disparada múltiplas vezes em sequência até o contador de mão atingir N. Toast de log já informa o texto.

### `draw_or_more`
**O quê:** compra N cartas; se o jogador tiver exatamente M prêmios, compra Alt cartas em vez de N.  
**UI sugerida:** igual a `OpDraw`, mas o toast deve deixar claro qual variante foi usada ("compra N" vs "compra Alt").

### `shuffle_hand_both` / `shuffle_hand_self`
**O quê:** um ou ambos os jogadores embaralham a mão de volta no deck.  
**UI sugerida:** animação de voo das cartas da mão de volta para o deck (inverso de `flyFromDeck`). Badge ou toast "embaralhando mão…".

### `draw_both`
**O quê:** ambos os jogadores compram N cartas.  
**UI sugerida:** `flyFromDeck` disparado para os dois lados em sequência ou simultaneamente.

### `draw_per_prize_both`
**O quê:** ambos compram 1 carta por prêmio restante.  
**UI sugerida:** mesmo que `draw_both`, com N calculado dinamicamente a partir de `state.players[i].prizeCount`.

### `damage_opp_bench` / `damage_self_bench`
**O quê:** coloca N de dano em cada Pokémon do banco do oponente (ou do próprio).  
**UI sugerida:** flash vermelho rápido em cada slot de banco afetado (variante da animação de dano no Ativo), sem shake (banco não ataca). Atualização dos HP bars de cada card no banco.

### `heal_self`
**O quê:** remove N de dano do Pokémon Ativo do atacante.  
**UI sugerida:** flash verde + counter de HP subindo no card Ativo. Pode reusar a estrutura de animação de dano com cor/direção invertida.

### `discard_self_energy` / `discard_opp_energy`
**O quê:** descarta 1–N energias ligadas ao Ativo (próprio ou adversário). N = -1 = todas.  
**UI sugerida:** chips de energia sumindo com fade-out no card afetado. Toast já aparece via log.

### `scale_per_energy_self` / `scale_per_energy_opp`
**O quê:** o dano do ataque escala +N por energia ligada. Resolvido pelo motor como dano extra; o frontend não vê o multiplicador.  
**UI sugerida:** nenhuma obrigatória (o dano final já aparece no flash de dano). Opcional: tooltip ou badge "×energy" no card de ataque para indicar o escalonamento antes de confirmar.

### `shuffle_deck`
**O quê:** embaralha o deck do jogador ativo.  
**UI sugerida:** animação curta de "riffle" no sprite do deck (shake horizontal leve). Toast "deck embaralhado" já existe via log.

### `damage_self`
**O quê:** coloca N de dano no próprio Pokémon Ativo (recuo por confusão, dano de recoil de ataque).  
**UI sugerida:** flash vermelho + shake no card Ativo do atacante (mesmo efeito do dano recebido, só que no próprio lado). Já existe parcialmente no commit de flash; só falta apontar para o card do jogador ativo em vez do adversário.

### `OpStatus` — animação de aplicação
**O quê:** badge de condição especial já é exibido estaticamente, mas não há animação quando a condição é *aplicada*.  
**UI sugerida:** pop-in do badge com escala (scale 0 → 1.2 → 1) + som/vibração curto ao ser adicionado. Distinto do badge idle que já existe.

---

## Como conectar ao motor

O frontend hoje detecta mudanças de estado por polling/diff após cada ação. Para animar efeitos ordenados (ex.: `shuffle_hand_both` seguido de `draw_both`), o servidor precisará expor um `lastEvent` estruturado junto com o `GameState`, ou um array `events []Event` no response de cada ação.

Estrutura sugerida (Go):
```go
type Event struct {
    Kind    string `json:"kind"`    // OpKind string
    Player  int    `json:"player"`  // 0 | 1
    N       int    `json:"n,omitempty"`
    Targets []int  `json:"targets,omitempty"` // índices de bench afetados
}
```

Sem isso, o frontend só pode inferir o que mudou comparando estado anterior/novo — suficiente para atualizar valores mas não para disparar animações com semântica correta.
