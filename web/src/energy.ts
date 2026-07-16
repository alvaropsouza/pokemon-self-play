
const sveNum: Record<string, number> = {
  Grass: 1, Fire: 2, Water: 3, Lightning: 4,
  Psychic: 5, Fighting: 6, Darkness: 7, Metal: 8,
}

export function energyImage(nameEN: string): string {
  const el = Object.keys(sveNum).find(k => nameEN.startsWith(k))
  return el ? `https://images.pokemontcg.io/sve/${sveNum[el]}.png` : ''
}

export function hiresImage(url: string): string {
  return url.includes('tcgdex')
    ? url.replace('/low.webp', '/high.webp')
    : url.replace('.png', '_hires.png')
}

const colors: Record<string, string> = {
  Grass: '#5fa338', Fire: '#d6543f', Water: '#3187c4', Lightning: '#e3b62c',
  Psychic: '#9a5aa8', Fighting: '#b06a3a', Darkness: '#31495c', Metal: '#7e8c99',
  Dragon: '#b8963e', Colorless: '#a8a49c',
}

export function energyColor(nameEN: string): string {
  const el = Object.keys(colors).find(k => nameEN.startsWith(k))
  return el ? colors[el] : '#a8a49c'
}

export function energyDotStyle(nameEN: string): React.CSSProperties {
  return { backgroundColor: energyColor(nameEN) }
}
