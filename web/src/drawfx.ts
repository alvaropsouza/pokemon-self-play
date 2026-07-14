// Animação de compra: um clone da carta (position:fixed em document.body, fora
// do overflow do handtray) voa do baralho até a posição final na mão via WAAPI.
// Só transform/filter/opacity — nada de reflow. Cancelável: compra em sequência
// chama cancelFlights() e cada voo pendente é abortado e limpo.

let flights: (() => void)[] = []

export function cancelFlights() {
  flights.forEach(stop => stop())
  flights = []
}

// Efeito sonoro opcional: web/public/draw.mp3. Sem o arquivo, falha em silêncio.
let sfx: HTMLAudioElement | null = null
function playDrawSound() {
  sfx ??= new Audio('/draw.mp3')
  sfx.currentTime = 0
  void sfx.play().catch(() => {})
}

// cardEl: a carta real já renderizada na mão (fica invisível durante o voo).
// to: rect da posição final (medido antes de qualquer animação começar).
// finalTilt: ângulo do leque no destino. delay: escalona compras múltiplas.
export function flyFromDeck(cardEl: HTMLElement, to: DOMRect, finalTilt = 0, delay = 0) {
  // primeiro .pile do tapete do jogador = baralho (descarte com carta não usa .pile)
  const deck = document.querySelector('.mat.you .pile')
  const face = cardEl.querySelector('.card')
  if (!deck || !face) return
  const from = deck.getBoundingClientRect()
  playDrawSound()

  const wrap = document.createElement('div')
  wrap.className = 'flycard'
  Object.assign(wrap.style, {
    left: `${to.left}px`, top: `${to.top}px`,
    width: `${to.width}px`, height: `${to.height}px`,
  })
  const img = face.cloneNode(true) as HTMLElement
  img.style.width = '100%'
  const glow = document.createElement('div')
  glow.className = 'flyglow'
  wrap.append(img, glow)
  document.body.append(wrap)
  cardEl.style.visibility = 'hidden'

  const done = () => { wrap.remove(); cardEl.style.visibility = '' }

  const dx = from.left + from.width / 2 - (to.left + to.width / 2)
  const dy = from.top + from.height / 2 - (to.top + to.height / 2)
  const s0 = from.width / to.width
  // arco: desvio perpendicular ao vetor baralho→mão no meio do trajeto
  const len = Math.hypot(dx, dy) || 1
  const arcX = (-dy / len) * 40
  const arcY = (dx / len) * 40

  const opts: KeyframeAnimationOptions = { duration: 550, delay, fill: 'backwards' }
  const fly = wrap.animate([
    { offset: 0, easing: 'cubic-bezier(0.4,0,0.2,1)',
      transform: `translate(${dx}px, ${dy}px) scale(${s0})`,
      filter: 'blur(0px) drop-shadow(0 2px 4px rgba(0,0,0,.4))' },
    // descola do topo do baralho: sobe 8px com inclinação
    { offset: 0.15, easing: 'cubic-bezier(0.4,0,0.2,1)',
      transform: `translate(${dx}px, ${dy - 8}px) rotate(-6deg) scale(${s0})`,
      filter: 'blur(0px) drop-shadow(0 6px 12px rgba(0,0,0,.45))' },
    // meio do arco: mais rápido, blur leve, sombra alta
    { offset: 0.6, easing: 'cubic-bezier(0.34,1.56,0.64,1)', // easeOutBack na chegada
      transform: `translate(${dx * 0.42 + arcX}px, ${dy * 0.42 + arcY}px) rotate(5deg) scale(0.94)`,
      filter: 'blur(2px) drop-shadow(0 14px 28px rgba(0,0,0,.55))' },
    // chegada: overshoot de escala…
    { offset: 0.85, easing: 'ease-out',
      transform: `rotate(${finalTilt}deg) scale(1.05)`,
      filter: 'blur(0px) drop-shadow(0 4px 10px rgba(0,0,0,.4))' },
    // …e assenta
    { transform: `rotate(${finalTilt}deg) scale(1)`,
      filter: 'blur(0px) drop-shadow(0 2px 6px rgba(0,0,0,.35))' },
  ], opts)
  // brilho varre a carta durante o deslocamento
  glow.animate([
    { transform: 'translateX(-160%) rotate(18deg)', opacity: 0, offset: 0 },
    { opacity: 0.9, offset: 0.55, easing: 'ease-out' },
    { transform: 'translateX(320%) rotate(18deg)', opacity: 0 },
  ], opts)

  const stop = () => { fly.cancel(); done() }
  flights.push(stop)
  fly.onfinish = () => { done(); flights = flights.filter(f => f !== stop) }
}
