import { createContext } from 'react'
import type { CardView } from './api'
import { energyStyle } from './energy'

// Hover em qualquer carta publica o CardView aqui; App renderiza o painel
// flutuante com a imagem em alta resolução (cartas pequenas no tabuleiro
// são ilegíveis sem isso).
export const PreviewCtx = createContext<(c: CardView | null) => void>(() => {})

export function CardPreview({ card }: { card: CardView | null }) {
  if (!card) return null
  return (
    <div id="preview">
      {card.image
        ? <img src={card.image.replace('/low.webp', '/high.webp')} alt={card.name} />
        : <BigEnergy nameEN={card.nameEN} name={card.name} />}
    </div>
  )
}

function BigEnergy({ nameEN, name }: { nameEN: string; name: string }) {
  const s = energyStyle(nameEN)
  return (
    <div className="energycard big" style={{ background: s.color }}>
      <span className="etype">ENERGIA</span>
      <span className="eicon">{s.icon}</span>
      <span className="ename">{name}</span>
    </div>
  )
}
