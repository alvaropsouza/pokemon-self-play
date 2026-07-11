// Seleção corrente na UI: carta da mão ou Pokémon em jogo.
export type Sel = { kind: 'hand' | 'active' | 'bench'; idx: number } | null
