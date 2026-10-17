// Verification-only config. No project .env loading or API proxy.
import { createRequire } from 'node:module'
import { fileURLToPath, pathToFileURL } from 'node:url'
import { join } from 'node:path'

const frontendRoot = fileURLToPath(new URL('../../../../frontend', import.meta.url))
const fromFrontend = createRequire(join(frontendRoot, 'package.json'))
const load = async name => import(pathToFileURL(fromFrontend.resolve(name)).href)
const { defineConfig } = await load('vite')
const { default: vue } = await load('@vitejs/plugin-vue')
const { default: vuetify } = await load('vite-plugin-vuetify')

export default defineConfig({
  root: frontendRoot,
  envDir: false,
  envPrefix: ['R110_DISABLED_AMBIENT_PREFIX_'],
  plugins: [vue(), vuetify({ autoImport: true })],
  test: {
    server: { deps: { inline: ['vuetify'] } },
    maxWorkers: 1,
  },
})
