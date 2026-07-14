import type { CardView, SideView } from '../api'
import type { Sel } from '../api'
import { Card, DeckPile, DiscardPile, EmptySlot, PokemonSlot } from './Card'

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
      <div className="mlabel">Banco</div>
      <div className="benchrow">
        {Array.from({ length: 5 }, (_, i) => {
          const b = side.bench?.[i]
          const isPicking = isYou && !!b && (pickMode === 'all' || pickMode === 'bench')
          return (
            <PokemonSlot key={i} view={b}
              onClick={isYou && b ? () => onSelect('bench', i) : undefined}
              onDropCard={onDropHand ? data => onDropHand(i, data) : undefined}
              dragData={dragBench && b ? `bench:${i}` : undefined}
              placeholder={isYou ? '+ Pokémon' : undefined}
              selected={isYou && sel?.kind === 'bench' && sel.idx === i}
              picking={isPicking} />
          )
        })}
      </div>
    </div>
  )
}

function ActiveSpot({ side, isYou, sel, onSelect, onDropHand, pickMode }: {
  side: SideView
  isYou: boolean
  sel: Sel
  onSelect: (kind: 'active' | 'bench', idx: number) => void
  onDropHand?: (slot: number, data: string) => void
  pickMode?: 'all' | 'bench' | null
}) {
  const isPicking = isYou && !!side.active && pickMode === 'all'
  return (
    <div className="activewrap active">
      <div>
        <PokemonSlot view={side.active}
          onClick={isYou && side.active ? () => onSelect('active', -1) : undefined}
          onDropCard={onDropHand ? data => onDropHand(-1, data) : undefined}
          selected={isYou && sel?.kind === 'active'}
          picking={isPicking} />
      </div>
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

// Metade do tabuleiro do bot: pilhas à esquerda, banco em cima do ativo,
// Estádio (zona compartilhada) à direita.
export function BotMat({ side, stadium }: { side: SideView; stadium?: CardView }) {
  const noop = () => {}
  return (
    <section className="mat bot">
      <Piles side={side} />
      <div className="fieldcol">
        <Bench side={side} isYou={false} sel={null} onSelect={noop} />
        <ActiveSpot side={side} isYou={false} sel={null} onSelect={noop} />
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

// Metade do jogador: espelhada (ativo em cima do banco, pilhas à direita).
export function YouMat({ side, sel, onSelect, onDropHand, dragBench, pickMode }: {
  side: SideView
  sel: Sel
  onSelect: (kind: 'active' | 'bench', idx: number) => void
  onDropHand?: (slot: number, data: string) => void
  dragBench?: boolean
  // 'all' = ativo + banco destacados (pending target); 'bench' = só banco (retreating)
  pickMode?: 'all' | 'bench' | null
}) {
  return (
    <section className="mat you">
      <div className="fieldcol">
        <ActiveSpot side={side} isYou sel={sel} onSelect={onSelect} onDropHand={onDropHand} pickMode={pickMode} />
        <Bench side={side} isYou sel={sel} onSelect={onSelect} onDropHand={onDropHand} dragBench={dragBench} pickMode={pickMode} />
      </div>
      <Piles side={side} />
    </section>
  )
}
