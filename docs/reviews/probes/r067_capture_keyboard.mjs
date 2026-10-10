// R067 root-owned isolated synthetic UI capture; derived from read-only ui-screenshots.mjs.
// CCMAI-UX-000: dependency-free screenshot capture over the Chrome DevTools Protocol.
// Captures each route at desktop and mobile size in light and dark theme, and records
// uncaught exceptions, console errors and requests to hosts other than the app.
//
// Usage:
//   node scripts/ui-screenshots.mjs --base http://127.0.0.1:4173 --out <dir> \
//     --routes "/wireframes/design-system,/wireframes/design-system?dialog=review" \
//     [--token-file <file with access token>] [--themes light,dark] [--viewports desktop,mobile] \
//     [--locale vi] [--wait 1500] [--chrome <path>]
// Each --routes entry may be "name=path" to control the file name.
// Exit code 1 when any page has a JS error; the JSON report lists everything.

import { spawn } from 'node:child_process'
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'
import { setTimeout as sleep } from 'node:timers/promises'

const VIEWPORTS = {
  desktop: { width: 1440, height: 900, mobile: false, scale: 1 },
  mobile: { width: 390, height: 844, mobile: true, scale: 2 },
}
const MAX_CAPTURE_HEIGHT = 8000

function parseArgs(argv) {
  const out = {}
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i]
    if (!a.startsWith('--')) continue
    const key = a.slice(2)
    const next = argv[i + 1]
    if (next === undefined || next.startsWith('--')) out[key] = true
    else {
      out[key] = next
      i++
    }
  }
  return out
}

function findChrome(explicit) {
  const candidates = [
    explicit,
    process.env.CHROME_PATH,
    'C:/Program Files/Google/Chrome/Application/chrome.exe',
    'C:/Program Files (x86)/Google/Chrome/Application/chrome.exe',
    'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe',
    '/usr/bin/google-chrome',
    '/usr/bin/chromium',
    '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
  ].filter(Boolean)
  const found = candidates.find((p) => existsSync(p))
  if (!found) throw new Error('Chrome/Edge not found; pass --chrome <path>')
  return found
}

class Cdp {
  constructor(ws) {
    this.ws = ws
    this.id = 0
    this.pending = new Map()
    this.listeners = new Map()
    ws.addEventListener('message', (ev) => {
      const msg = JSON.parse(typeof ev.data === 'string' ? ev.data : ev.data.toString())
      if (msg.id && this.pending.has(msg.id)) {
        const { resolve: ok, reject } = this.pending.get(msg.id)
        this.pending.delete(msg.id)
        if (msg.error) reject(new Error(`${msg.error.message} (${msg.error.code})`))
        else ok(msg.result)
      } else if (msg.method) {
        for (const fn of this.listeners.get(msg.method) ?? []) fn(msg.params)
      }
    })
  }
  static async connect(url) {
    const ws = new WebSocket(url)
    await new Promise((ok, fail) => {
      ws.addEventListener('open', ok, { once: true })
      ws.addEventListener('error', fail, { once: true })
    })
    return new Cdp(ws)
  }
  send(method, params = {}) {
    const id = ++this.id
    this.ws.send(JSON.stringify({ id, method, params }))
    return new Promise((ok, reject) => this.pending.set(id, { resolve: ok, reject }))
  }
  on(method, fn) {
    if (!this.listeners.has(method)) this.listeners.set(method, [])
    this.listeners.get(method).push(fn)
  }
  once(method, timeoutMs) {
    return new Promise((ok) => {
      const timer = setTimeout(() => ok(null), timeoutMs)
      const fn = (p) => {
        clearTimeout(timer)
        const list = this.listeners.get(method)
        list.splice(list.indexOf(fn), 1)
        ok(p)
      }
      this.on(method, fn)
    })
  }
  close() {
    this.ws.close()
  }
}

async function launchChrome(chromePath) {
  const profile = mkdtempSync(join(tmpdir(), 'ccma-shots-'))
  const proc = spawn(
    chromePath,
    [
      '--headless=new',
      '--remote-debugging-port=0',
      `--user-data-dir=${profile}`,
      '--no-first-run',
      '--no-default-browser-check',
      '--disable-extensions',
      '--hide-scrollbars',
      '--force-color-profile=srgb',
      'about:blank',
    ],
    { stdio: 'ignore' },
  )
  const portFile = join(profile, 'DevToolsActivePort')
  for (let i = 0; i < 100 && !existsSync(portFile); i++) await sleep(100)
  if (!existsSync(portFile)) throw new Error('Chrome did not start (no DevToolsActivePort)')
  const port = readFileSync(portFile, 'utf8').split('\n')[0].trim()
  return { proc, profile, port }
}

function slugify(s) {
  return s.replace(/^\//, '').replace(/[^a-zA-Z0-9]+/g, '-').replace(/^-|-$/g, '') || 'root'
}

async function main() {
  const args = parseArgs(process.argv.slice(2))
  const base = String(args.base ?? 'http://127.0.0.1:4173').replace(/\/$/, '')
  const outDir = resolve(String(args.out ?? 'ui-screenshots-out'))
  const routes = String(args.routes ?? '/wireframes/design-system')
    .split(',')
    .map((r) => r.trim())
    .filter(Boolean)
    .map((r) => (r.includes('=') && !r.startsWith('/') ? { name: r.split('=')[0], path: r.slice(r.indexOf('=') + 1) } : { name: slugify(r), path: r }))
  const themes = String(args.themes ?? 'light,dark').split(',')
  const viewports = String(args.viewports ?? 'desktop,mobile').split(',')
  const locale = String(args.locale ?? 'vi')
  const waitMs = Number(args.wait ?? 1500)
  const token = args['token-file'] ? readFileSync(String(args['token-file']), 'utf8').trim() : ''
  const baseOrigin = new URL(base).origin

  mkdirSync(outDir, { recursive: true })
  const chrome = await launchChrome(findChrome(args.chrome))
  const report = { base, generatedAt: new Date().toISOString(), pages: [] }

  try {
    const target = await (await fetch(`http://127.0.0.1:${chrome.port}/json/new?about:blank`, { method: 'PUT' })).json()
    const cdp = await Cdp.connect(target.webSocketDebuggerUrl)
    await cdp.send('Page.enable')
    await cdp.send('Runtime.enable')
    await cdp.send('Log.enable')
    await cdp.send('Network.enable')
    await cdp.send('Fetch.enable', { patterns: [{ urlPattern: 'http*', requestStage: 'Request' }] })
    cdp.on('Fetch.requestPaused', async p => {
      const url = p.request.url
      const local = new URL(url).origin === new URL(base).origin
      if (local) await cdp.send('Fetch.continueRequest', { requestId: p.requestId })
      else await cdp.send('Fetch.failRequest', { requestId: p.requestId, errorReason: 'BlockedByClient' })
    })
    // A background headless target throttles requestAnimationFrame, which leaves
    // animated charts (Chart.js) mid-animation in the capture. Keep the page focused.
    await cdp.send('Page.bringToFront')
    await cdp.send('Emulation.setFocusEmulationEnabled', { enabled: true })

    let current = null
    cdp.on('Runtime.exceptionThrown', (p) => {
      current?.jsErrors.push(p.exceptionDetails?.exception?.description ?? p.exceptionDetails?.text ?? 'exception')
    })
    cdp.on('Runtime.consoleAPICalled', (p) => {
      if (p.type === 'error') current?.consoleErrors.push(p.args.map((a) => a.value ?? a.description ?? '').join(' '))
    })
    cdp.on('Log.entryAdded', (p) => {
      if (p.entry.level === 'error') current?.logErrors.push(`${p.entry.source}: ${p.entry.text} ${p.entry.url ?? ''}`.trim())
    })
    cdp.on('Network.requestWillBeSent', (p) => {
      const url = p.request.url
      if (/^https?:/.test(url) && new URL(url).origin !== baseOrigin) current?.externalRequests.push(url)
    })

    for (const vpName of viewports) {
      const vp = VIEWPORTS[vpName]
      if (!vp) throw new Error(`unknown viewport ${vpName}`)
      for (const theme of themes) {
        // Storage is set before any app script runs, on every navigation.
        const init = await cdp.send('Page.addScriptToEvaluateOnNewDocument', {
          source: `try {
            localStorage.setItem('ccma_theme', ${JSON.stringify(theme)});
            localStorage.setItem('cqa_locale', ${JSON.stringify(locale)});
            ${token ? `localStorage.setItem('cqa_access_token', ${JSON.stringify(token)});` : ''}
          } catch (e) {}`,
        })
        for (const route of routes) {
          await cdp.send('Emulation.setDeviceMetricsOverride', {
            width: vp.width,
            height: vp.height,
            deviceScaleFactor: vp.scale,
            mobile: vp.mobile,
          })
          current = { route: route.path, viewport: vpName, theme, jsErrors: [], consoleErrors: [], logErrors: [], externalRequests: [] }
          const loaded = cdp.once('Page.loadEventFired', 20000)
          await cdp.send('Page.navigate', { url: base + route.path })
          await loaded
          await sleep(waitMs)
          await cdp.send('Runtime.evaluate', { expression: `document.querySelectorAll('details').forEach(d => d.open = false); document.activeElement?.blur()` })
          const press = async (key, code, vk) => {
            await cdp.send('Input.dispatchKeyEvent', { type: 'keyDown', key, code, windowsVirtualKeyCode: vk, nativeVirtualKeyCode: vk })
            await cdp.send('Input.dispatchKeyEvent', { type: 'keyUp', key, code, windowsVirtualKeyCode: vk, nativeVirtualKeyCode: vk })
            await sleep(50)
          }
          const read = async expression => (await cdp.send('Runtime.evaluate', { expression, returnByValue: true })).result.value
          await press('Tab', 'Tab', 9)
          const tabSummary = await read(`document.activeElement?.tagName === 'SUMMARY'`)
          await press('Enter', 'Enter', 13)
          const enterOpened = await read(`document.querySelector('details')?.open === true`)
          await press(' ', 'Space', 32)
          const spaceClosed = await read(`document.querySelector('details')?.open === false`)
          if (!tabSummary || !enterOpened || !spaceClosed) throw new Error('Native details keyboard check failed')
          current.keyboard = { tabSummary, enterOpened, spaceClosed }
          await press('Enter', 'Enter', 13)
          const metrics = await cdp.send('Page.getLayoutMetrics')
          const size = metrics.cssContentSize ?? metrics.contentSize
          const height = Math.min(Math.max(Math.ceil(size.height), vp.height), MAX_CAPTURE_HEIGHT)
          // Grow the viewport to the full page first and let the page settle: capturing
          // beyond the viewport resizes it at capture time, and responsive charts
          // (Chart.js) are then caught mid-redraw with their points bunched to one side.
          await cdp.send('Emulation.setDeviceMetricsOverride', {
            width: vp.width,
            height,
            deviceScaleFactor: vp.scale,
            mobile: vp.mobile,
          })
          await sleep(1500)
          const shot = await cdp.send('Page.captureScreenshot', {
            format: 'png',
            clip: { x: 0, y: 0, width: vp.width, height, scale: 1 },
          })
          const file = `${route.name}--${vpName}--${theme}.png`
          writeFileSync(join(outDir, file), Buffer.from(shot.data, 'base64'))
          current.file = file
          current.finalUrl = (await cdp.send('Runtime.evaluate', { expression: 'location.href', returnByValue: true })).result.value
          current.horizontalOverflow = (
            await cdp.send('Runtime.evaluate', {
              expression: 'document.documentElement.scrollWidth > document.documentElement.clientWidth + 1',
              returnByValue: true,
            })
          ).result.value
          report.pages.push(current)
          const flag = current.jsErrors.length || current.consoleErrors.length ? 'JS-ERROR' : 'ok'
          console.log(`${flag.padEnd(8)} ${file}${current.horizontalOverflow ? '  (horizontal overflow)' : ''}`)
        }
        await cdp.send('Page.removeScriptToEvaluateOnNewDocument', { identifier: init.identifier })
      }
    }
    cdp.close()
  } finally {
    chrome.proc.kill()
    await sleep(500)
    try {
      rmSync(chrome.profile, { recursive: true, force: true })
    } catch {
      // Chrome may still hold the profile briefly on Windows; it is in the temp folder.
    }
  }

  const jsErrorPages = report.pages.filter((p) => p.jsErrors.length || p.consoleErrors.length)
  report.summary = {
    pages: report.pages.length,
    pagesWithJsErrors: jsErrorPages.length,
    pagesWithHorizontalOverflow: report.pages.filter((p) => p.horizontalOverflow).length,
    externalRequests: [...new Set(report.pages.flatMap((p) => p.externalRequests))],
  }
  writeFileSync(join(outDir, 'report.json'), JSON.stringify(report, null, 2))
  console.log(JSON.stringify(report.summary))
  process.exit(jsErrorPages.length ? 1 : 0)
}

main().catch((err) => {
  console.error(err)
  process.exit(2)
})
