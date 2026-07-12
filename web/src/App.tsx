import { useCallback, useEffect, useState } from 'react'
import { fetchState, postAction, postNew, type CardView, type GameConfig, type GameState } from './api'
import type { Sel } from './selection'
import { CardPreview, PreviewCtx, type Preview } from './preview'
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
  const mid = (hand.length - 1) / 2
  return (
    <div id="handtray">
      {hand.length === 0 && <span className="empty-hand">Mão vazia</span>}
      {hand.map((c, i) => {
        // leque: rotação/queda proporcionais à distância do centro da mão
        const off = i - mid
        const fan = hand.length > 1
          ? { transform: `rotate(${off * 2}deg) translateY(${Math.abs(off) * 3}px)` }
          : undefined
        return (
          <div key={i} className="fan" style={fan}>
            <Card view={c} onClick={() => onSelect(i)}
              dragData={c.category === 'Energy' ? `energy:${i}`
                : c.category === 'Pokemon'
                  ? (c.stage === 'Basic' ? `pokemon:${i}` : `evolve:${i}`)
                  : undefined}
              selected={sel?.kind === 'hand' && sel.idx === i} />
          </div>
        )
      })}
    </div>
  )
}

function RightRail({ pane, setPane, endTurn }: {
  pane: Pane
  setPane: (p: Pane) => void
  endTurn: () => void
}) {
  const toggle = (p: Pane) => setPane(pane === p ? '' : p)
  const tool = (p: Pane, label: string) => (
    <div className={'tool' + (pane === p ? ' on' : '')} onClick={() => toggle(p)}>
      {label}
    </div>
  )
  return (
    <aside id="right">
      {tool('cfg', 'PARTIDA')}
      <div className="spacer" />
      {tool('log', 'LOG')}
      {tool('arb', 'ARBITRAR')}
      <button id="endturn" onClick={endTurn}>TERMINAR<br />TURNO</button>
    </aside>
  )
}

const TYPES = ['Grass','Fire','Water','Lightning','Psychic','Fighting','Darkness','Metal','Dragon','Colorless']

function LobbyScreen({ onStart, err }: { onStart: (c: GameConfig) => void; err: string }) {
  const [mytype, setMytype] = useState('Fire')
  const [bottype, setBottype] = useState('Water')
  const [seed, setSeed] = useState(1)
  return (
    <div id="lobby">
      <div className="lobby-box">
        <div className="lobby-title">Pokémon TCG</div>
        <div className="lobby-row">
          <label>Seu tipo</label>
          <select value={mytype} onChange={e => setMytype(e.target.value)}>
            {TYPES.map(t => <option key={t}>{t}</option>)}
          </select>
        </div>
        <div className="lobby-row">
          <label>Tipo do bot</label>
          <select value={bottype} onChange={e => setBottype(e.target.value)}>
            {TYPES.map(t => <option key={t}>{t}</option>)}
          </select>
        </div>
        <div className="lobby-row">
          <label>Seed</label>
          <input type="number" min={1} value={seed} onChange={e => setSeed(Math.max(1, +e.target.value))} />
        </div>
        {err && <div className="lobby-err">{err}</div>}
        <button className="primary" onClick={() => onStart({ mytype, bottype, seed })}>
          Iniciar partida
        </button>
      </div>
    </div>
  )
}

function WinnerOverlay({ winner, onReplay, onNew }: { winner: number; onReplay: () => void; onNew: () => void }) {
  if (winner < 0 && winner !== -2) return null
  const txt = winner === -2 ? 'Sudden Death!' : winner === 0 ? 'Você venceu!' : 'Bot venceu.'
  return (
    <div id="winner">
      <div className="winner-box">
        <div className="winner-txt">{txt}</div>
        <div style={{ display:'flex', gap:8, justifyContent:'center' }}>
          <button className="primary" onClick={onReplay}>Repetir</button>
          <button onClick={onNew}>Nova partida</button>
        </div>
      </div>
    </div>
  )
}

export default function App() {
  const [s, setS] = useState<GameState | null>(null)
  const [config, setConfig] = useState<GameConfig>({ mytype: 'Fire', bottype: 'Water', seed: 1 })
  const [lobbyErr, setLobbyErr] = useState('')
  const [sel, setSel] = useState<Sel>(null)
  const [err, setErr] = useState('')
  const [pane, setPane] = useState<Pane>('')
  const [preview, setPreview] = useState<Preview | null>(null)
  const publishPreview = useCallback((c: CardView | null, rect?: DOMRect) =>
    setPreview(c && rect ? { card: c, rect } : null), [])

  useEffect(() => { fetchState().then(setS) }, [])

  const startGame = useCallback((c: GameConfig) => {
    setConfig(c)
    postNew(c).then(j => {
      if (j.phase === 'lobby') {
        setLobbyErr(j.error ?? 'Erro ao criar partida')
        setS(null)
      } else {
        setLobbyErr('')
        setS(j)
        setSel(null)
        setErr('')
      }
    })
  }, [])

  const post = useCallback((body: Record<string, unknown>) => {
    postAction(body).then(j => {
      setErr(j.error ?? '')
      setS(j)
      setSel(null)
    })
  }, [])

  const select = (kind: 'hand' | 'active' | 'bench', idx: number) =>
    setSel(cur => (cur && cur.kind === kind && cur.idx === idx) ? null : { kind, idx })

  if (!s || s.phase === 'lobby') {
    return <LobbyScreen onStart={startGame} err={lobbyErr} />
  }

  return (
    <PreviewCtx.Provider value={publishPreview}>
      <div id="app">
        <Sidebar you={s.you} bot={s.bot} current={s.current} />
        <div id="center">
          <BotMat side={s.bot} stadium={s.stadium} />
          <YouMat side={s.you} sel={sel} onSelect={select}
            dragBench={!!s.needPromote?.[0]}
            onDropHand={(slot, data) => {
              const [kind, idx] = data.split(':')
              const i = parseInt(idx)
              if (isNaN(i)) return
              if (kind === 'energy') post({ action: 'attach_energy', hand: i, slot })
              else if (kind === 'evolve') post({ action: 'evolve', hand: i, slot })
              else if (kind === 'bench') {
                if (slot === -1) post({ action: 'promote', bench: i })
              } else if (kind === 'pokemon') {
                if (slot === -1 && !s.you.active) post({ action: 'place_active', hand: i })
                else post({ action: 'place_bench', hand: i })
              }
            }} />
          <ActionBar s={s} sel={sel} err={err} post={post} />
          <HandTray s={s} sel={sel} onSelect={i => select('hand', i)} />
        </div>
        <RightRail pane={pane} setPane={setPane} endTurn={() => post({ action: 'end_turn' })} />
        <Drawer pane={pane} s={s} post={post} />
        <CardPreview p={preview} />
        <WinnerOverlay
          winner={s.winner}
          onReplay={() => startGame(config)}
          onNew={() => setS(null)}
        />
      </div>
    </PreviewCtx.Provider>
  )
}
