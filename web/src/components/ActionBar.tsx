import { useEffect, useState, type ReactElement } from 'react'
import type { CardView, GameState } from '../api'
import type { Sel } from '../api'
import { energyDotStyle } from '../energy'

// Custo pagável com as energias ligadas? Tipados casam por prefixo do nome EN
// ("Fire Energy" paga Fire); Colorless aceita qualquer sobra.
// ponytail: heurística visual — energias especiais que valem 2+ não são
// interpretadas; o motor continua validando o ataque de verdade.
export function canPay(cost: string[] | null, energies: CardView[]): boolean {
  if (!cost?.length) return true
  const used = new Set<number>()
  for (const t of cost.filter(c => c !== 'Colorless')) {
    const i = energies.findIndex((e, idx) => !used.has(idx) && e.nameEN.startsWith(t))
    if (i < 0) return false
    used.add(i)
  }
  return energies.length - used.size >= cost.filter(c => c === 'Colorless').length
}

export function TurnTimer({ turn, current }: { turn: number; current: number }) {
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
export function EnergyCost({ cost }: { cost: string[] | null }) {
  if (!cost?.length) return null
  return (
    <span className="btn-cost">
      {cost.map((t, i) => (
        <span key={i} className="edot" style={energyDotStyle(t)} title={t} />
      ))}
    </span>
  )
}

// Rótulos das fases para o jogador — oculta strings internas do motor.
export const PHASE_LABEL: Record<string, string> = {
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
  const ability = act.card.ability
  return (
    <div className="movebox">
      {ability && !act.abilityUsed && (
        <button className="move ability" onClick={() => setSel({ kind: 'ability', slot: -1 })}>
          <span className="move-name">✦ {ability.name}</span>
          <span className="move-dmg">habilidade</span>
        </button>
      )}
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

const PENDING_LABELS: Record<string, string> = {
  attach_energy: 'energia',
  evolve: 'Pokémon para evoluir',
  attach_tool: 'Pokémon para a ferramenta',
}

type HandAction = { test: (c: CardView) => boolean; label: string; action: (idx: number, post: Post, setSel: SetSel) => void }

const HAND_ACTIONS: HandAction[] = [
  { test: c => c.category === 'Energy',                    label: 'Ligar Energia',    action: (i, _, ss) => ss({ kind: 'pending', action: 'attach_energy', handIdx: i }) },
  { test: c => c.category === 'Pokemon' && c.stage === 'Basic', label: 'Baixar no Banco', action: (i, p) => p({ action: 'place_bench', hand: i }) },
  { test: c => c.category === 'Pokemon' && c.stage !== 'Basic', label: 'Evoluir',       action: (i, _, ss) => ss({ kind: 'pending', action: 'evolve', handIdx: i }) },
  { test: c => c.trainerType === 'Item',                   label: 'Jogar Item',       action: (i, p) => p({ action: 'play_item', hand: i }) },
  { test: c => c.trainerType === 'Supporter',              label: 'Jogar Suporte',    action: (i, p) => p({ action: 'play_supporter', hand: i }) },
  { test: c => c.trainerType === 'Stadium',                label: 'Jogar Estádio',    action: (i, p) => p({ action: 'play_stadium', hand: i }) },
  { test: c => c.trainerType === 'Tool',                   label: 'Ligar Ferramenta', action: (i, _, ss) => ss({ kind: 'pending', action: 'attach_tool', handIdx: i }) },
]

export function ContextBar({ s, sel, setSel, post }: {
  s: GameState
  sel: Sel
  setSel: SetSel
  post: Post
}) {
  const btn = (label: string, fn: () => void, key?: string, cls?: string) =>
    <button key={key ?? label} className={cls} onClick={fn}>{label}</button>
  const hint = (text: string) => <span key="hint" className="pick-hint">{text}</span>
  const cancel = () => btn('Cancelar', () => setSel(null), 'cancel')

  const mode = s.phase === 'setup' ? 'setup'
    : s.needPromote?.[0] ? 'promote'
    : s.current !== 0 ? 'idle'
    : sel?.kind ?? 'idle'

  const handlers: Record<string, () => ReactElement[]> = {
    idle: () => [],

    setup: () => {
      const nodes = [<span key="msg">Setup: escolha seu Ativo e Banco (cartas da mão).</span>]
      if (sel?.kind === 'hand') nodes.push(
        s.you.active
          ? btn('Colocar no Banco',  () => post({ action: 'place_bench', hand: sel.idx }), 'bench', 'primary')
          : btn('Colocar como Ativo', () => post({ action: 'place_active', hand: sel.idx }), 'active', 'primary')
      )
      if (s.you.active) nodes.push(btn('Concluir setup', () => post({ action: 'finish_setup' })))
      return nodes
    },

    promote: () => {
      const nodes = [<span key="msg">Seu Ativo caiu — clique num Pokémon do banco e promova.</span>]
      if (sel?.kind === 'bench') nodes.push(btn('Promover', () => post({ action: 'promote', bench: sel.idx }), 'promote', 'primary'))
      return nodes
    },

    ability: () => [hint('↑ Clique no Pokémon alvo da Habilidade'), cancel()],

    pending: () => {
      const action = (sel as Extract<Sel, { kind: 'pending' }>)?.action ?? ''
      return [hint(`↑ Clique no ${PENDING_LABELS[action] ?? 'alvo'} no tabuleiro`), cancel()]
    },

    retreating: () => {
      const r = sel as Extract<Sel, { kind: 'retreating' }>
      if (r.benchIdx === null) return [hint('↑ Clique no Pokémon do Banco para onde recuar'), cancel()]
      const cost = s.you.active?.card.retreat ?? 0
      const energies = s.you.active?.energies ?? []
      const chosen = r.energyIdxs
      const toggle = (i: number) => {
        const next = chosen.includes(i) ? chosen.filter(x => x !== i) : [...chosen, i]
        setSel({ kind: 'retreating', benchIdx: r.benchIdx, energyIdxs: next })
      }
      const canConfirm = cost === 0 || chosen.length === cost
      return [
        <span key="retreat-label" className="retreat-prompt">
          {cost > 0
            ? <>Descartar {cost} energia{cost !== 1 ? 's' : ''}:{' '}
                {energies.map((e, i) => (
                  <span key={i}
                    className={'edot selectable' + (chosen.includes(i) ? ' chosen' : '')}
                    style={energyDotStyle(e.nameEN)} title={e.name}
                    role="button" tabIndex={0}
                    aria-pressed={chosen.includes(i)} aria-label={e.name}
                    onClick={() => toggle(i)}
                    onKeyDown={ev => { if (ev.key === 'Enter' || ev.key === ' ') { ev.preventDefault(); toggle(i) } }} />
                ))}
              </>
            : <span>Recuo gratuito</span>}
        </span>,
        <button key="confirm" className={canConfirm ? 'primary' : ''} disabled={!canConfirm}
          onClick={() => { post({ action: 'retreat', bench: r.benchIdx!, energies: chosen }); setSel(null) }}>
          Confirmar Recuo
        </button>,
        cancel(),
      ]
    },

    active: () => {
      const ab = s.you.active
      if (!ab?.card.ability || ab.abilityUsed) return []
      return [btn(`Habilidade: ${ab.card.ability.name}`, () => setSel({ kind: 'ability', slot: -1 }), 'ability')]
    },

    bench: () => {
      const idx = (sel as { kind: string; idx: number })?.idx
      const bk = s.you.bench?.[idx]
      if (!bk?.card.ability || bk.abilityUsed) return []
      return [btn(`Habilidade: ${bk.card.ability.name}`, () => setSel({ kind: 'ability', slot: idx }), 'ability')]
    },

    hand: () => {
      const idx = (sel as { kind: string; idx: number })?.idx
      const c = s.you.hand?.[idx]
      if (!c) return []
      return HAND_ACTIONS
        .filter(a => a.test(c))
        .map(a => btn(a.label, () => a.action(idx, post, setSel), a.label))
    },
  }

  const nodes = handlers[mode]?.() ?? []
  if (nodes.length === 0) return null
  return <div id="ctxbar">{nodes}</div>
}
