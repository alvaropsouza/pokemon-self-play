import type { CardView, SideView } from '../api'
import type { Sel } from '../selection'
import { Card, DeckPile, DiscardPile, EmptySlot, PokemonSlot } from './Card'

function Bench({ side, isYou, sel, onSelect }: {
  side: SideView
  isYou: boolean
  sel: Sel
  onSelect: (kind: 'active' | 'bench', idx: number) => void
}) {
  return (
    <div>
      <div className="mlabel">Área de Banco</div>
      <div className="benchrow">
        {Array.from({ length: 5 }, (_, i) => {
          const b = side.bench?.[i]
          return (
            <PokemonSlot key={i} view={b}
              onClick={isYou && b ? () => onSelect('bench', i) : undefined}
              selected={isYou && sel?.kind === 'bench' && sel.idx === i} />
          )
        })}
      </div>
    </div>
  )
}

function ActiveSpot({ side, isYou, sel, onSelect }: {
  side: SideView
  isYou: boolean
  sel: Sel
  onSelect: (kind: 'active' | 'bench', idx: number) => void
}) {
  return (
    <div className="activewrap active">
      <div>
        <div className="mlabel">Pokémon Ativo</div>
        <PokemonSlot view={side.active}
          onClick={isYou && side.active ? () => onSelect('active', -1) : undefined}
          selected={isYou && sel?.kind === 'active'} />
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
export function YouMat({ side, sel, onSelect }: {
  side: SideView
  sel: Sel
  onSelect: (kind: 'active' | 'bench', idx: number) => void
}) {
  return (
    <section className="mat you">
      <div className="fieldcol">
        <ActiveSpot side={side} isYou sel={sel} onSelect={onSelect} />
        <Bench side={side} isYou sel={sel} onSelect={onSelect} />
      </div>
      <Piles side={side} />
    </section>
  )
}
