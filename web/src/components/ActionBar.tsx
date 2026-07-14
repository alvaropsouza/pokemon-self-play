import { useEffect, useState } from 'react'
import type { CardView, GameState } from '../api'
import type { Sel } from '../api'
import { energyColor } from '../energy'

// Custo pagável com as energias ligadas? Tipados casam por prefixo do nome EN
// ("Fire Energy" paga Fire); Colorless aceita qualquer sobra.
// ponytail: heurística visual — energias especiais que valem 2+ não são
// interpretadas; o motor continua validando o ataque de verdade.
function canPay(cost: string[] | null, energies: CardView[]): boolean {
  if (!cost?.length) return true
  const used = new Set<number>()
  for (const t of cost.filter(c => c !== 'Colorless')) {
    const i = energies.findIndex((e, idx) => !used.has(idx) && e.nameEN.startsWith(t))
    if (i < 0) return false
    used.add(i)
  }
  return energies.length - used.size >= cost.filter(c => c === 'Colorless').length
}

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

// Custo de energia como bolinhas coloridas dentro do botão de ataque.
function EnergyCost({ cost }: { cost: string[] | null }) {
  if (!cost?.length) return null
  return (
    <span className="btn-cost">
      {cost.map((t, i) => (
        <span key={i} className="edot" style={{ background: energyColor(t) }} title={t} />
      ))}
    </span>
  )
}

// Rótulos das fases para o jogador — oculta strings internas do motor.
const PHASE_LABEL: Record<string, string> = {
  setup: 'Preparação',
  turn: 'Em jogo',
  finished: 'Encerrada',
}

type Post = (body: Record<string, unknown>) => void
type SetSel = React.Dispatch<React.SetStateAction<Sel>>

// Menu de golpes estilo Game Boy, renderizado abaixo do HUD do jogador (Mat).
// Ataques + Recuar saem da barra inferior e vivem junto do Pokémon Ativo.
export function AttackMenu({ s, sel, setSel, post }: {
  s: GameState
  sel: Sel
  setSel: SetSel
  post: Post
}) {
  const act = s.you.active
  if (!act || s.phase !== 'turn' || s.current !== 0 || s.winner >= 0) return null
  if (s.needPromote?.[0]) return null
  if (sel?.kind === 'pending' || sel?.kind === 'retreating') return null

  const firstTurn = s.turn === 1
  return (
    <div className="movebox">
      {firstTurn
        ? <span className="move-hint">Sem ataque no 1º turno</span>
        : act.card.attacks?.map((a, i) => {
            const payable = canPay(a.cost, act.energies ?? [])
            return (
              <button key={i} className={'move' + (payable ? '' : ' off')}
                title={payable ? undefined : 'Energia insuficiente'}
                onClick={() => post({ action: 'attack', attack: i })}>
                <EnergyCost cost={a.cost} />
                <span className="move-name">{a.name}</span>
                <span className="move-dmg">{a.damage || 'efeito'}</span>
              </button>
            )
          })}
      {s.you.bench?.length ? (
        <button className="move retreat" onClick={() => setSel({ kind: 'retreating', benchIdx: null, energyIdxs: [] })}>
          <span className="move-name">Recuar</span>
          <span className="move-dmg">{act.card.retreat || 'grátis'}</span>
        </button>
      ) : null}
    </div>
  )
}

export function ActionBar({ s, sel, setSel, err, post }: {
  s: GameState
  sel: Sel
  setSel: SetSel
  err: string
  post: Post
}) {
  const myTurn = s.current === 0
  const phaseTxt = PHASE_LABEL[s.phase] ?? s.phase

  const actions: React.ReactNode[] = []

  const util = (label: string, fn: () => void, key?: string) => (
    <button key={key ?? label} onClick={fn}>{label}</button>
  )
  const critical = (label: string, fn: () => void, key?: string) => (
    <button key={key ?? label} className="primary" onClick={fn}>{label}</button>
  )

  if (s.phase === 'setup') {
    actions.push(<span key="msg">Setup: escolha seu Ativo e Banco (cartas da mão).</span>)
    if (sel?.kind === 'hand') {
      if (!s.you.active) actions.push(critical('Colocar como Ativo', () => post({ action: 'place_active', hand: sel.idx })))
      else actions.push(critical('Colocar no Banco', () => post({ action: 'place_bench', hand: sel.idx })))
    }
    if (s.you.active) actions.push(util('Concluir setup', () => post({ action: 'finish_setup' })))

  } else if (s.needPromote?.[0]) {
    actions.push(<span key="msg">Seu Ativo caiu — clique num Pokémon do banco e promova.</span>)
    if (sel?.kind === 'bench') actions.push(critical('Promover', () => post({ action: 'promote', bench: sel.idx })))

  } else if (s.current !== 0) {
    actions.push(<span key="msg" style={{ color: 'var(--dim)' }}>Turno do bot…</span>)

  } else if (sel?.kind === 'pending') {
    // Modo pick: aguardando clique num slot do tabuleiro.
    const labels: Record<string, string> = {
      attach_energy: 'energia',
      evolve: 'Pokémon para evoluir',
      attach_tool: 'Pokémon para a ferramenta',
    }
    actions.push(
      <span key="hint" className="pick-hint">↑ Clique no {labels[sel.action] ?? 'alvo'} no tabuleiro</span>,
      util('Cancelar', () => setSel(null), 'cancel'),
    )

  } else if (sel?.kind === 'retreating') {
    if (sel.benchIdx === null) {
      // Passo 1: escolher slot do banco.
      actions.push(
        <span key="hint" className="pick-hint">↑ Clique no Pokémon do Banco para onde recuar</span>,
        util('Cancelar', () => setSel(null), 'cancel'),
      )
    } else {
      // Passo 2: selecionar energias a descartar.
      const cost = s.you.active?.card.retreat ?? 0
      const energies = s.you.active?.energies ?? []
      const chosen = sel.energyIdxs

      const toggleEnergy = (i: number) => {
        const next = chosen.includes(i) ? chosen.filter(x => x !== i) : [...chosen, i]
        setSel({ kind: 'retreating', benchIdx: sel.benchIdx, energyIdxs: next })
      }

      const canConfirm = cost === 0 || chosen.length === cost

      actions.push(
        <span key="retreat-label" className="retreat-prompt">
          {cost > 0
            ? <>Descartar {cost} energia{cost !== 1 ? 's' : ''}:{' '}
                {energies.map((e, i) => (
                  <span key={i}
                    className={'edot selectable' + (chosen.includes(i) ? ' chosen' : '')}
                    style={{ background: energyColor(e.nameEN) }}
                    title={e.name}
                    onClick={() => toggleEnergy(i)} />
                ))}
              </>
            : <span>Recuo gratuito</span>
          }
        </span>,
        <button key="confirm" className={canConfirm ? 'primary' : ''} disabled={!canConfirm}
          onClick={() => { post({ action: 'retreat', bench: sel.benchIdx!, energies: chosen }); setSel(null) }}>
          Confirmar Recuo
        </button>,
        util('Cancelar', () => setSel(null), 'cancel'),
      )
    }

  } else {
    // Turno normal.
    if (sel?.kind === 'hand') {
      const c = s.you.hand![sel.idx]
      if (c.category === 'Energy') {
        actions.push(util('Ligar Energia', () => setSel({ kind: 'pending', action: 'attach_energy', handIdx: sel.idx })))
      }
      if (c.category === 'Pokemon' && c.stage === 'Basic') {
        actions.push(util('Baixar no Banco', () => post({ action: 'place_bench', hand: sel.idx })))
      }
      if (c.category === 'Pokemon' && c.stage !== 'Basic') {
        actions.push(util('Evoluir', () => setSel({ kind: 'pending', action: 'evolve', handIdx: sel.idx })))
      }
      if (c.trainerType === 'Item') actions.push(util('Jogar Item', () => post({ action: 'play_item', hand: sel.idx })))
      if (c.trainerType === 'Supporter') actions.push(util('Jogar Suporte', () => post({ action: 'play_supporter', hand: sel.idx })))
      if (c.trainerType === 'Stadium') actions.push(util('Jogar Estádio', () => post({ action: 'play_stadium', hand: sel.idx })))
      if (c.trainerType === 'Tool') {
        actions.push(util('Ligar Ferramenta', () => setSel({ kind: 'pending', action: 'attach_tool', handIdx: sel.idx })))
      }
    }

    // Ataques e Recuar vivem no AttackMenu, abaixo do HUD do Ativo (Mat).
  }

  return (
    <>
      <div id="actionbar">
        <span id="status">
          <span className={'vez ' + (myTurn ? 'you' : 'bot')}>{myTurn ? 'SUA VEZ' : 'VEZ DO BOT'}</span>
          <span>Turno {s.turn} · {phaseTxt}</span>
          <TurnTimer turn={s.turn} current={s.current} />
        </span>
        <span className="bar-div" aria-hidden="true" />
        <span id="actions">{actions}</span>
      </div>
      {err && <div className="err-banner">{err}</div>}
    </>
  )
}
