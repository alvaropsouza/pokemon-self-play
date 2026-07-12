import { useContext } from 'react'
import type { CardView, PokemonView } from '../api'
import { energyColor, energyImage } from '../energy'
import { PreviewCtx } from '../preview'

// Carta genérica: imagem quando existe; energia básica sem imagem usa o scan
// do pokemontcg.io; demais sem imagem, fallback textual. Overlays de dano e
// condições quando o view é um Pokémon em jogo. Hover publica no preview.
export function Card({ view, selected, onClick, dragData }: {
  view: CardView | PokemonView
  selected?: boolean
  onClick?: () => void
  // Valor posto no dataTransfer ao arrastar (carta da mão arrastável).
  dragData?: string
}) {
  const c = 'card' in view ? view.card : view
  const pk = 'card' in view ? view : null
  const setPreview = useContext(PreviewCtx)
  const img = c.image || (c.category === 'Energy' ? energyImage(c.nameEN) : '')
  const cls = 'cardbox' + (onClick ? ' click' : '') + (selected ? ' sel' : '')
  return (
    <div className={cls} onClick={onClick}
      draggable={dragData !== undefined}
      onDragStart={dragData !== undefined
        ? e => e.dataTransfer.setData('text/plain', dragData)
        : undefined}
      onMouseEnter={() => setPreview(c)} onMouseLeave={() => setPreview(null)}>
      {img
        ? <img className="card" src={img} title={c.name} alt={c.name} />
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
// Energias ligadas aparecem como bolinhas coloridas por elemento.
export function PokemonSlot({ view, selected, onClick, onDropCard, dragData }: {
  view: PokemonView | null | undefined
  selected?: boolean
  onClick?: () => void
  // Recebe o dataTransfer de uma carta da mão largada neste slot (mesmo vazio).
  onDropCard?: (data: string) => void
  // Torna o Pokémon deste slot arrastável (ex.: promover do banco).
  dragData?: string
}) {
  const droppable = onDropCard !== undefined
  const dropProps = droppable ? {
    onDragOver: (e: React.DragEvent) => e.preventDefault(),
    onDrop: (e: React.DragEvent) => {
      e.preventDefault()
      onDropCard(e.dataTransfer.getData('text/plain'))
    },
  } : {}
  if (!view) return <div {...dropProps}><EmptySlot /></div>
  const energies = view.energies ?? []
  return (
    <div {...dropProps}>
      {/* key pelo id: evoluir troca a carta e remonta o nó → animação de entrada */}
      <Card key={view.card.id} view={view} selected={selected} onClick={onClick} dragData={dragData} />
      {(energies.length > 0 || view.tool) && (
        <div className="sub">
          {energies.map((e, i) => (
            <span key={i} className="edot" title={e.name}
              style={{ background: energyColor(e.nameEN) }} />
          ))}
          {view.tool && <span title={view.tool.name}> 🔧</span>}
        </div>
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
      <div style={{ position: 'relative', display: 'inline-block' }}>
        <Card view={discard![n - 1]} />
        <span className="cnt">{n}</span>
      </div>
    </div>
  )
}
