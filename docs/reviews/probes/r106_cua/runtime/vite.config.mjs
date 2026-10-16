// R105 candidate template. Only the three JSON string literals below may be replaced
// after source review; generated file must be hashed and confined to R105 runtime.
import { createRequire } from 'node:module'
import { pathToFileURL } from 'node:url'
import { join } from 'node:path'

const frontendRoot = "D:/UNG DUNG AI/TOOL AI 2026/CVF-Workspace/Customer-Care-Monitor-AI/frontend"
const apiOrigin = "http://127.0.0.1:60641"
const healthNonce = "2653e1a5d4e933363d6439764e34f80a"
if (!/^http:\/\/127\.0\.0\.1:[1-9][0-9]{0,4}$/.test(apiOrigin)) {
  throw new Error('R105 requires a fresh loopback API origin')
}
const requireFromFrontend = createRequire(join(frontendRoot, 'package.json'))
const importFromFrontend = async (name) =>
  import(pathToFileURL(requireFromFrontend.resolve(name)).href)

const [{ defineConfig }, { default: vue }, { default: vuetify }] = await Promise.all([
  importFromFrontend('vite'),
  importFromFrontend('@vitejs/plugin-vue'),
  importFromFrontend('vite-plugin-vuetify'),
])

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
