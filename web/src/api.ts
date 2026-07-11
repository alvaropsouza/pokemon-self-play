// Tipos do estado servido por cmd/play (/api/state, /api/action) e helpers de fetch.

export interface Attack {
  name: string
  cost: string[] | null
  damage: string
}

export interface CardView {
  id: string
  name: string
  nameEN: string
  image: string
  category: string
  stage: string
  trainerType: string
  hp: number
  retreat: number
  attacks: Attack[] | null
}

export interface PokemonView {
  card: CardView
  damage: number
  energies: CardView[] | null
  conditions: string[]
  tool?: CardView
}

export interface SideView {
  deck: number
  prizes: number
  prizesTaken: number
  active: PokemonView | null
  bench: PokemonView[] | null
  discard: CardView[] | null
  handCount: number
  hand?: CardView[]
}

export interface GameState {
  phase: string
  turn: number
  current: number
  winner: number
  needPromote: boolean[]
  log: string[] | null
  you: SideView
  bot: SideView
  stadium?: CardView
  error?: string
}

export async function fetchState(): Promise<GameState> {
  const r = await fetch('/api/state')
  return r.json()
}

export async function postAction(body: Record<string, unknown>): Promise<GameState> {
  const r = await fetch('/api/action', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  return r.json()
}
