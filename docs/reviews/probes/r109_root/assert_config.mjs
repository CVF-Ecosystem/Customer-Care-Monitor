import assert from 'node:assert/strict'
import { pathToFileURL } from 'node:url'
const config = (await import(pathToFileURL(process.argv[2]).href)).default
assert.equal(config.root.replaceAll('\\', '/'), process.argv[3].replaceAll('\\', '/'))
assert.equal(config.envDir, false)
assert.deepEqual(config.envPrefix, ['R109_DISABLED_AMBIENT_PREFIX_'])
assert.equal(config.server.host, '127.0.0.1')
assert.equal(config.server.port, 3011)
assert.equal(config.server.strictPort, true)
assert.deepEqual(config.server.proxy, {'/api': {target:'http://127.0.0.1:52962',changeOrigin:true}})
assert.equal(config.plugins.length, 4)
assert.equal(config.plugins[0].name, 'r109-ui-post-budget')
assert.equal(config.plugins[3].name, 'r109-owned-health')
console.log(JSON.stringify({result:'PASS', node:process.version, mode:'MODULE_IMPORT_ONLY_NO_SERVER', pluginGroups:4}))
