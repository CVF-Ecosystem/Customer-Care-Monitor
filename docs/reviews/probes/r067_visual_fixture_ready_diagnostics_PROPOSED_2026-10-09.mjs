// NEW root-owned proposed fixture; only capture entry differs from literal-path fixture. Not executed.
// Root reviewer fixture repair: literal module specifiers; historical percent-encoded fixture remains unchanged.
// Isolated saved-run UI screenshot fixture. It does not start the CCMA app or call an API.
import { spawn } from 'node:child_process'
import { existsSync, lstatSync, mkdirSync, mkdtempSync, readFileSync, realpathSync, rmdirSync, symlinkSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { setTimeout as delay } from 'node:timers/promises'
import net from 'node:net'

const here = dirname(fileURLToPath(import.meta.url))
const repoRoot = resolve(here, '../../..')
const frontend = join(repoRoot, 'frontend')
const modules = join(frontend, 'node_modules')
const component = join(frontend, 'src/components/ui/RunObservationPanel.vue')
const componentDir = dirname(component)
const helperDir = join(frontend, 'src/views/Jobs/job-detail')
const localeDir = join(frontend, 'src/i18n')
const screenshots = join(here, 'r067_capture_ready_diagnostics_PROPOSED_2026-10-09.mjs')
const tempRoot = mkdtempSync(join(tmpdir(), 'ccmai-r067-visual-'))
const fixtureRoot = join(tempRoot, 'fixture')
const screenshotRoot = join(tempRoot, 'screenshots')
const configPath = join(fixtureRoot, 'vite.config.mjs')
const fixtureNodeModules = join(fixtureRoot, 'node_modules')
const csp = "default-src 'none'; script-src 'self' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:; connect-src 'self' ws://127.0.0.1:*; base-uri 'none'; form-action 'none'; frame-src 'none'; object-src 'none'"
let fixtureJunctionCreated = false

function assertLocalPrerequisites() {
  for (const path of [modules, component, helperDir, localeDir, screenshots]) {
    if (!existsSync(path)) throw new Error(`Required cached/local artifact missing: ${path}`)
  }
}

function safeJson(value) { return JSON.stringify(value) }

function writeFixture() {
  mkdirSync(fixtureRoot, { recursive: true })
  mkdirSync(screenshotRoot, { recursive: true })
  symlinkSync(modules, fixtureNodeModules, 'junction')
  fixtureJunctionCreated = true

  const componentPath = `/@fs/${component.replaceAll('\\', '/')}`
  const enPath = `/@fs/${join(localeDir, 'en.ts').replaceAll('\\', '/')}`
  const viPath = `/@fs/${join(localeDir, 'vi.ts').replaceAll('\\', '/')}`
  const allow = [fixtureRoot, componentDir, helperDir, localeDir].map(safeJson).join(', ')
  writeFileSync(configPath, `
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
export default defineConfig({
  root: ${safeJson(fixtureRoot)},
  plugins: [vue()],
  server: { host: '127.0.0.1', strictPort: true, fs: { allow: [${allow}] } },
})
`, 'utf8')

  writeFileSync(join(fixtureRoot, 'index.html'), `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta http-equiv="Content-Security-Policy" content="${csp}">
    <title>Saved-run observation fixture</title>
    <style>
      :root { --v-theme-text-primary: 25, 33, 45; --v-theme-text-muted: 90, 99, 112; --v-theme-border: 192, 200, 210; --v-theme-surface: 255, 255, 255; --v-theme-primary: 35, 90, 180; color-scheme: light; }
      * { box-sizing: border-box; }
      body { margin: 0; padding: 24px; background: #f4f6f9; color: #19212d; font: 16px/1.5 system-ui, sans-serif; }
      main { margin: 0 auto; max-width: 760px; padding: 20px; background: white; border: 1px solid #d8dee8; border-radius: 12px; }
      h1 { margin: 0 0 12px; font-size: 22px; }
      .fixture-note { margin: 0 0 16px; color: #586273; }
      @media (max-width: 600px) { body { padding: 8px; } main { padding: 12px; } h1 { font-size: 19px; } }
    </style>
  </head>
  <body><div id="app"></div><script type="module" src="/main.ts"></script></body>
</html>
`, 'utf8')

  const preparation = {
    version: 'ccmai.source-preparation.v1', scope: 'preparation_only', tenant_id: '11111111-1111-4111-8111-111111111111',
    job_id: '22222222-2222-4222-8222-222222222222', run_id: '33333333-3333-4333-8333-333333333333',
    mode: 'conditional', metadata_incomplete: false, selection_status: 'COMPLETE', selected: 2, visited: 2, unvisited: 0,
    counts: { PREPARED_FOR_INFERENCE: 1, UNCHANGED_VERIFIED: 1 },
    entries: [
      { conversation_id: '44444444-4444-4444-8444-444444444444', outcome: 'PREPARED_FOR_INFERENCE', snapshot_schema: 'ccma.snapshot.v1', snapshot_digest: 'a'.repeat(64), coverage: 'complete', reference: { state: 'NONE' }, metadata_incomplete: false },
      { conversation_id: '55555555-5555-4555-8555-555555555555', outcome: 'UNCHANGED_VERIFIED', snapshot_schema: 'ccma.snapshot.v1', snapshot_digest: 'b'.repeat(64), coverage: 'partial', coverage_reasons: ['HISTORY_WINDOWED'], reference: { state: 'VERIFIED', evaluation_id: '66666666-6666-4666-8666-666666666666', run_id: '77777777-7777-4777-8777-777777777777', snapshot_id: '88888888-8888-4888-8888-888888888888' }, metadata_incomplete: false },
    ],
    omitted_entries: 0, entries_complete: true, scan_complete: true,
  }
  const usage = {
    version: 'ccmai.usage-observation.v1', scope: 'successful_interface_response_local_estimate',
    token_basis: 'INTERFACE_VALUES_PRESENCE_UNAVAILABLE', billing: 'NOT_OBSERVED', price_revision: 'NOT_CAPTURED',
    responses: 1, invalid_tokens: 0, priced: 1, unpriced: 0, invalid_costs: 0, token_overflow: false,
    cost_overflow: false, counter_overflow: false, input_tokens: 0, output_tokens: 0,
    local_estimate_usd: 0, tokens_complete: true, cost_complete: true,
  }
  const execution = {
    version: 'ccmai.source-execution.v1', scope: 'analyzer_provider_interface_only', tenant_id: preparation.tenant_id,
    job_id: preparation.job_id, run_id: preparation.run_id, mode: 'conditional', metadata_incomplete: false,
    calls_begun: 1, response_returned: 1, error_returned: 0, interrupted: 0, in_flight: 0,
    item_count: 1, items_saved: 1, items_save_failed: 0, items_not_published: 0, items_pending: 0,
    usage_writes: { NOT_ATTEMPTED: 0, WRITE_SUCCEEDED: 1, WRITE_FAILED: 0, WRITE_OUTCOME_UNKNOWN: 0 },
    parsing: { NOT_ATTEMPTED: 0, ACCEPTED: 0, REJECTED: 0, NOT_SEPARATELY_OBSERVABLE: 1 },
    calls: [{ sequence: 1, method: 'SINGLE', item_count: 1, member_ids: ['55555555-5555-4555-8555-555555555555'], omitted_members: 0,
      members_complete: true, metadata_incomplete: false, invocation_outcome: 'RESPONSE_RETURNED', usage_write_outcome: 'WRITE_SUCCEEDED',
      parsing_outcome: 'NOT_SEPARATELY_OBSERVABLE', items_saved: 1, items_save_failed: 0, items_not_published: 0, items_pending: 0 }],
    omitted_calls: 0, omitted_members: 0, entries_complete: true, members_complete: true, execution_complete: true,
    stop_reason: 'NONE', usage_observation: usage,
  }
  const rule = {
    version: 'ccmai.rule-observation.v1', scope: 'analyzer_job_input_only', tenant_id: preparation.tenant_id,
    job_id: preparation.job_id, run_id: preparation.run_id, job_type: 'qc_analysis', metadata_incomplete: false,
    rule_authority: 'JOB_INPUT_OBSERVED', policy_version_status: 'NOT_AVAILABLE', permission_status: 'NOT_OBSERVED_AT_ANALYZER',
    wait_data_status: 'NOT_IMPLEMENTED', fingerprint_status: 'OBSERVED', fingerprint: 'c'.repeat(64),
  }
  const summary = safeJson({ source_preparation: preparation, source_execution: execution, rule_observation: rule })
  writeFileSync(join(fixtureRoot, 'main.ts'), `
import { createApp, h, onMounted } from 'vue'
import { createI18n } from 'vue-i18n'
import Panel from '${componentPath}'
import en from '${enPath}'
import vi from '${viPath}'
const locale = location.pathname === '/vi' ? 'vi' : 'en'
const context = { tenantId: '11111111-1111-4111-8111-111111111111', jobId: '22222222-2222-4222-8222-222222222222', runId: '33333333-3333-4333-8333-333333333333', runJobId: '22222222-2222-4222-8222-222222222222' }
const summary = ${safeJson(summary)}
const i18n = createI18n({ legacy: false, locale, fallbackLocale: 'en', messages: { en, vi } })
const App = { setup() { onMounted(() => { const details = document.querySelector('details'); if (details) details.open = true }); return () => h('main', [h('h1', locale === 'vi' ? 'Quan sát lần chạy đã lưu' : 'Saved-run observations'), h('p', { class: 'fixture-note' }, locale === 'vi' ? 'Dữ liệu giả lập, không gửi yêu cầu đến API.' : 'Synthetic saved receipt, with no API request.'), h(Panel, { summary, context })]) } }
createApp(App).use(i18n).mount('#app')
`, 'utf8')
}

function allocatePort() {
  return new Promise((resolvePort, reject) => {
    const server = net.createServer()
    server.once('error', reject)
    server.listen(0, '127.0.0.1', () => {
      const address = server.address()
      server.close(() => resolvePort(address.port))
    })
  })
}

async function waitForFixture(url, process) {
  for (let attempt = 0; attempt < 100; attempt++) {
    if (process.exitCode !== null) throw new Error(`Vite exited early (${process.exitCode})`)
    try {
      const response = await fetch(url)
      if (response.ok) return
    } catch {}
    await delay(100)
  }
  throw new Error('Loopback fixture did not become ready')
}

function waitForExit(child, label, timeoutMs = 5000) {
  if (!child || child.exitCode !== null || child.signalCode !== null) return Promise.resolve()
  return new Promise((resolveExit, reject) => {
    const onExit = () => {
      clearTimeout(timeout)
      resolveExit()
    }
    const timeout = setTimeout(() => {
      child.removeListener('exit', onExit)
      reject(new Error(`${label} process did not exit within ${timeoutMs}ms`))
    }, timeoutMs)
    child.once('exit', onExit)
  })
}

async function stopAndVerifyExit(child, label) {
  if (!child || child.exitCode !== null || child.signalCode !== null) return
  const exited = waitForExit(child, label)
  child.kill()
  await exited
}

function removeVerifiedFixtureJunction() {
  if (!fixtureJunctionCreated) return
  const stats = lstatSync(fixtureNodeModules)
  if (!stats.isSymbolicLink()) throw new Error('Fixture node_modules path is no longer a symbolic link; leaving it untouched')
  const expectedTarget = realpathSync(modules).toLowerCase()
  const actualTarget = realpathSync(fixtureNodeModules).toLowerCase()
  if (actualTarget !== expectedTarget) throw new Error('Fixture node_modules junction target changed; leaving it untouched')
  rmdirSync(fixtureNodeModules)
  fixtureJunctionCreated = false
}

async function main() {
  assertLocalPrerequisites()
  let server
  try {
    writeFixture()
    const port = await allocatePort()
    const vite = join(modules, 'vite/bin/vite.js')
    const node = process.execPath
    server = spawn(node, [vite, '--config', configPath, '--host', '127.0.0.1', '--port', String(port), '--strictPort'], {
      cwd: fixtureRoot, stdio: ['ignore', 'pipe', 'pipe'], windowsHide: true,
    })
    let serverOutput = ''
    server.stdout.on('data', chunk => { serverOutput += chunk.toString() })
    server.stderr.on('data', chunk => { serverOutput += chunk.toString() })
    const base = `http://127.0.0.1:${port}`
    await waitForFixture(`${base}/en`, server)
    const localAppData = process.env.LOCALAPPDATA ?? ''
    const chrome = join(localAppData || 'C:/Users/tiennm/AppData/Local', 'ms-playwright/chromium-1228/chrome-win64/chrome.exe')
    if (!existsSync(chrome)) throw new Error('Cached Playwright Chromium 1228 was not found; no download attempted')
    const capture = spawn(node, [screenshots, '--base', base, '--out', screenshotRoot,
      '--routes', 'en=/en,vi=/vi', '--themes', 'light', '--viewports', 'desktop,mobile', '--wait', '250', '--chrome', chrome], {
      cwd: repoRoot, stdio: ['ignore', 'pipe', 'pipe'], windowsHide: true,
    })
    let captureOutput = ''
    capture.stdout.on('data', chunk => { captureOutput += chunk.toString() })
    capture.stderr.on('data', chunk => { captureOutput += chunk.toString() })
    const exitCode = await new Promise((resolveExit, reject) => {
      capture.once('error', reject)
      capture.once('close', resolveExit)
    })
    writeFileSync(join(tempRoot, 'vite.log'), serverOutput, 'utf8')
    writeFileSync(join(tempRoot, 'capture.log'), captureOutput, 'utf8')
    if (exitCode !== 0) throw new Error(`Screenshot capture failed (${exitCode}); raw output retained in ${tempRoot}`)
    const report = JSON.parse(readFileSync(join(screenshotRoot, 'report.json'), 'utf8'))
    if (report.pages.length !== 4 || report.summary.pagesWithJsErrors !== 0 || report.summary.pagesWithHorizontalOverflow !== 0 || report.summary.externalRequests.length !== 0 || report.pages.some(page => page.logErrors?.length || !page.keyboard?.tabSummary || !page.keyboard?.enterOpened || !page.keyboard?.spaceClosed)) {
      throw new Error(`Visual fixture check failed; report retained in ${tempRoot}`)
    }
    console.log(JSON.stringify({ status: 'PASS', pages: report.pages.length, locales: ['en', 'vi'], viewports: ['desktop', 'mobile'], externalRequests: report.summary.externalRequests, artifacts: screenshotRoot, logs: tempRoot }))
  } finally {
    await stopAndVerifyExit(server, 'Vite')
    removeVerifiedFixtureJunction()
  }
}

main().catch(error => {
  console.error(error)
  process.exitCode = 1
})
