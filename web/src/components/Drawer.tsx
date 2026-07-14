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
        <input type="number" value={amount} step={10} min={10} max={9999} onChange={e => setAmount(Math.max(10, Math.min(9999, +e.target.value)))} />
      </div>
      <div className="row">
        <button type="button" onClick={() => arb('arb_damage')}>Dano</button>
        <button type="button" onClick={() => arb('arb_heal')}>Cura</button>
        <button type="button" onClick={() => arb('arb_draw')}>Comprar N</button>
        <button type="button" onClick={() => arb('arb_switch')}>Trocar Ativo</button>
        <button type="button" onClick={() => arb('arb_shuffle')}>Embaralhar</button>
      </div>
      <div className="row">
        <select value={cond} onChange={e => setCond(e.target.value)}>
          {['poisoned', 'burned', 'asleep', 'confused', 'paralyzed'].map(c => <option key={c}>{c}</option>)}
        </select>
        <button type="button" onClick={() => post({ action: 'arb_condition', player, condition: cond })}>Aplicar condição</button>
      </div>
    </div>
  )
}

function CfgPane({ s, onExit }: { s: GameState; onExit: () => void }) {
  // Confirmação inline em 2 passos: sair descarta a partida em andamento.
  const [confirming, setConfirming] = useState(false)
  return (
    <div className="pane-cfg">
      <h3>Partida</h3>
      <div className="cfg-info">
        Turno: {s.turn}<br />
        Fase: {s.phase}<br />
        Vez: {s.current === 0 ? 'você' : 'bot'}<br />
        Seu deck: {s.you.deck} cartas · descarte: {s.you.discard?.length ?? 0}<br />
        Deck do bot: {s.bot.deck} cartas · mão: {s.bot.handCount}
      </div>
      <div className="cfg-exit">
        {!confirming ? (
          <button type="button" className="exit-btn" onClick={() => setConfirming(true)}>
            Sair da partida
          </button>
        ) : (
          <div className="exit-confirm" role="alertdialog" aria-label="Confirmar saída da partida">
            <span>Abandonar a partida e voltar à seleção de decks?</span>
            <div className="exit-row">
              <button type="button" className="exit-btn danger" onClick={onExit}>Sair</button>
              <button type="button" className="exit-btn" onClick={() => setConfirming(false)}>Continuar jogando</button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

export function Drawer({ pane, s, post, onExit }: {
  pane: Pane
  s: GameState
  post: (body: Record<string, unknown>) => void
  onExit: () => void
}) {
  if (!pane) return null
  const label = pane === 'log' ? 'Log da partida' : pane === 'arb' ? 'Arbitragem' : 'Partida'
  return (
    <div id="drawer" role="dialog" aria-label={label}>
      {pane === 'log' && <LogPane log={s.log ?? []} />}
      {pane === 'arb' && <ArbPane post={post} />}
      {pane === 'cfg' && <CfgPane s={s} onExit={onExit} />}
    </div>
  )
}
