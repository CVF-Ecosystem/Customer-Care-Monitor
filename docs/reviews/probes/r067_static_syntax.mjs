// Cached syntax-only source audit: no emit, semantic typecheck, build, test runner or network.
import { createRequire } from 'node:module'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
const require = createRequire(resolve('frontend/package.json'))
const ts = require('typescript')
const sfc = require('@vue/compiler-sfc')
const files = ['frontend/src/views/Jobs/JobDetail.vue','frontend/src/views/Jobs/job-detail/run-observation.ts','frontend/src/components/ui/RunObservationPanel.vue','frontend/src/i18n/en.ts','frontend/src/i18n/vi.ts','frontend/src/__tests__/run-observation.spec.ts','frontend/src/__tests__/run-observation-panel.spec.ts','frontend/src/__tests__/run-observation-job-detail.spec.ts']
const errors = []
for (const file of files) {
  const text = readFileSync(file, 'utf8')
  const blocks = file.endsWith('.vue') ? (() => { const parsed = sfc.parse(text, { filename: file }); errors.push(...parsed.errors.map(error => ({ file, message: String(error) }))); return [parsed.descriptor.script, parsed.descriptor.scriptSetup].filter(Boolean).map(block => block.content) })() : [text]
  for (const block of blocks) {
    const source = ts.createSourceFile(file, block, ts.ScriptTarget.Latest, true, ts.ScriptKind.TS)
    if (!Array.isArray(source.parseDiagnostics)) throw new Error('TypeScript parser diagnostics unavailable')
    errors.push(...source.parseDiagnostics.map(error => ({ file, line: source.getLineAndCharacterOfPosition(error.start ?? 0).line + 1, message: ts.flattenDiagnosticMessageText(error.messageText, '\n') })))
  }
}
console.log(JSON.stringify({ kind: 'syntax_only_static_audit', files: files.length, errors, vitest: 0, go: 0, fullTypecheckBuild: 0 }, null, 2))
process.exitCode = errors.length ? 1 : 0
