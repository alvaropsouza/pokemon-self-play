import { useContext, useState } from 'react'
import type { CardView, PokemonView } from '../api'
import { energyColor, energyImage } from '../energy'
import { PreviewCtx } from '../preview'

// Sigla + cor por condição especial (sem emoji — PRODUCT.md).
export const COND: Record<string, [string, string]> = {
  poisoned: ['PSN', '#7c4a8c'], burned: ['BRN', '#c04a36'], asleep: ['SLP', '#4a6a8c'],
  confused: ['CNF', '#b0507e'], paralyzed: ['PAR', '#a8862a'],
}

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
      role={onClick ? 'button' : undefined}
      tabIndex={onClick ? 0 : undefined}
      onKeyDown={onClick
        ? e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onClick() } }
        : undefined}
      draggable={dragData !== undefined}
      onDragStart={dragData !== undefined
        ? e => { e.dataTransfer.setData('text/plain', dragData); setPreview(null) }
        : undefined}
      onMouseEnter={e => setPreview(c, e.currentTarget.getBoundingClientRect())}
      onMouseLeave={() => setPreview(null)}>
      {img
        ? <img className="card" src={img} title={c.name} alt={c.name} />
        : <div className="card txt">{c.name}</div>}
      {/* key pelo valor: mudar o dano remonta o nó → pulso de dmgpop */}
      {pk && pk.damage > 0 && <span key={pk.damage} className="dmg">{pk.damage}</span>}
      {pk && pk.conditions.length > 0 && (
        <span className="cond">
          {pk.conditions.map(cd => {
            const [tag, bg] = COND[cd] ?? [cd.slice(0, 3).toUpperCase(), '#555']
            return <span key={cd} className="cbadge" style={{ background: bg }} title={cd}>{tag}</span>
          })}
        </span>
      )}
    </div>
  )
}

export function EmptySlot({ label }: { label?: string }) {
  return <div className="slot">{label}</div>
}

// HP restante visível sem abrir a carta: barra + texto.
function HpGauge({ view }: { view: PokemonView }) {
  const hp = view.card.hp
  if (hp <= 0) return null
  const rem = Math.max(0, hp - view.damage)
  const pct = rem / hp
  const cls = pct <= 0.25 ? 'low' : pct <= 0.5 ? 'mid' : ''
  return (
    <div className="hp">
      <div className="hpbar"><i className={cls} style={{ width: `${pct * 100}%` }} /></div>
      <div className="hptxt">{rem}/{hp} PS</div>
    </div>
  )
}

// Slot de Pokémon em jogo (ativo/banco); vazio vira slot tracejado.
// Energias ligadas aparecem como bolinhas coloridas por elemento.
export function PokemonSlot({ view, selected, onClick, onDropCard, dragData, placeholder, picking }: {
  view: PokemonView | null | undefined
  selected?: boolean
  onClick?: () => void
  onDropCard?: (data: string) => void
  dragData?: string
  placeholder?: string
  // Destaque verde pulsante: slot é alvo válido no modo pick inline.
  picking?: boolean
}) {
  const droppable = onDropCard !== undefined
  // Realce verde do alvo durante drag-over (feedback de ação válida).
  const [over, setOver] = useState(false)
  const dropProps = droppable ? {
    onDragOver: (e: React.DragEvent) => { e.preventDefault(); setOver(true) },
    onDragLeave: () => setOver(false),
    onDrop: (e: React.DragEvent) => {
      e.preventDefault()
      setOver(false)
      onDropCard(e.dataTransfer.getData('text/plain'))
    },
  } : {}
  // Em modo picking, o slot inteiro (incluindo padding do .base) responde ao clique.
  // onClick fica só no Card quando não estamos em picking, evitando duplo disparo.
  const containerClick = picking ? onClick : undefined
  const cardClick = picking ? undefined : onClick
  const cls = 'base' + (over ? ' over' : '') + (picking && !over ? ' picking' : '')
  if (!view) return <div className={cls} {...dropProps}><EmptySlot label={droppable ? placeholder : undefined} /></div>
  const energies = view.energies ?? []
  return (
    <div className={cls} {...dropProps}
      onClick={containerClick}
      role={containerClick ? 'button' : undefined}
      tabIndex={containerClick ? 0 : undefined}
      onKeyDown={containerClick
        ? e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); containerClick() } }
        : undefined}>
      {/* key pelo id: evoluir troca a carta e remonta o nó → animação de entrada */}
      <Card key={view.card.id} view={view} selected={selected} onClick={cardClick} dragData={dragData} />
      <HpGauge view={view} />
      {(energies.length > 0 || view.tool) && (
        <div className="sub">
          {energies.map((e, i) => (
            <span key={i} className="edot" title={e.name}
              style={{ background: energyColor(e.nameEN) }} />
          ))}
          {view.tool && <span className="tooldot" title={view.tool.name}>{view.tool.name.slice(0, 3).toUpperCase()}</span>}
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
