import { useEffect, useState } from 'react'
import type { GameState } from '../api'
import type { Sel } from '../api'
import { energyColor } from '../energy'

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

  // Botão de ação: ataque = primary, utilitário = secondary.
  const attack = (label: React.ReactNode, fn: () => void, key: string) => (
    <button key={key} className="primary" onClick={fn}>{label}</button>
  )
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

    if (s.you.active) {
      // Turno 1: quem começa não pode atacar (regra CLAUDE.md §6).
      const firstTurn = s.turn === 1 && myTurn
      if (!firstTurn) {
        s.you.active.card.attacks?.forEach((a, i) => {
          actions.push(
            attack(
              <><EnergyCost cost={a.cost} />{a.name} ({a.damage || 'efeito'})</>,
              () => post({ action: 'attack', attack: i }),
              `attack-${i}`,
            )
          )
        })
      } else {
        actions.push(<span key="no-atk" style={{ color: 'var(--dim)', fontSize: 11 }}>Sem ataque no 1º turno</span>)
      }
      if (s.you.bench?.length) {
        actions.push(util('Recuar', () => setSel({ kind: 'retreating', benchIdx: null, energyIdxs: [] })))
      }
    }
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
