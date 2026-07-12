// Energias básicas do TCGdex vêm sem imagem (image:{} no set sve); a UI
// desenha a carta a partir do elemento deduzido do nome EN ("Grass Energy").
export interface EnergyStyle {
  color: string
  icon: string
}

const styles: Record<string, EnergyStyle> = {
  Grass:     { color: '#5fa338', icon: '🍃' },
  Fire:      { color: '#d6543f', icon: '🔥' },
  Water:     { color: '#3187c4', icon: '💧' },
  Lightning: { color: '#e3b62c', icon: '⚡' },
  Psychic:   { color: '#9a5aa8', icon: '👁' },
  Fighting:  { color: '#b06a3a', icon: '✊' },
  Darkness:  { color: '#31495c', icon: '🌙' },
  Metal:     { color: '#7e8c99', icon: '⚙' },
  Colorless: { color: '#a8a49c', icon: '✦' },
}

export function energyStyle(nameEN: string): EnergyStyle {
  const el = Object.keys(styles).find(k => nameEN.startsWith(k))
  return el ? styles[el] : styles.Colorless
}
