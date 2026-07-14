import { useCallback, useEffect, useLayoutEffect, useRef, useState, type CSSProperties } from 'react'
import { fetchState, postAction, postNew, type CardView, type GameConfig, type GameState, type Sel } from './api'
import { energyColor } from './energy'
import { cancelFlights, flyFromDeck } from './drawfx'
import { CardPreview, PreviewCtx, type Preview } from './preview'
import { Sidebar } from './components/Sidebar'
import { BotMat, YouMat } from './components/Mat'
import { ActionBar, AttackMenu } from './components/ActionBar'
import { Card } from './components/Card'
import { Drawer, type Pane } from './components/Drawer'

function HandTray({ s, sel, onSelect }: {
  s: GameState
  sel: Sel
  onSelect: (idx: number) => void
}) {
  const hand = s.you.hand ?? []
  const mid = (hand.length - 1) / 2
  const trayRef = useRef<HTMLDivElement>(null)
  const prev = useRef({ len: hand.length, deck: s.you.deck, rects: [] as DOMRect[] })

  // Compra detectada (mão cresceu): cartas novas voam do baralho (drawfx);
  // as existentes deslizam para a nova posição do leque (FLIP). Tudo medido
  // antes de qualquer animação começar para os rects serem os de layout puro.
  useLayoutEffect(() => {
    const boxes = Array.from(trayRef.current?.querySelectorAll<HTMLElement>('.cardbox') ?? [])
    const rects = boxes.map(el => el.getBoundingClientRect())
    const p = prev.current
    const grew = hand.length > p.len
    if (grew && !matchMedia('(prefers-reduced-motion: reduce)').matches) {
      cancelFlights()
      boxes.slice(0, p.len).forEach((el, i) => {
        const r = p.rects[i]
        if (!r) return
        const ddx = r.left - rects[i].left
        const ddy = r.top - rects[i].top
        if (Math.abs(ddx) + Math.abs(ddy) > 1) el.animate(
          [{ transform: `translate(${ddx}px, ${ddy}px)` }, { transform: 'none' }],
          { duration: 250, easing: 'cubic-bezier(0.22,1,0.36,1)' })
      })
      const fromDeck = s.you.deck < p.deck
      boxes.slice(p.len).forEach((el, j) => {
        const i = p.len + j
        if (fromDeck) flyFromDeck(el, rects[i], (i - mid) * 2, j * 80)
        else el.animate([{ opacity: 0 }, { opacity: 1 }], { duration: 200 })
      })
    }
    prev.current = { len: hand.length, deck: s.you.deck, rects }
  })

  return (
    <div id="handtray" ref={trayRef}>
      {hand.length === 0 && <span className="empty-hand">Mão vazia</span>}
      {hand.map((c, i) => {
        // leque: rotação/queda proporcionais à distância do centro da mão
        const off = i - mid
        const fan = hand.length > 1
          ? { transform: `rotate(${off * 2}deg) translateY(${Math.abs(off) * 3}px)` }
          : undefined
        return (
          <div key={`${c.id}-${i}`} className="fan" style={fan}>
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

const TYPES: [string, string][] = [
  ['Grass', 'Planta'], ['Fire', 'Fogo'], ['Water', 'Água'], ['Lightning', 'Elétrico'],
  ['Psychic', 'Psíquico'], ['Fighting', 'Lutador'], ['Darkness', 'Escuridão'],
  ['Metal', 'Metal'], ['Dragon', 'Dragão'], ['Colorless', 'Incolor'],
]

function TypePicker({ value, onChange }: { value: string; onChange: (t: string) => void }) {
  return (
    <div className="typegrid">
      {TYPES.map(([en, pt]) => (
        <button key={en} type="button"
          className={'typechip' + (value === en ? ' on' : '')}
          style={{ '--el': energyColor(en) } as CSSProperties}
          onClick={() => onChange(en)}>
          <span className="edot" style={{ background: energyColor(en) }} />{pt}
        </button>
      ))}
    </div>
  )
}

function LobbyScreen({ onStart, err }: { onStart: (c: GameConfig) => void; err: string }) {
  const [mytype, setMytype] = useState('Fire')
  const [bottype, setBottype] = useState('Water')
  return (
    <div id="lobby">
      <div className="lobby-box">
        <div className="lobby-head">
          <div className="lobby-title">Pokémon TCG</div>
          <div className="lobby-sub">Escolha os tipos de energia dos dois decks</div>
        </div>
        <div className="lobby-sides">
          <section className="lobby-side you">
            <h2>Você</h2>
            <TypePicker value={mytype} onChange={setMytype} />
          </section>
          <div className="lobby-vs" aria-hidden="true">vs</div>
          <section className="lobby-side bot">
            <h2>Bot</h2>
            <TypePicker value={bottype} onChange={setBottype} />
          </section>
        </div>
        {err && <div className="lobby-err">{err}</div>}
        <button className="primary lobby-start" onClick={() => onStart({ mytype, bottype })}>
          Iniciar partida
        </button>
      </div>
    </div>
  )
}

// Overlay de escolha pendente: busca no deck ou troca de Pokémon Ativo.
function ChoiceOverlay({ pc, post }: {
  pc: NonNullable<GameState['pendingChoice']>
  post: (body: Record<string, unknown>) => void
}) {
  const [picks, setPicks] = useState<number[]>([])
  const toggle = (i: number) => setPicks(cur =>
    cur.includes(i) ? cur.filter(x => x !== i)
      : cur.length < pc.max ? [...cur, i] : cur)

  const isSwitch = pc.kind === 'switch_self' || pc.kind === 'switch_opp'
  const title = pc.kind === 'switch_self'
    ? 'Escolha 1 Pokémon do Banco para trocar com o Ativo'
    : pc.kind === 'switch_opp'
      ? 'Escolha 1 Pokémon do Banco do oponente para virar Ativo'
      : `Busca no deck: escolha até ${pc.max} carta${pc.max > 1 ? 's' : ''} ${pc.dest === 'bench' ? 'para o Banco' : 'para a mão'}`

  const confirm = () => post({ action: 'resolve_choice', picks })
  const skip = () => post({ action: 'resolve_choice', picks: [] })

  return (
    <div className="choice-overlay">
      <div className="winner-box choice-box">
        <div className="choice-title">{title}</div>
        <div className="choice-cards">
          {pc.candidates.map((c, i) => (
            <Card key={i} view={c} selected={picks.includes(i)} onClick={() => toggle(i)} />
          ))}
        </div>
        <div style={{ display: 'flex', gap: 8, justifyContent: 'center' }}>
          <button className="primary" onClick={confirm}>
            {isSwitch ? 'Confirmar' : `Confirmar (${picks.length})`}
          </button>
          {!isSwitch && (
            <button onClick={skip}>Não pegar nada</button>
          )}
        </div>
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
  const [config, setConfig] = useState<GameConfig>({ mytype: 'Fire', bottype: 'Water' })
  const [lobbyErr, setLobbyErr] = useState('')
  const [sel, setSel] = useState<Sel>(null)
  const [err, setErr] = useState('')
  const [pane, setPane] = useState<Pane>('')
  const [preview, setPreview] = useState<Preview | null>(null)
  const publishPreview = useCallback((c: CardView | null, rect?: DOMRect) =>
    setPreview(c && rect ? { card: c, rect } : null), [])

  useEffect(() => { fetchState().then(setS).catch(() => {}) }, [])

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
    }).catch(e => setLobbyErr(String(e)))
  }, [])

  const post = useCallback((body: Record<string, unknown>) => {
    postAction(body).then(j => {
      setErr(j.error ?? '')
      // Guarda contra panic do servidor: resposta sem `phase` não atualiza o estado
      // (mantém o tabuleiro visível em vez de crashar no render).
      if (j.phase) setS(j)
      setSel(null)
    }).catch(e => setErr(String(e)))
  }, [])

  const select = useCallback((kind: 'hand' | 'active' | 'bench', idx: number) => {
    // Modo pending: clique no slot completa a ação sem prompt().
    if (sel?.kind === 'pending') {
      if (kind === 'active' || kind === 'bench') {
        post({ action: sel.action, hand: sel.handIdx, slot: kind === 'active' ? -1 : idx })
        setSel(null)
      } else {
        setSel({ kind: 'hand', idx })
      }
      return
    }
    // Modo retreating passo 1: escolher banco.
    if (sel?.kind === 'retreating' && sel.benchIdx === null) {
      if (kind === 'bench') setSel({ kind: 'retreating', benchIdx: idx, energyIdxs: [] })
      else setSel(kind === 'hand' ? { kind: 'hand', idx } : null)
      return
    }
    // Modo retreating passo 2: seleção de energias — cliques no tapete ignorados.
    if (sel?.kind === 'retreating') return
    // Seleção normal: toggle.
    setSel(cur => {
      if (!cur || cur.kind !== kind) return { kind, idx }
      if ((cur as { kind: string; idx: number }).idx === idx) return null
      return { kind, idx }
    })
  }, [sel, post])

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
            menu={<AttackMenu s={s} sel={sel} setSel={setSel} post={post} />}
            pickMode={
              sel?.kind === 'pending' ? 'all'
              : (sel?.kind === 'retreating' && sel.benchIdx === null) ? 'bench'
              : null
            }
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
          <div id="hand-zone">
            <ActionBar s={s} sel={sel} setSel={setSel} err={err} post={post} />
            <HandTray s={s} sel={sel} onSelect={i => select('hand', i)} />
          </div>
        </div>
        <RightRail pane={pane} setPane={setPane} endTurn={() => post({ action: 'end_turn' })} />
        <Drawer pane={pane} s={s} post={post} />
        <CardPreview p={preview} />
        {s.pendingChoice && <ChoiceOverlay key={s.log?.length} pc={s.pendingChoice} post={post} />}
        <WinnerOverlay
          winner={s.winner}
          onReplay={() => startGame(config)}
          onNew={() => setS(null)}
        />
      </div>
    </PreviewCtx.Provider>
  )
}
