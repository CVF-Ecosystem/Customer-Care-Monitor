import assert from 'node:assert/strict'
import { pathToFileURL } from 'node:url'
const config = (await import(pathToFileURL(process.argv[2]).href)).default
assert.equal(config.root.replaceAll('\\', '/'), process.argv[3].replaceAll('\\', '/'))
assert.equal(config.envDir, false)
assert.deepEqual(config.envPrefix, ['R105_DISABLED_AMBIENT_PREFIX_'])
assert.equal(config.server.host, '127.0.0.1')
assert.equal(config.server.port, 3011)
assert.equal(config.server.strictPort, true)
assert.deepEqual(config.server.proxy, {'/api': {target: 'http://127.0.0.1:52962', changeOrigin: true}})
assert.equal(config.plugins.length, 3)
assert.ok(config.plugins[0])
assert.ok(config.plugins[1])
assert.equal(config.plugins[2].name, 'r105-owned-health')
console.log(JSON.stringify({result: 'PASS', node: process.version, configRoot: config.root, envDir: config.envDir, host: config.server.host, port: config.server.port, strictPort: config.server.strictPort, proxy: config.server.proxy, pluginGroups: config.plugins.length, mode: 'MODULE_IMPORT_ONLY_NO_SERVER'}))
