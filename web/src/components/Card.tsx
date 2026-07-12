import type { CardView, PokemonView } from '../api'
import { energyStyle } from '../energy'
import { usePreview } from '../preview'

// Carta de energia desenhada em CSS: energias básicas do TCGdex não têm imagem.
function EnergyFace({ c }: { c: CardView }) {
  const s = energyStyle(c.nameEN)
  return (
    <div className="card energycard" style={{ background: s.color }} title={c.name}>
      <span className="etype">ENERGIA</span>
      <span className="eicon">{s.icon}</span>
      <span className="ename">{c.name}</span>
    </div>
  )
}

// Carta genérica: imagem quando existe; energia sem imagem vira EnergyFace;
// demais sem imagem, fallback textual. Overlays de dano e condições quando o
// view é um Pokémon em jogo. Hover publica a carta no painel de preview.
export function Card({ view, selected, onClick }: {
  view: CardView | PokemonView
  selected?: boolean
  onClick?: () => void
}) {
  const c = 'card' in view ? view.card : view
  const pk = 'card' in view ? view : null
  const setPreview = usePreview()
  const cls = 'cardbox' + (onClick ? ' click' : '') + (selected ? ' sel' : '')
  return (
    <div className={cls} onClick={onClick}
      onMouseEnter={() => setPreview(c)} onMouseLeave={() => setPreview(null)}>
      {c.image
        ? <img className="card" src={c.image} title={c.name} alt={c.name} />
        : c.category === 'Energy'
          ? <EnergyFace c={c} />
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
export function PokemonSlot({ view, selected, onClick }: {
  view: PokemonView | null | undefined
  selected?: boolean
  onClick?: () => void
}) {
  if (!view) return <div><EmptySlot /></div>
  const energies = view.energies ?? []
  return (
    <div>
      <Card view={view} selected={selected} onClick={onClick} />
      {(energies.length > 0 || view.tool) && (
        <div className="sub">
          {energies.map((e, i) => (
            <span key={i} className="edot" title={e.name}
              style={{ background: energyStyle(e.nameEN).color }} />
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
