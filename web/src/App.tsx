import { useCallback, useEffect, useLayoutEffect, useRef, useState, type CSSProperties, type RefObject } from 'react'
import { fetchDecks, fetchState, postAction, postNew, type CardView, type DeckInfo, type GameConfig, type GameState, type Sel } from './api'
import { energyColor, energyDotStyle, energyImage, hiresImage } from './energy'
import { cancelFlights, flyFromDeck } from './drawfx'
import { playEvents } from './effectsfx'
import { CardPreview, PreviewCtx, type Preview } from './preview'
import { Sidebar } from './components/Sidebar'
import { BotMat, YouMat } from './components/Mat'
import { AttackMenu, ContextBar, PHASE_LABEL, TurnTimer } from './components/ActionBar'
import { Card } from './components/Card'
import { Drawer, type Pane } from './components/Drawer'

function netErr(e: unknown): string {
  const msg = String(e)
  if (msg.includes('Failed to fetch') || msg.includes('NetworkError') || msg.includes('ERR_CONNECTION')) {
    return 'Servidor indisponível. Verifique se o servidor está rodando.'
  }
  return msg
}

function useFocusTrap(ref: RefObject<HTMLElement | null>, active = true) {
  useEffect(() => {
    if (!active) return
    const el = ref.current
    if (!el) return
    const focusable = Array.from(el.querySelectorAll<HTMLElement>(
      'button:not([disabled]), [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    ))
    focusable[0]?.focus()
    const trap = (e: KeyboardEvent) => {
      if (e.key !== 'Tab') return
      const first = focusable[0]
      const last = focusable[focusable.length - 1]
      if (e.shiftKey) {
        if (document.activeElement === first) { e.preventDefault(); last?.focus() }
      } else {
        if (document.activeElement === last) { e.preventDefault(); first?.focus() }
      }
    }
    el.addEventListener('keydown', trap)
    return () => el.removeEventListener('keydown', trap)
  }, [ref, active])
}

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

// Painel lateral direito: status da vez + ferramentas + Terminar Turno.
// Substitui a antiga barra horizontal entre tabuleiro e mão.
function HudRail({ s, pane, setPane, endTurn }: {
  s: GameState
  pane: Pane
  setPane: (p: Pane) => void
  endTurn: () => void
}) {
  const myTurn = s.current === 0
  const toggle = (p: Pane) => setPane(pane === p ? '' : p)
  const tool = (p: Pane, label: string) => (
    <button type="button" className={'tool' + (pane === p ? ' on' : '')}
      aria-pressed={pane === p} onClick={() => toggle(p)}>
      {label}
    </button>
  )
  return (
    <aside id="right">
      <div className={'hud-status ' + (myTurn ? 'you-turn' : 'bot-turn')}>
        <span className={'vez ' + (myTurn ? 'you' : 'bot')}>{myTurn ? 'SUA VEZ' : 'VEZ DO BOT'}</span>
        <TurnTimer turn={s.turn} current={s.current} />
        <div className="hud-turnline">Turno {s.turn} · {PHASE_LABEL[s.phase] ?? s.phase}</div>
      </div>
      <div className="toolstack">
        {tool('cfg', 'Partida')}
        {tool('log', 'Log')}
        {tool('arb', 'Arbitrar')}
      </div>
      <div className="spacer" />
      <button id="endturn" onClick={endTurn} disabled={!myTurn || s.phase !== 'turn'}>
        TERMINAR<br />TURNO
      </button>
    </aside>
  )
}

// Toasts temporários: erros de ação e eventos novos do log (efeitos, nocautes).
// Histórico completo continua no painel de Log.
function Toasts({ s, err, errN }: { s: GameState; err: string; errN: number }) {
  const [toasts, setToasts] = useState<{ id: number; text: string; kind: 'err' | 'info' }[]>([])
  const nextId = useRef(1)
  const prevLog = useRef<number | null>(null)

  const push = useCallback((text: string, kind: 'err' | 'info') => {
    const id = nextId.current++
    setToasts(t => [...t.slice(-3), { id, text, kind }])
    setTimeout(() => setToasts(t => t.filter(x => x.id !== id)), 4000)
  }, [])

  useEffect(() => { if (err) push(err, 'err') }, [err, errN, push])

  const logLen = s.log?.length ?? 0
  useEffect(() => {
    const p = prevLog.current
    prevLog.current = logLen
    // p === null: primeiro render (não toastar histórico); salto > 5: partida nova/setup.
    if (p === null || logLen <= p || logLen - p > 5) return
    s.log!.slice(p).forEach(l => push(l, 'info'))
  }, [logLen, push]) // eslint-disable-line react-hooks/exhaustive-deps

  // Container sempre montado: região aria-live precisa existir antes da
  // mensagem para o leitor de tela anunciar.
  return (
    <div id="toasts" role="status" aria-live="polite">
      {toasts.map(t => <div key={t.id} className={'toast ' + t.kind}>{t.text}</div>)}
    </div>
  )
}

const TYPE_PT: Record<string, string> = {
  Grass: 'Planta', Fire: 'Fogo', Water: 'Água', Lightning: 'Elétrico',
  Psychic: 'Psíquico', Fighting: 'Lutador', Darkness: 'Escuridão',
  Metal: 'Metal', Dragon: 'Dragão', Colorless: 'Incolor',
}

const cardImg = (c: CardView) => c.image || (c.category === 'Energy' ? energyImage(c.nameEN) : '')

// Menu de decks de um lado: busca + lista rolável; deck selecionado ganha
// capa (carta-estrela), composição e botão para abrir o slide das 60 cartas.
function DeckMenu({ decks, sel, setSel, who, onView }: {
  decks: DeckInfo[]
  sel: string // id do deck selecionado
  setSel: (id: string) => void
  who: 'you' | 'bot'
  onView: (d: DeckInfo) => void
}) {
  const [q, setQ] = useState('')
  if (!decks.length) {
    return (
      <section className={`lobby-side deckshow ${who}`}>
        <h2>{who === 'you' ? 'Você' : 'Bot'}</h2>
        <div className="dk-none">Nenhum deck disponível. Importe sets com <code>task import</code>.</div>
      </section>
    )
  }
  const d = decks.find(dk => dk.id === sel) ?? decks[0]
  const col = energyColor(d.type)
  const cover = d.star.image ? hiresImage(d.star.image) : cardImg(d.star)
  const n = (cat: string) => d.counts[cat] ?? 0
  const norm = (s: string) => s.normalize('NFD').replace(/[̀-ͯ]/g, '').toLowerCase()
  const nq = norm(q.trim())
  const hits = decks.filter(dk =>
    !nq || norm(`${dk.name} ${dk.type} ${TYPE_PT[dk.type] ?? ''} ${dk.star.name}`).includes(nq))
  return (
    <section className={`lobby-side deckshow ${who}`} style={{ '--el': col } as CSSProperties}>
      <h2>{who === 'you' ? 'Você' : 'Bot'}</h2>
      <input type="search" className="dk-search" placeholder="Buscar deck ou tipo…"
        value={q} onChange={e => setQ(e.target.value)} aria-label="Buscar deck" />
      <div className="dk-menu" role="listbox" aria-label={`Decks — ${who === 'you' ? 'você' : 'bot'}`}>
        {hits.length === 0 && <div className="dk-none">Nenhum deck para “{q}”</div>}
        {hits.map(dk => (
          <button key={dk.id} type="button" role="option" aria-selected={dk.id === d.id}
            className={'dk-row' + (dk.id === d.id ? ' on' : '')}
            style={{ '--el': energyColor(dk.type) } as CSSProperties}
            onClick={() => setSel(dk.id)}>
            <img src={cardImg(dk.star)} alt="" loading="lazy" />
            <span className="dk-row-name">{dk.name}</span>
            <span className="dk-row-type">
              <i className="edot" style={energyDotStyle(dk.type)} />
              {TYPE_PT[dk.type] ?? dk.type}
            </span>
          </button>
        ))}
      </div>
      <div className="dk-sel">
        {/* key: trocar de deck remonta a imagem → animação de entrada */}
        <img key={d.id} className="coverimg mini" src={cover} alt={d.name} />
        <div className="dk-sel-info">
          <div className="deckname">{d.name}</div>
          <div className="deckmeta">
            {n('Pokemon')} Pokémon · {n('Trainer')} Treinadores · {n('Energy')} Energias
          </div>
          <button type="button" className="dk-view" onClick={() => onView(d)}>Ver as 60 cartas</button>
        </div>
      </div>
    </section>
  )
}

// Slide horizontal com todas as cartas do deck (scroll-snap + setas).
function DeckViewer({ deck, onClose }: { deck: DeckInfo; onClose: () => void }) {
  const boxRef = useRef<HTMLDivElement>(null)
  useFocusTrap(boxRef)
  const stripRef = useRef<HTMLDivElement>(null)
  useEffect(() => {
    const h = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
      if (e.key === 'ArrowLeft') stripRef.current?.scrollBy({ left: -400, behavior: 'smooth' })
      if (e.key === 'ArrowRight') stripRef.current?.scrollBy({ left: 400, behavior: 'smooth' })
    }
    window.addEventListener('keydown', h)
    return () => window.removeEventListener('keydown', h)
  }, [onClose])
  const scroll = (dir: number) => stripRef.current?.scrollBy({ left: dir * 400, behavior: 'smooth' })
  // Clique na lista rola o slide até a carta correspondente (mesma ordem).
  const goTo = (i: number) => stripRef.current?.children[i]
    ?.scrollIntoView({ behavior: 'smooth', inline: 'center', block: 'nearest' })
  const groups: [string, string][] = [['Pokemon', 'Pokémon'], ['Trainer', 'Treinadores'], ['Energy', 'Energias']]
  return (
    <div className="deckviewer" onClick={onClose}>
      <div className="dv-box" role="dialog" aria-modal="true" aria-label={deck.name} ref={boxRef} onClick={e => e.stopPropagation()}>
        <header className="dv-head">
          <span className="edot" style={energyDotStyle(deck.type)} />
          <span className="dv-title">{deck.name}</span>
          <span className="dv-sub">{deck.cards.length} cartas distintas · 60 no total</span>
          <button type="button" className="dv-close" aria-label="Fechar" onClick={onClose}>×</button>
        </header>
        <div className="dv-wrap">
          <button type="button" className="dk-arrow" aria-label="Rolar para trás" onClick={() => scroll(-1)}>‹</button>
          <div className="dv-strip" ref={stripRef}>
            {deck.cards.map(e => (
              <figure className="dv-card" key={e.card.id}>
                <img src={cardImg(e.card)} alt={e.card.name} loading="lazy" />
                <span className="dv-count">×{e.count}</span>
                <figcaption>{e.card.name}</figcaption>
              </figure>
            ))}
          </div>
          <button type="button" className="dk-arrow" aria-label="Rolar para frente" onClick={() => scroll(1)}>›</button>
        </div>
        <div className="dv-list">
          {groups.map(([cat, label]) => {
            const rows = deck.cards
              .map((e, i) => ({ ...e, i }))
              .filter(e => e.card.category === cat)
            if (rows.length === 0) return null
            const total = rows.reduce((s, e) => s + e.count, 0)
            return (
              <section key={cat} className="dv-group">
                <h3>{label} <span>{total}</span></h3>
                <ul>
                  {rows.map(e => (
                    <li key={e.card.id}>
                      <button type="button" onClick={() => goTo(e.i)}>
                        <b>{e.count}×</b> {e.card.name}
                      </button>
                    </li>
                  ))}
                </ul>
              </section>
            )
          })}
        </div>
      </div>
    </div>
  )
}

function LobbyScreen({ onStart, err }: { onStart: (c: GameConfig) => void; err: string }) {
  const [decks, setDecks] = useState<DeckInfo[] | null>(null)
  const [loadErr, setLoadErr] = useState('')
  const [myId, setMyId] = useState('')
  const [botId, setBotId] = useState('')
  const [viewer, setViewer] = useState<DeckInfo | null>(null)

  useEffect(() => {
    fetchDecks().then(ds => {
      setDecks(ds)
      const at = (t: string) => ds.find(d => d.type === t)?.id ?? ds[0]?.id ?? ''
      setMyId(at('Fire'))
      setBotId(at('Water'))
    }).catch(e => setLoadErr(netErr(e)))
  }, [])

  return (
    <div id="lobby">
      <div className="lobby-box">
        <div className="lobby-head">
          <div className="lobby-title">Pokémon TCG</div>
          <div className="lobby-sub">Escolha o Battle Deck de cada lado</div>
        </div>
        {!decks && <div className="lobby-sub">{loadErr || 'Carregando decks…'}</div>}
        {decks && (
          <div className="lobby-sides">
            <DeckMenu decks={decks} sel={myId} setSel={setMyId} who="you" onView={setViewer} />
            <div className="lobby-vs" aria-hidden="true">vs</div>
            <DeckMenu decks={decks} sel={botId} setSel={setBotId} who="bot" onView={setViewer} />
          </div>
        )}
        {err && <div className="lobby-err">{err}</div>}
        <button type="button" className="primary lobby-start"
          disabled={!decks || !myId || !botId}
          onClick={() => decks && onStart({ mytype: myId, bottype: botId })}>
          Iniciar partida
        </button>
      </div>
      {viewer && <DeckViewer deck={viewer} onClose={() => setViewer(null)} />}
    </div>
  )
}

// Overlay de escolha pendente: busca no deck ou troca de Pokémon Ativo.
function ChoiceOverlay({ pc, post }: {
  pc: NonNullable<GameState['pendingChoice']>
  post: (body: Record<string, unknown>) => void
}) {
  const boxRef = useRef<HTMLDivElement>(null)
  useFocusTrap(boxRef)
  const [picks, setPicks] = useState<number[]>([])
  const toggle = (i: number) => setPicks(cur =>
    cur.includes(i) ? cur.filter(x => x !== i)
      : cur.length < pc.max ? [...cur, i] : cur)

  const isSwitch = pc.kind === 'switch_self' || pc.kind === 'switch_opp'
  const isDiscard = pc.kind === 'discard_hand'
  const title = pc.kind === 'switch_self'
    ? 'Escolha 1 Pokémon do Banco para trocar com o Ativo'
    : pc.kind === 'switch_opp'
      ? 'Escolha 1 Pokémon do Banco do oponente para virar Ativo'
      : isDiscard
        ? `Descarte ${pc.min} carta${pc.min > 1 ? 's' : ''} da mão`
        : `Busca no deck: escolha até ${pc.max} carta${pc.max > 1 ? 's' : ''} ${pc.dest === 'bench' ? 'para o Banco' : 'para a mão'}`

  const confirm = () => post({ action: 'resolve_choice', picks })
  const skip = () => post({ action: 'resolve_choice', picks: [] })
  const needMore = picks.length < (pc.min ?? 0)

  return (
    <div className="choice-overlay">
      <div className="winner-box choice-box" role="dialog" aria-modal="true" aria-label={title} ref={boxRef}>
        <div className="choice-title">{title}</div>
        <div className="choice-cards">
          {pc.candidates.length === 0
            ? <p className="dk-none">Nenhuma carta disponível.</p>
            : pc.candidates.map((c, i) => (
                <Card key={i} view={c} selected={picks.includes(i)} onClick={() => toggle(i)} />
              ))}
        </div>
        <div style={{ display: 'flex', gap: 8, justifyContent: 'center' }}>
          <button type="button" className="primary" onClick={confirm} disabled={needMore && pc.candidates.length > 0}>
            {isSwitch ? 'Confirmar' : `Confirmar (${picks.length})`}
          </button>
          {!isSwitch && !isDiscard && (
            <button type="button" onClick={skip}>Não pegar nada</button>
          )}
        </div>
      </div>
    </div>
  )
}

function WinnerOverlay({ winner, onReplay, onNew }: { winner: number; onReplay: () => void; onNew: () => void }) {
  const ref = useRef<HTMLDivElement>(null)
  useFocusTrap(ref, winner >= 0 || winner === -2)
  if (winner < 0 && winner !== -2) return null
  const txt = winner === -2 ? 'Sudden Death!' : winner === 0 ? 'Você venceu!' : 'Bot venceu.'
  return (
    <div id="winner" role="dialog" aria-modal="true" aria-label={txt}>
      <div className="winner-box" ref={ref}>
        <div className="winner-txt">{txt}</div>
        <div style={{ display:'flex', gap:8, justifyContent:'center' }}>
          <button type="button" className="primary" onClick={onReplay}>Repetir</button>
          <button type="button" onClick={onNew}>Nova partida</button>
        </div>
      </div>
    </div>
  )
}

export default function App() {
  const [s, setS] = useState<GameState | null>(null)
  const [config, setConfig] = useState<GameConfig>({ mytype: 'Fire', bottype: 'Water' })
  const [lobbyErr, setLobbyErr] = useState('')
  const [connectErr, setConnectErr] = useState('')
  const [sel, setSel] = useState<Sel>(null)
  const [err, setErr] = useState('')
  const [errN, setErrN] = useState(0) // nonce: mesmo erro repetido gera novo toast
  const [pane, setPane] = useState<Pane>('')
  const [preview, setPreview] = useState<Preview | null>(null)
  const publishPreview = useCallback((c: CardView | null, rect?: DOMRect) =>
    setPreview(c && rect ? { card: c, rect } : null), [])

  useEffect(() => {
    fetchState().then(setS).catch(e => {
      // 404 = sem partida em andamento → ir para o lobby normalmente
      if (String(e).includes('404') || String(e).includes('HTTP 404')) return
      setConnectErr(netErr(e))
    })
  }, [])

  // Escape fecha a gaveta lateral (log/arbitragem/config).
  useEffect(() => {
    if (!pane) return
    const h = (e: KeyboardEvent) => { if (e.key === 'Escape') setPane('') }
    window.addEventListener('keydown', h)
    return () => window.removeEventListener('keydown', h)
  }, [pane])

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
    }).catch(e => setLobbyErr(netErr(e)))
  }, [])

  const post = useCallback((body: Record<string, unknown>) => {
    postAction(body).then(j => {
      setErr(j.error ?? '')
      if (j.error) setErrN(n => n + 1)
      // Guarda contra panic do servidor: resposta sem `phase` não atualiza o estado
      // (mantém o tabuleiro visível em vez de crashar no render).
      if (j.phase) {
        setS(j)
        playEvents(j.events)
      }
      setSel(null)
    }).catch(e => setErr(netErr(e)))
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

  if (connectErr) {
    return (
      <div style={{ display:'flex', alignItems:'center', justifyContent:'center', height:'100vh', flexDirection:'column', gap:16 }}>
        <div className="lobby-err" role="alert">{connectErr}</div>
        <button type="button" className="primary" onClick={() => window.location.reload()}>Recarregar</button>
      </div>
    )
  }

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
            <ContextBar s={s} sel={sel} setSel={setSel} post={post} />
            <HandTray s={s} sel={sel} onSelect={i => select('hand', i)} />
          </div>
        </div>
        <HudRail s={s} pane={pane} setPane={setPane} endTurn={() => post({ action: 'end_turn' })} />
        <Toasts s={s} err={err} errN={errN} />
        <Drawer pane={pane} s={s} post={post} onExit={() => { setPane(''); setS(null) }} />
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
