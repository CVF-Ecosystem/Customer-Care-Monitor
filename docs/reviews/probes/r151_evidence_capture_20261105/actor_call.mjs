// R150 finite VERIFY: one original native sendCommand per short-lived actor process.
// This file does not start or ensure a daemon and accepts only the fixed order methods.
import {pathToFileURL} from 'node:url';
import path from 'node:path';

const [pkgRoot, sessionId, action, pageIdText] = process.argv.slice(2);
const clientUrl = pathToFileURL(path.join(pkgRoot, 'build/src/daemon/client.js')).href;
const {sendCommand} = await import(clientUrl);
const commands = {
  status: {method: 'status'},
  new_page: {method: 'invoke_tool', tool: 'new_page', args: {url: 'about:blank'}},
  list_pages: {method: 'invoke_tool', tool: 'list_pages', args: {}},
  snapshot_missing: {method: 'invoke_tool', tool: 'take_snapshot', args: {}},
  stop: {method: 'stop'},
};
let command = commands[action];
if (action === 'snapshot_valid') {
  const pageId = Number(pageIdText);
  if (!Number.isInteger(pageId) || pageId < 0) throw new Error('invalid admitted pageId');
  command = {method: 'invoke_tool', tool: 'take_snapshot', args: {pageId}};
}
if (!command || !/^[a-fA-F0-9-]+$/.test(sessionId)) {
  throw new Error('unadmitted actor command or session');
}
const startedAt = new Date().toISOString();
try {
  const response = await sendCommand(command, sessionId, 30000);
  process.stdout.write(JSON.stringify({actor: action, command, response, startedAt,
    terminal: {kind: 'RESOLVED', at: new Date().toISOString()}}));
} catch (error) {
  process.stdout.write(JSON.stringify({actor: action, command, startedAt,
    terminal: {kind: 'REJECTED', message: String(error), at: new Date().toISOString()}}));
  process.exitCode = 1;
}
