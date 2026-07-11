import type { CardView, PokemonView } from '../api'

// Carta genérica: imagem quando existe, fallback textual. Overlays de dano e
// condições quando o view é um Pokémon em jogo.
export function Card({ view, selected, onClick }: {
  view: CardView | PokemonView
  selected?: boolean
  onClick?: () => void
}) {
  const c = 'card' in view ? view.card : view
  const pk = 'card' in view ? view : null
  const cls = 'cardbox' + (onClick ? ' click' : '') + (selected ? ' sel' : '')
  return (
    <div className={cls} onClick={onClick}>
      {c.image
        ? <img className="card" src={c.image} title={c.name} alt={c.name} />
        : <div className="card txt">{c.name}</div>}
      {pk && pk.damage > 0 && <span className="dmg">{pk.damage}</span>}
      {pk && pk.conditions.length > 0 && <span className="cond">{pk.conditions.join(',')}</span>}
    </div>
  )
}

export function EmptySlot() {
  return <div className="slot" />
}

// Slot de Pokémon em jogo (ativo/banco); vazio vira slot tracejado.
export function PokemonSlot({ view, selected, onClick }: {
  view: PokemonView | null | undefined
  selected?: boolean
  onClick?: () => void
}) {
  if (!view) return <div><EmptySlot /></div>
  const n = view.energies?.length ?? 0
  return (
    <div>
      <Card view={view} selected={selected} onClick={onClick} />
      {(n > 0 || view.tool) && (
        <div className="sub">{n > 0 && `${n}⚡`}{view.tool && ' 🔧'}</div>
      )}
    </div>
  )
}

export function DeckPile({ count }: { count: number }) {
  if (count <= 0) return <div className="pile empty" />
  return (
    <div className="pile">
      <div className="ball" />
      <span className="cnt">{count}</span>
    </div>
  )
}

export function DiscardPile({ discard }: { discard: CardView[] | null }) {
  const n = discard?.length ?? 0
  if (!n) return <div className="discard-top"><div className="pile empty" /></div>
  return (
    <div className="discard-top">
      <div className="cardbox">
        <img className="card" src={discard![n - 1].image} title={discard![n - 1].name} alt={discard![n - 1].name} />
        <span className="cnt">{n}</span>
      </div>
    </div>
  )
}
