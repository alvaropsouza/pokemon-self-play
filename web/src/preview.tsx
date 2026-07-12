import { createContext } from 'react'
import type { CardView } from './api'
import { energyImage, hiresImage } from './energy'

// Hover em qualquer carta publica o CardView aqui; App renderiza o painel
// flutuante com a imagem em alta resolução (cartas pequenas no tabuleiro
// são ilegíveis sem isso).
export const PreviewCtx = createContext<(c: CardView | null) => void>(() => {})

export function CardPreview({ card }: { card: CardView | null }) {
  if (!card) return null
  const img = card.image || (card.category === 'Energy' ? energyImage(card.nameEN) : '')
  return (
    <div id="preview">
      {img
        ? <img src={hiresImage(img)} alt={card.name} />
        : <div className="card txt">{card.name}</div>}
    </div>
  )
}
