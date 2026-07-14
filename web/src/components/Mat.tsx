import type { CardView, PokemonView, SideView } from '../api'
import type { Sel } from '../api'
import { COND, Card, DeckPile, DiscardPile, EmptySlot, EnergyDots, PokemonSlot } from './Card'
import { EnergyCost, canPay } from './ActionBar'

// Golpes do Ativo do bot, somente leitura — informação pública (carta + energias
// visíveis); apagado quando a energia ligada não paga o custo.
function MoveList({ view }: { view: PokemonView }) {
  if (!view.card.attacks?.length) return null
  return (
    <div className="movebox">
      {view.card.attacks.map((a, i) => (
        <div key={i} className={'move ro' + (canPay(a.cost, view.energies ?? []) ? '' : ' off')}>
          <EnergyCost cost={a.cost} />
          <span className="move-name">{a.name}</span>
          <span className="move-dmg">{a.damage || 'efeito'}</span>
        </div>
      ))}
    </div>
  )
}

// HUD de batalha estilo Game Boy: nome, HP em barra colorida, condições,
// energias e ferramenta do Pokémon Ativo. Substitui os overlays na carta.
function BattleHud({ view }: { view: PokemonView | null | undefined }) {
  if (!view) return <div className="hud empty">aguardando Pokémon</div>
  const hp = view.card.hp
  const rem = Math.max(0, hp - view.damage)
  const pct = hp > 0 ? rem / hp : 0
  const barCls = pct <= 0.25 ? 'low' : pct <= 0.5 ? 'mid' : ''
  const energies = view.energies ?? []
  return (
    <div className="hud">
      <div className="hud-name">
        <span>{view.card.name}</span>
        {view.conditions.length > 0 && (
          <span>
            {view.conditions.map(cd => {
              const [tag, bg] = COND[cd] ?? [cd.slice(0, 3).toUpperCase(), '#555']
              return <span key={cd} className="cbadge" style={{ background: bg }} title={cd}>{tag}</span>
            })}
          </span>
        )}
      </div>
      {hp > 0 && (
        <div className="hud-hpline">
          <span className="hud-hplabel">HP</span>
          <span className="hud-bar"><i className={barCls} style={{ width: `${pct * 100}%` }} /></span>
        </div>
      )}
      <div className="hud-foot">
        {hp > 0 && <span className="hud-hptxt">{rem}/{hp}</span>}
        <span className="hud-attach">
          <EnergyDots energies={energies} />
          {view.tool && <span className="tooldot" title={view.tool.name}>{view.tool.name.slice(0, 3).toUpperCase()}</span>}
        </span>
      </div>
    </div>
  )
}

function Bench({ side, isYou, sel, onSelect, onDropHand, dragBench, pickMode }: {
  side: SideView
  isYou: boolean
  sel: Sel
  onSelect: (kind: 'active' | 'bench', idx: number) => void
  onDropHand?: (slot: number, data: string) => void
  dragBench?: boolean
  pickMode?: 'all' | 'bench' | null
}) {
  return (
    <div>
      <div className="benchrow">
        {Array.from({ length: 5 }, (_, i) => {
          const b = side.bench?.[i]
          const isPicking = isYou && !!b && (pickMode === 'all' || pickMode === 'bench')
          return (
            <PokemonSlot key={i} view={b}
              onClick={isYou && b ? () => onSelect('bench', i) : undefined}
              onDropCard={onDropHand ? data => onDropHand(i, data) : undefined}
              dragData={dragBench && b ? `bench:${i}` : undefined}
              selected={isYou && sel?.kind === 'bench' && sel.idx === i}
              picking={isPicking} />
          )
        })}
      </div>
    </div>
  )
}

// Palco de batalha: Ativo sobre plataforma elíptica + HUD ao lado, como na
// tela de batalha do Game Boy (inimigo à direita-cima, jogador à esquerda-baixo).
function Stage({ side, isYou, sel, onSelect, onDropHand, pickMode, menu }: {
  side: SideView
  isYou: boolean
  sel: Sel
  onSelect: (kind: 'active' | 'bench', idx: number) => void
  onDropHand?: (slot: number, data: string) => void
  pickMode?: 'all' | 'bench' | null
  // menu de golpes (AttackMenu), renderizado abaixo do HUD do jogador
  menu?: React.ReactNode
}) {
  const isPicking = isYou && !!side.active && pickMode === 'all'
  const slot = (
    <div className="platform-wrap">
      <PokemonSlot view={side.active}
        onClick={isYou && side.active ? () => onSelect('active', -1) : undefined}
        onDropCard={onDropHand ? data => onDropHand(-1, data) : undefined}
        selected={isYou && sel?.kind === 'active'}
        picking={isPicking} />
    </div>
  )
  const hud = <div className="hud-stack"><BattleHud view={side.active} />{menu}</div>
  // Bot: HUD à esquerda, Pokémon à direita. Jogador: espelhado.
  return (
    <div className={'stage ' + (isYou ? 'you' : 'bot')}>
      {isYou ? <>{slot}{hud}</> : <>{hud}{slot}</>}
    </div>
  )
}

function Piles({ side }: { side: SideView }) {
  return (
    <div className="pilescol">
      <div>
        <div className="mlabel">Baralho</div>
        <DeckPile count={side.deck} />
      </div>
      <div>
        <div className="mlabel">Descarte</div>
        <DiscardPile discard={side.discard} />
      </div>
    </div>
  )
}

// Metade do bot: pilhas à esquerda, banco no fundo (topo da tela), Ativo em
// plataforma na frente (junto à costura central); Estádio à direita.
export function BotMat({ side, stadium }: { side: SideView; stadium?: CardView }) {
  const noop = () => {}
  return (
    <section className="mat bot" aria-label="Lado do bot">
      <Piles side={side} />
      <div className="battlefield">
        <Bench side={side} isYou={false} sel={null} onSelect={noop} />
        <Stage side={side} isYou={false} sel={null} onSelect={noop}
          menu={side.active ? <MoveList view={side.active} /> : undefined} />
      </div>
      <div className="apoiocol">
        <div className="apoio">
          <div className="mlabel">Estádio</div>
          {stadium ? <Card view={stadium} /> : <EmptySlot />}
        </div>
      </div>
    </section>
  )
}

// Metade do jogador: espelhada — Ativo na frente (junto à costura), banco no
// fundo (base da tela), pilhas à direita.
export function YouMat({ side, sel, onSelect, onDropHand, dragBench, pickMode, menu }: {
  side: SideView
  sel: Sel
  onSelect: (kind: 'active' | 'bench', idx: number) => void
  onDropHand?: (slot: number, data: string) => void
  dragBench?: boolean
  // 'all' = ativo + banco destacados (pending target); 'bench' = só banco (retreating)
  pickMode?: 'all' | 'bench' | null
  menu?: React.ReactNode
}) {
  return (
    <section className="mat you" aria-label="Seu lado">
      <div className="battlefield">
        <Stage side={side} isYou sel={sel} onSelect={onSelect} onDropHand={onDropHand} pickMode={pickMode} menu={menu} />
        <Bench side={side} isYou sel={sel} onSelect={onSelect} onDropHand={onDropHand} dragBench={dragBench} pickMode={pickMode} />
      </div>
      <Piles side={side} />
    </section>
  )
}
