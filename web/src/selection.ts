// Seleção corrente na UI: carta da mão, Pokémon em jogo, ou modo de pick inline.
export type Sel =
  | { kind: 'hand' | 'active' | 'bench'; idx: number }
  // Aguardando clique num slot para completar ação (substitui window.prompt).
  | { kind: 'pending'; action: string; handIdx: number }
  // Recuo em dois passos: 1) escolher slot do banco, 2) selecionar energias.
  | { kind: 'retreating'; benchIdx: number | null; energyIdxs: number[] }
  | null
