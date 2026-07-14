import type { SideView } from '../api'

function PrizeColumn({ side }: { side: SideView }) {
  const n = side.prizes
  return (
    <div className="prizes">
      <h4>PRÊMIOS</h4>
      <div className="col" role="img"
        aria-label={`${n} prêmio${n !== 1 ? 's' : ''} restante${n !== 1 ? 's' : ''}`}>
        {Array.from({ length: 6 }, (_, i) => (
          <div key={i} className={'prize' + (i >= n ? ' taken' : '')} aria-hidden="true" />
        ))}
      </div>
    </div>
  )
}

export function Sidebar({ you, bot, current }: { you: SideView; bot: SideView; current: number }) {
  return (
    <aside id="left">
      <div className={'pp bot' + (current === 1 ? ' turn' : '')}>
        <div className="avatar">B</div>
        <div className="who">OPONENTE</div>
        <div className="sub">Bot · {bot.prizes} prêmios</div>
      </div>
      <PrizeColumn side={bot} />
      <div className={'pp you' + (current === 0 ? ' turn' : '')}>
        <div className="avatar">T</div>
        <div className="who">VOCÊ</div>
        <div className="sub">Treinador · {you.prizes} prêmios</div>
      </div>
      <PrizeColumn side={you} />
    </aside>
  )
}
