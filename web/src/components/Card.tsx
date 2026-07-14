import { useContext, useRef, useState, useEffect } from 'react'
import type { CardView, PokemonView } from '../api'
import { energyColor, energyDotStyle, energyImage } from '../energy'
import { PreviewCtx } from '../preview'

// Energias ligadas agrupadas por tipo: símbolo oficial + ×N quando N > 1.
// O símbolo é o centro do scan real da carta de energia (recorte circular via
// background); energia especial sem scan cai na bolinha de cor do tipo.
export function EnergyDots({ energies }: { energies: CardView[] }) {
  const groups = new Map<string, { name: string; nameEN: string; n: number }>()
  for (const e of energies) {
    const key = energyColor(e.nameEN)
    const g = groups.get(key)
    if (g) g.n++
    else groups.set(key, { name: e.name, nameEN: e.nameEN, n: 1 })
  }
  return (
    <>
      {[...groups.entries()].map(([color, g]) => (
        <span key={color} className="egroup" title={`${g.n}× ${g.name}`}>
          <span className="edot" style={energyDotStyle(g.nameEN)} />
          {g.n > 1 && <span className="ecount">×{g.n}</span>}
        </span>
      ))}
    </>
  )
}

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
  const [imgErr, setImgErr] = useState(false)
  useEffect(() => { setImgErr(false) }, [img])
  const prevDmg = useRef(pk?.damage ?? 0)
  const [hit, setHit] = useState(false)
  useEffect(() => {
    const d = pk?.damage ?? 0
    if (d > prevDmg.current) {
      setHit(true)
      const t = setTimeout(() => setHit(false), 450)
      prevDmg.current = d
      return () => clearTimeout(t)
    }
    prevDmg.current = d
  }, [pk?.damage])
  const cls = 'cardbox' + (onClick ? ' click' : '') + (selected ? ' sel' : '') + (hit ? ' hit' : '')
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
      {img && !imgErr
        ? <img className="card" src={img} alt={c.name} title={c.name} onError={() => setImgErr(true)} />
        : <div className="card txt">{c.name}</div>}
      {/* key pelo valor: mudar o dano remonta o nó → pulso de dmgpop */}
      {pk && pk.damage > 0 && <span key={pk.damage} className="dmg">{pk.damage}</span>}
      {/* energias/ferramenta como overlay na carta: nunca cortadas pelo layout */}
      {pk && ((pk.energies?.length ?? 0) > 0 || pk.tool) && (
        <span className="eside">
          <EnergyDots energies={pk.energies ?? []} />
          {pk.tool && <span className="tooldot" title={pk.tool.name}>{pk.tool.name.slice(0, 3).toUpperCase()}</span>}
        </span>
      )}
      {pk && pk.conditions.length > 0 && (
        <span className="cond">
          {pk.conditions.map(cd => {
            const [tag, bg] = COND[cd] ?? [cd.slice(0, 3).toUpperCase(), '#555']
            return <span key={cd} className="cbadge" style={{ background: bg }} title={cd} aria-label={cd}>{tag}</span>
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
    </div>
  )
}

export function DeckPile({ count }: { count: number }) {
  if (count <= 0) return <div className="pile empty" aria-label="Baralho vazio" />
  return (
    <div className="pile" aria-label={`Baralho: ${count} carta${count !== 1 ? 's' : ''}`}>
      <img className="back" src="/cardback.jpg" alt="" />
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
