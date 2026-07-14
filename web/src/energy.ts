const svgRaw = import.meta.glob('./assets/energy/*.svg', { as: 'raw', eager: true })

function toDataUri(path: string): string {
  const raw = svgRaw[path] as string | undefined
  return raw ? 'data:image/svg+xml,' + encodeURIComponent(raw) : ''
}

const typeToFile: Record<string, string> = {
  Grass: './assets/energy/grass.svg',
  Fire: './assets/energy/fire.svg',
  Water: './assets/energy/water.svg',
  Lightning: './assets/energy/electric.svg',
  Psychic: './assets/energy/psychic.svg',
  Fighting: './assets/energy/fighting.svg',
  Darkness: './assets/energy/dark.svg',
  Metal: './assets/energy/steel.svg',
  Dragon: './assets/energy/dragon.svg',
  Colorless: './assets/energy/normal.svg',
}

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
  const color = energyColor(nameEN)
  const el = Object.keys(typeToFile).find(k => nameEN.startsWith(k))
  const uri = el ? toDataUri(typeToFile[el]) : ''
  return uri
    ? {
        backgroundColor: color,
        backgroundImage: `url("${uri}")`,
        backgroundPosition: 'center',
        backgroundSize: '150%',
        backgroundRepeat: 'no-repeat',
      }
    : { backgroundColor: color }
}
