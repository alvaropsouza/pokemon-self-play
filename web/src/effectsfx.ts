// Animações de efeitos estruturados (GameState.events): coisas que o diff de
// estado não captura, como embaralhar (contagens não mudam). Dano/cura/condição
// são animados por diff nos próprios componentes (Card.tsx).

import type { GameEvent } from './api'

export function playEvents(events: GameEvent[] | undefined) {
  if (!events?.length || matchMedia('(prefers-reduced-motion: reduce)').matches) return
  const shaken = new Set<number>()
  requestAnimationFrame(() => {
    for (const ev of events) {
      if ((ev.kind === 'shuffle_deck' || ev.kind === 'shuffle_hand') && !shaken.has(ev.player)) {
        shaken.add(ev.player)
        shakeDeck(ev.player)
      }
    }
  })
}

function shakeDeck(player: number) {
  const pile = document.querySelector(`.mat.${player === 0 ? 'you' : 'bot'} .pile`)
  pile?.animate([
    { transform: 'translateX(0) rotate(0deg)' },
    { transform: 'translateX(-4px) rotate(-3deg)' },
    { transform: 'translateX(4px) rotate(3deg)' },
    { transform: 'translateX(-3px) rotate(-2deg)' },
    { transform: 'translateX(3px) rotate(2deg)' },
    { transform: 'translateX(0) rotate(0deg)' },
  ], { duration: 450, easing: 'ease-in-out' })
}
