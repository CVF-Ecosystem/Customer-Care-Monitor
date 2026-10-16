// R109 owned health check. The supervisor replaces only these JSON literals.
import { createRequire } from 'node:module'
import { closeSync, fsyncSync, openSync, writeSync } from 'node:fs'
import { pathToFileURL } from 'node:url'
import { join } from 'node:path'

const frontendRoot = "D:/UNG DUNG AI/TOOL AI 2026/CVF-Workspace/Customer-Care-Monitor-AI/frontend"
const campaignDirectory = "D:/UNG DUNG AI/TOOL AI 2026/CVF-Workspace/Customer-Care-Monitor-AI/docs/reviews/probes/r109_root/harness_validation"
const apiOrigin = "http://127.0.0.1:52962"
const healthNonce = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const postLedgerPath = join(campaignDirectory, 'ui_post_submissions.jsonl')
const postLimit = 14
let postAttempts = 0
let forwardedPosts = 0
if (!/^http:\/\/127\.0\.0\.1:[1-9][0-9]{0,4}$/.test(apiOrigin)) {
  throw new Error('R109 requires a fresh loopback API origin')
}

const requireFromFrontend = createRequire(join(frontendRoot, 'package.json'))
const importFromFrontend = async (name) =>
  import(pathToFileURL(requireFromFrontend.resolve(name)).href)

const appendPostObservation = (record) => {
  const fd = openSync(postLedgerPath, 'a')
  try {
    writeSync(fd, `${JSON.stringify(record)}\n`)
    fsyncSync(fd)
  } finally {
    closeSync(fd)
  }
}

const { defineConfig } = await importFromFrontend('vite')
const { default: vue } = await importFromFrontend('@vitejs/plugin-vue')
const { default: vuetify } = await importFromFrontend('vite-plugin-vuetify')

export default defineConfig({
  root: frontendRoot,
  envDir: false,
  envPrefix: ['R109_DISABLED_AMBIENT_PREFIX_'],
  plugins: [
    { name: 'r109-ui-post-budget', enforce: 'pre', configureServer(server) {
      server.middlewares.use((request, response, next) => {
        const pathname = (request.url || '').split('?', 1)[0]
        if (request.method !== 'POST' || (pathname !== '/api' && !pathname.startsWith('/api/'))) {
          return next()
        }
        postAttempts += 1
        const forwarded = forwardedPosts < postLimit
        if (forwarded) forwardedPosts += 1
        try {
          appendPostObservation({
            ordinal: postAttempts,
            state: forwarded ? 'FORWARDED_TO_API' : 'BLOCKED_POST_LIMIT',
            routeClass: 'API',
            limit: postLimit,
          })
        } catch {
          response.statusCode = 503
          response.end()
          return
        }
        if (!forwarded) {
          response.statusCode = 429
          response.setHeader('X-R109-Harness-Budget-Denial', 'POST_LIMIT')
          response.end()
          return
        }
        next()
      })
    } },
    vue(),
    vuetify({ autoImport: true }),
    { name: 'r109-owned-health', configureServer(server) {
      server.middlewares.use('/__r109_health', (request, response) => {
        if (request.method !== 'GET') {
          response.statusCode = 405
          response.end()
          return
        }
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
