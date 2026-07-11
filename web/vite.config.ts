import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Dev: `npm run dev` sobe o Vite com proxy de /api para o servidor Go
// (`task play`). Build: `npm run build` gera web/dist, embutido no binário
// Go via go:embed (web/embed.go).
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: { '/api': 'http://localhost:8080' },
  },
})
