import { useCallback, useEffect, useState } from 'react'
import { fetchState, postAction, type GameState } from './api'
import type { Sel } from './selection'
import { Sidebar } from './components/Sidebar'
import { BotMat, YouMat } from './components/Mat'
import { ActionBar } from './components/ActionBar'
import { Card } from './components/Card'
import { Drawer, type Pane } from './components/Drawer'

function HandTray({ s, sel, onSelect }: {
  s: GameState
  sel: Sel
  onSelect: (idx: number) => void
}) {
  const hand = s.you.hand ?? []
  return (
    <div id="handtray">
      {hand.length === 0 && <span className="empty-hand">Mão vazia</span>}
      {hand.map((c, i) => (
        <Card key={i} view={c} onClick={() => onSelect(i)}
          selected={sel?.kind === 'hand' && sel.idx === i} />
      ))}
    </div>
  )
}

function RightRail({ pane, setPane, endTurn }: {
  pane: Pane
  setPane: (p: Pane) => void
  endTurn: () => void
}) {
  const toggle = (p: Pane) => setPane(pane === p ? '' : p)
  const tool = (p: Pane, ico: string, label: string) => (
    <div className={'tool' + (pane === p ? ' on' : '')} onClick={() => toggle(p)}>
      <span className="ico">{ico}</span>{label}
    </div>
  )
  return (
    <aside id="right">
      {tool('cfg', '⚙️', 'PARTIDA')}
      <div className="spacer" />
      {tool('log', '💬', 'CHAT')}
      {tool('arb', '📋', 'AÇÕES')}
      <button id="endturn" onClick={endTurn}>⏳<br />TERMINAR<br />TURNO</button>
    </aside>
  )
}

function WinnerOverlay({ winner }: { winner: number }) {
  if (winner < 0 && winner !== -2) return null
  const txt = winner === -2 ? 'Sudden Death!' : winner === 0 ? '🏆 Você venceu!' : '🤖 Bot venceu.'
  return <div id="winner"><div>{txt}</div></div>
}

export default function App() {
  const [s, setS] = useState<GameState | null>(null)
  const [sel, setSel] = useState<Sel>(null)
  const [err, setErr] = useState('')
  const [pane, setPane] = useState<Pane>('')

  useEffect(() => { fetchState().then(setS) }, [])

  const post = useCallback((body: Record<string, unknown>) => {
    postAction(body).then(j => {
      setErr(j.error ?? '')
      setS(j)
      setSel(null)
    })
  }, [])

  const select = (kind: 'hand' | 'active' | 'bench', idx: number) =>
    setSel(cur => (cur && cur.kind === kind && cur.idx === idx) ? null : { kind, idx })

  if (!s) return null

  return (
    <div id="app">
      <Sidebar you={s.you} bot={s.bot} />
      <div id="center">
        <BotMat side={s.bot} stadium={s.stadium} />
        <YouMat side={s.you} sel={sel} onSelect={select} />
        <ActionBar s={s} sel={sel} err={err} post={post} />
        <HandTray s={s} sel={sel} onSelect={i => select('hand', i)} />
      </div>
      <RightRail pane={pane} setPane={setPane} endTurn={() => post({ action: 'end_turn' })} />
      <Drawer pane={pane} s={s} post={post} />
      <WinnerOverlay winner={s.winner} />
    </div>
  )
}
