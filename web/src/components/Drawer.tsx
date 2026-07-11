import { useEffect, useRef, useState } from 'react'
import type { GameState } from '../api'

export type Pane = '' | 'log' | 'arb' | 'cfg'

function LogPane({ log }: { log: string[] }) {
  const ref = useRef<HTMLDivElement>(null)
  useEffect(() => {
    if (ref.current) ref.current.scrollTop = ref.current.scrollHeight
  }, [log])
  return (
    <div className="pane-log">
      <h3>Log da partida</h3>
      <div id="log" ref={ref}>
        {log.map((l, i) => <div key={i}>{l}</div>)}
      </div>
    </div>
  )
}

// Arbitragem manual: efeitos de carta que o motor não interpreta (CLAUDE.md).
function ArbPane({ post }: { post: (body: Record<string, unknown>) => void }) {
  const [player, setPlayer] = useState(0)
  const [slot, setSlot] = useState(-1)
  const [amount, setAmount] = useState(10)
  const [cond, setCond] = useState('poisoned')
  const arb = (action: string) => post({ action, player, slot, bench: slot, amount })
  return (
    <div className="arb">
      <h3>Arbitragem manual (efeitos de carta)</h3>
      <div className="row">
        Lado: <select value={player} onChange={e => setPlayer(+e.target.value)}>
          <option value={0}>você</option><option value={1}>bot</option>
        </select>
        Alvo: <select value={slot} onChange={e => setSlot(+e.target.value)}>
          <option value={-1}>Ativo</option>
          {[0, 1, 2, 3, 4].map(i => <option key={i} value={i}>Banco {i + 1}</option>)}
        </select>
        <input type="number" value={amount} step={10} onChange={e => setAmount(+e.target.value)} />
      </div>
      <div className="row">
        <button onClick={() => arb('arb_damage')}>Dano</button>
        <button onClick={() => arb('arb_heal')}>Cura</button>
        <button onClick={() => arb('arb_draw')}>Comprar N</button>
        <button onClick={() => arb('arb_switch')}>Trocar Ativo</button>
        <button onClick={() => arb('arb_shuffle')}>Embaralhar</button>
      </div>
      <div className="row">
        <select value={cond} onChange={e => setCond(e.target.value)}>
          {['poisoned', 'burned', 'asleep', 'confused', 'paralyzed'].map(c => <option key={c}>{c}</option>)}
        </select>
        <button onClick={() => post({ action: 'arb_condition', player, condition: cond })}>Aplicar condição</button>
      </div>
    </div>
  )
}

function CfgPane({ s }: { s: GameState }) {
  return (
    <div>
      <h3>Partida</h3>
      <div className="cfg-info">
        Turno: {s.turn}<br />
        Fase: {s.phase}<br />
        Vez: {s.current === 0 ? 'você' : 'bot'}<br />
        Seu deck: {s.you.deck} cartas · descarte: {s.you.discard?.length ?? 0}<br />
        Deck do bot: {s.bot.deck} cartas · mão: {s.bot.handCount}
      </div>
    </div>
  )
}

export function Drawer({ pane, s, post }: {
  pane: Pane
  s: GameState
  post: (body: Record<string, unknown>) => void
}) {
  if (!pane) return null
  return (
    <div id="drawer">
      {pane === 'log' && <LogPane log={s.log ?? []} />}
      {pane === 'arb' && <ArbPane post={post} />}
      {pane === 'cfg' && <CfgPane s={s} />}
    </div>
  )
}
