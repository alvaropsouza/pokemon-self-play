// Tipos do estado servido por cmd/play (/api/state, /api/action) e helpers de fetch.

export interface Attack {
  name: string
  cost: string[] | null
  damage: string
}

export interface Ability {
  name: string
  effect: string
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
  ability?: Ability
}

export interface PokemonView {
  card: CardView
  damage: number
  energies: CardView[] | null
  conditions: string[]
  tool?: CardView
  abilityUsed?: boolean
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

export interface PendingChoice {
  kind: 'search' | 'switch_self' | 'switch_opp' | 'discard_hand'
  max: number
  min: number
  dest: string
  candidates: (CardView | PokemonView)[]
}

export interface GameEvent {
  kind: 'shuffle_deck' | 'shuffle_hand'
  player: number
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
  pendingChoice?: PendingChoice
  events?: GameEvent[]
  error?: string
}

export interface GameConfig {
  mytype: string
  bottype: string
}

// Battle Deck fixo servido por /api/decks para a tela de seleção.
export interface DeckEntry {
  card: CardView
  count: number
}

export interface DeckInfo {
  id: string
  type: string
  name: string
  star: CardView
  counts: Record<string, number>
  cards: DeckEntry[]
}

export type Sel =
  | { kind: 'hand' | 'active' | 'bench'; idx: number }
  | { kind: 'pending'; action: string; handIdx: number }
  | { kind: 'retreating'; benchIdx: number | null; energyIdxs: number[] }
  | { kind: 'ability'; slot: number }
  | null

async function readJSON<T>(r: Response): Promise<T> {
  const text = await r.text()
  if (!text) throw new Error(`HTTP ${r.status}: resposta vazia`)
  try {
    return JSON.parse(text) as T
  } catch {
    throw new Error(`HTTP ${r.status}: resposta não-JSON — ${text.slice(0, 120)}`)
  }
}

export async function fetchState(): Promise<GameState> {
  const r = await fetch('/api/state')
  return readJSON<GameState>(r)
}

export async function postAction(body: Record<string, unknown>): Promise<GameState> {
  const r = await fetch('/api/action', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  return readJSON<GameState>(r)
}

export async function fetchDecks(): Promise<DeckInfo[]> {
  const r = await fetch('/api/decks')
  return readJSON<DeckInfo[]>(r)
}

export async function postNew(config: GameConfig): Promise<GameState> {
  const r = await fetch('/api/new', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config),
  })
  return readJSON<GameState>(r)
}
