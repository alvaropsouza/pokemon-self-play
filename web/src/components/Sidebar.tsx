import { useEffect, useRef, useState } from 'react'
import type { SideView } from '../api'
import { PHASE_LABEL } from './ActionBar'

function MatchTimer() {
  const startRef = useRef(Date.now())
  const [sec, setSec] = useState(0)
  useEffect(() => {
    const id = setInterval(() => setSec(Math.floor((Date.now() - startRef.current) / 1000)), 1000)
    return () => clearInterval(id)
  }, [])
  const mm = String(Math.floor(sec / 60)).padStart(2, '0')
  const ss = String(sec % 60).padStart(2, '0')
  return <span className="timer">{mm}:{ss}</span>
}

function PrizeBalls({ count, variant }: { count: number; variant: 'bot' | 'you' }) {
  const prevCount = useRef(count)
  const [justTaken, setJustTaken] = useState<number | null>(null)

  useEffect(() => {
    if (count < prevCount.current) {
      setJustTaken(count)
      const id = setTimeout(() => setJustTaken(null), 450)
      prevCount.current = count
      return () => clearTimeout(id)
    }
    prevCount.current = count
  }, [count])

  const label = variant === 'you'
    ? `Seus prêmios: ${count} restantes`
    : `Prêmios do oponente: ${count} restantes`

  return (
    <div role="group" className={`prize-track prize-track--${variant}`} aria-label={label}>
      <div className="prize-header">
        <span className="prize-label">Prêmios</span>
        <span className={'prize-count' + (count <= 2 ? ' urgent' : '')} key={count}>
          {count}<span className="prize-total">/6</span>
        </span>
      </div>
      <div className="prize-dots">
        {Array.from({ length: 6 }, (_, i) => (
          <div
            key={i}
            className={'prize' + (i >= count ? ' taken' : '') + (i === justTaken ? ' just-taken' : '')}
            aria-hidden="true"
          />
        ))}
      </div>
    </div>
  )
}

export function Sidebar({ you, bot, current, turn, phase, botThinking }: {
  you: SideView; bot: SideView; current: number; turn: number; phase: string; botThinking: boolean
}) {
  const isBotActive = current === 1 || botThinking
  return (
    <aside id="left">
      <div className={'pp bot' + (isBotActive ? ' turn' : '')}>
        <div className="avatar">B</div>
        <div className="who">OPONENTE</div>
        {botThinking
          ? <div className="bot-thinking" aria-label="Bot jogando"><span /><span /><span /></div>
          : <div className="sub">Bot</div>
        }
        <PrizeBalls count={bot.prizes} variant="bot" />
      </div>
      <div className={'pp you' + (!isBotActive ? ' turn' : '')}>
        <div className="avatar">T</div>
        <div className="who">VOCÊ</div>
        <div className="sub">Treinador</div>
        <PrizeBalls count={you.prizes} variant="you" />
      </div>
      <div className={'hud-status ' + (isBotActive ? 'bot-turn' : 'you-turn')}>
        <span className={'vez ' + (isBotActive ? 'bot' : 'you')}>
          {botThinking ? 'BOT JOGANDO' : isBotActive ? 'VEZ DO BOT' : 'SUA VEZ'}
        </span>
        <MatchTimer />
        <div className="hud-turnline">Turno {turn} · {PHASE_LABEL[phase] ?? phase}</div>
      </div>
    </aside>
  )
}
