import { createContext, useEffect, useState } from 'react'
import type { CardView } from './api'
import { energyImage, hiresImage } from './energy'

// Hover em qualquer carta publica o CardView + rect da carta aqui; App
// renderiza o painel flutuante com a imagem em alta resolução (cartas
// pequenas no tabuleiro são ilegíveis sem isso).
export type Preview = { card: CardView; rect: DOMRect }
export const PreviewCtx = createContext<(c: CardView | null, rect?: DOMRect) => void>(() => {})

// Posiciona ao lado da carta de origem (direita; esquerda se não couber),
// centrado verticalmente nela e limitado à viewport — nunca cobre a origem.
// Ao fechar (p = null) mantém a última carta montada para o fade de saída.
export function CardPreview({ p }: { p: Preview | null }) {
  const [last, setLast] = useState(p)
  useEffect(() => { if (p) setLast(p) }, [p])
  const shown = p ?? last
  if (!shown) return null
  const { card, rect } = shown
  const img = card.image || (card.category === 'Energy' ? energyImage(card.nameEN) : '')
  const w = Math.min(340, window.innerWidth * 0.26)
  const h = w * 88 / 63
  let left = rect.right + 14
  if (left + w > window.innerWidth - 8) left = rect.left - 14 - w
  if (left < 8) left = 8
  const top = Math.min(Math.max(rect.top + rect.height / 2 - h / 2, 8), window.innerHeight - h - 8)
  return (
    <div id="preview" className={p ? 'show' : ''} style={{ left, top, width: w }}>
      {img
        ? <img src={hiresImage(img)} alt={card.name} />
        : <div className="card txt">{card.name}</div>}
    </div>
  )
}
