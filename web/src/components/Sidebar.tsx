import type { SideView } from '../api'

function PrizeColumn({ side }: { side: SideView }) {
  return (
    <div className="prizes">
      <h4>PRÊMIOS</h4>
      <div className="col">
        {Array.from({ length: 6 }, (_, i) => (
          <div key={i} className={'prize' + (i >= side.prizes ? ' taken' : '')} />
        ))}
      </div>
    </div>
  )
}

export function Sidebar({ you, bot }: { you: SideView; bot: SideView }) {
  return (
    <aside id="left">
      <div className="pp bot">
        <div className="who">OPONENTE</div>
        <div className="avatar">🤖</div>
        <div className="sub">Bot</div>
      </div>
      <PrizeColumn side={bot} />
      <div className="pp you">
        <div className="who">VOCÊ</div>
        <div className="avatar">🧑</div>
        <div className="sub">Treinador</div>
      </div>
      <PrizeColumn side={you} />
    </aside>
  )
}
