import { useEffect, useState } from 'react'
import type { GameState } from '../api'
import type { Sel } from '../selection'

// Cronômetro do turno atual (client-side; o motor não tem relógio).
function TurnTimer({ turn, current }: { turn: number; current: number }) {
  const [sec, setSec] = useState(0)
  useEffect(() => {
    setSec(0)
    const id = setInterval(() => setSec(v => v + 1), 1000)
    return () => clearInterval(id)
  }, [turn, current])
  const mm = String(Math.floor(sec / 60)).padStart(2, '0')
  const ss = String(sec % 60).padStart(2, '0')
  return <span className="timer">{mm}:{ss}</span>
}

type Post = (body: Record<string, unknown>) => void

// Alvo em jogo escolhido (ou pergunta): -1 ativo, 0.. banco.
function chosenSlot(sel: Sel): number | null {
  if (sel?.kind === 'active') return -1
  if (sel?.kind === 'bench') return sel.idx
  const v = prompt('Alvo: -1 = Ativo, 0.. = posição do banco', '-1')
  return v === null ? null : parseInt(v)
}

function retreat(s: GameState, post: Post) {
  const b = prompt('Trocar com qual posição do banco? (0..)', '0')
  if (b === null) return
  const cost = s.you.active?.card.retreat ?? 0
  let en: number[] = []
  if (cost > 0) {
    const e = prompt(`Custo ${cost}: índices das energias a descartar (ex.: 0,1)`,
      Array.from({ length: cost }, (_, i) => i).join(','))
    if (e === null) return
    en = e.split(',').filter(x => x !== '').map(Number)
  }
  post({ action: 'retreat', bench: parseInt(b), energies: en })
}

// Botões contextuais: dependem da fase, da vez e da carta/slot selecionado.
export function ActionBar({ s, sel, err, post }: {
  s: GameState
  sel: Sel
  err: string
  post: Post
}) {
  const myTurn = s.current === 0

  const actions: React.ReactNode[] = []
  const btn = (label: string, fn: () => void, primary = false) => (
    <button key={label} className={primary ? 'primary' : ''} onClick={fn}>{label}</button>
  )

  if (s.phase === 'setup') {
    actions.push(<span key="msg">Setup: escolha seu Ativo e Banco (cartas da mão).</span>)
    if (sel?.kind === 'hand') {
      if (!s.you.active) actions.push(btn('Colocar como Ativo', () => post({ action: 'place_active', hand: sel.idx }), true))
      else actions.push(btn('Colocar no Banco', () => post({ action: 'place_bench', hand: sel.idx }), true))
    }
    if (s.you.active) actions.push(btn('Concluir setup', () => post({ action: 'finish_setup' })))
  } else if (s.needPromote?.[0]) {
    actions.push(<span key="msg">Seu Ativo caiu — clique num Pokémon do banco e promova.</span>)
    if (sel?.kind === 'bench') actions.push(btn('Promover', () => post({ action: 'promote', bench: sel.idx }), true))
  } else if (s.current !== 0) {
    actions.push(<span key="msg">Turno do bot…</span>)
  } else {
    if (sel?.kind === 'hand') {
      const c = s.you.hand![sel.idx]
      if (c.category === 'Energy') actions.push(btn('Ligar Energia', () => {
        const s2 = chosenSlot(sel); if (s2 !== null) post({ action: 'attach_energy', hand: sel.idx, slot: s2 })
      }, true))
      if (c.category === 'Pokemon' && c.stage === 'Basic') actions.push(btn('Baixar no Banco', () => post({ action: 'place_bench', hand: sel.idx }), true))
      if (c.category === 'Pokemon' && c.stage !== 'Basic') actions.push(btn('Evoluir', () => {
        const s2 = chosenSlot(sel); if (s2 !== null) post({ action: 'evolve', hand: sel.idx, slot: s2 })
      }, true))
      if (c.trainerType === 'Item') actions.push(btn('Jogar Item', () => post({ action: 'play_item', hand: sel.idx }), true))
      if (c.trainerType === 'Supporter') actions.push(btn('Jogar Suporte', () => post({ action: 'play_supporter', hand: sel.idx }), true))
      if (c.trainerType === 'Stadium') actions.push(btn('Jogar Estádio', () => post({ action: 'play_stadium', hand: sel.idx }), true))
      if (c.trainerType === 'Tool') actions.push(btn('Ligar Ferramenta', () => {
        const s2 = chosenSlot(sel); if (s2 !== null) post({ action: 'attach_tool', hand: sel.idx, slot: s2 })
      }, true))
    }
    if (s.you.active) {
      s.you.active.card.attacks?.forEach((a, i) => {
        actions.push(btn(`${a.name} (${a.damage || 'efeito'})`, () => post({ action: 'attack', attack: i }), true))
      })
      if (s.you.bench?.length) actions.push(btn('Recuar', () => retreat(s, post)))
    }
  }

  return (
    <div id="actionbar" className={err ? 'err' : ''}>
      <span id="status">
        <span className={'vez ' + (myTurn ? 'you' : 'bot')}>{myTurn ? 'SUA VEZ' : 'VEZ DO BOT'}</span>
        <span>Turno {s.turn} · {s.phase}</span>
        <TurnTimer turn={s.turn} current={s.current} />
        {err && <span className="err">{err}</span>}
      </span>
      <span id="actions">{actions}</span>
    </div>
  )
}
