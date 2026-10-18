// Root independent installation health test; no application UI or external URL.
const fs = require('node:fs');
const path = require('node:path');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const root = process.cwd();
const out = path.join(root, 'docs/reviews/probes/r103_root');
process.env.PLAYWRIGHT_BROWSERS_PATH = path.join(process.env.LOCALAPPDATA, 'ms-playwright');
const driver = path.join(process.env.LOCALAPPDATA, 'ms-playwright-go', '1.57.0', 'package');
const pkg = require(path.join(driver, 'package.json'));
assert.equal(pkg.name, 'playwright-core'); assert.equal(pkg.version, '1.57.0');
const {chromium} = require(driver);
(async () => {
 let browser;
 try {
  browser = await chromium.launch({headless:true, timeout:30000});
  const context = await browser.newContext({viewport:{width:960,height:540}});
  const page = await context.newPage();
  await page.goto('about:blank');
  await page.setContent("<html><head><title>R103 driver health</title></head><body><h1>Playwright 1.57.0</h1><p>Local driver health check only</p><button onclick=\"this.textContent='Control OK'\">Check control</button></body></html>");
  assert.equal(await page.title(), 'R103 driver health');
  await page.getByRole('button', {name:'Check control', exact:true}).click();
  assert.equal(await page.getByRole('button').innerText(), 'Control OK');
  const image = path.join(out, 'driver_health.png');
  await page.screenshot({path:image, fullPage:true});
  const data=fs.readFileSync(image); assert.ok(data.length>1000);
  fs.writeFileSync(path.join(out, 'driver_health_result.json'), JSON.stringify({
   observedAt:new Date().toISOString(),result:'PASS_STANDALONE_DRIVER_ONLY',driverVersion:pkg.version,
   browserVersion:browser.version(),executable:chromium.executablePath(),url:page.url(),
   controlClickVerified:true,screenshot:'docs/reviews/probes/r103_root/driver_health.png',
   screenshotSha256:crypto.createHash('sha256').update(data).digest('hex'),
   integratedAntigravityTool:'NOT_OBSERVED',R101UI:'NOT_RUN',providerCalls:0
  }, null, 2)+'\n');
  console.log('Standalone exact-version driver launch/control/capture PASS; integrated tool and R101 UI NOT_RUN');
 } finally { if(browser) await browser.close(); }
})().catch(e=>{ console.error(e.stack || String(e)); process.exitCode=1; });
