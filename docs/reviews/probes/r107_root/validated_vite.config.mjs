// R107 successor template; historical R105 marker/health protocol retained.
// Replace only three JSON literals; hash generated file in owned runtime.
import { createRequire } from 'node:module'
import { pathToFileURL } from 'node:url'
import { join } from 'node:path'

const frontendRoot = "D:/UNG DUNG AI/TOOL AI 2026/CVF-Workspace/Customer-Care-Monitor-AI/frontend"
const apiOrigin = "http://127.0.0.1:52962"
const healthNonce = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
if (!/^http:\/\/127\.0\.0\.1:[1-9][0-9]{0,4}$/.test(apiOrigin)) {
  throw new Error('R105 requires a fresh loopback API origin')
}
const requireFromFrontend = createRequire(join(frontendRoot, 'package.json'))
const importFromFrontend = async (name) =>
  import(pathToFileURL(requireFromFrontend.resolve(name)).href)

const { defineConfig } = await importFromFrontend('vite')
const { default: vue } = await importFromFrontend('@vitejs/plugin-vue')
const { default: vuetify } = await importFromFrontend('vite-plugin-vuetify')

export default defineConfig({
  root: frontendRoot,
  envDir: false,
  envPrefix: ['R105_DISABLED_AMBIENT_PREFIX_'],
  plugins: [
    vue(),
    vuetify({ autoImport: true }),
    { name: 'r105-owned-health', configureServer(server) {
      server.middlewares.use('/__r105_health', (_request, response) => {
        response.setHeader('Content-Type', 'text/plain')
        response.setHeader('Content-Length', String(Buffer.byteLength(healthNonce)))
        response.end(healthNonce)
      })
    } },
  ],
  server: {
    host: '127.0.0.1',
    port: 3011,
    strictPort: true,
    proxy: { '/api': { target: apiOrigin, changeOrigin: true } },
  },
})
