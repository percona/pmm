// usage: node capture.js <url-path-and-query> [out.json]; env CAP_USER, CAP_PASS, CAP_NOANNO=1
const { chromium } = require('/usr/lib/node_modules/@playwright/cli/node_modules/playwright-core');
const BASE = 'https://pmmqa-pmm-15663-bench.tp.int.percona.com';
const [path, out] = process.argv.slice(2);
const isData = (u) => u.includes('/api/ds/query') || u.includes('/resources/') || u.includes('/api/annotations');
(async () => {
  const browser = await chromium.launch({ executablePath: '/opt/pw-browsers/chromium-1246/chrome-linux64/chrome' });
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1600, height: 1000 } });
  const page = await ctx.newPage();
  const login = await page.request.post(`${BASE}/graph/login`, { data: { user: process.env.CAP_USER || 'admin', password: process.env.CAP_PASS || 'Bench-15663' } });
  if (!login.ok()) throw new Error('login ' + login.status());
  const pending = new Map(); const reqs = []; const anno = []; const failed = []; let last = Date.now();
  const isAnno = (r) => r.url().includes('/api/annotations') || (r.url().includes('/api/ds/query') && (r.postData() || '').includes('"refId":"Anno"'));
  if (process.env.CAP_NOANNO) await page.route('**/*', (route) => (isAnno(route.request()) ? route.abort() : route.continue()));
  page.on('request', (r) => { if (isData(r.url())) { pending.set(r, Date.now()); last = Date.now(); } });
  const done = async (r, ok) => {
    if (!pending.has(r)) return;
    const t0 = pending.get(r); pending.delete(r); last = Date.now();
    const rec = { url: r.url().slice(0, 200), urlLen: r.url().length, method: r.method(), body: (r.postData() || '').length, ms: Date.now() - t0, ok };
    reqs.push(rec);
    if (!ok) failed.push(rec);
    if (ok && isAnno(r)) {
      try {
        const resp = await r.response(); const st = resp.status(); const j = await resp.json();
        anno.push({ ...rec, status: st, texts: annoTexts(r.url(), j), sql: sqlOf(r.postData()) });
      } catch (e) { anno.push({ ...rec, error: String(e) }); }
    }
  };
  page.on('requestfinished', async (r) => { const resp = await r.response(); await done(r, resp && resp.status() < 400); });
  page.on('requestfailed', (r) => done(r, false));
  const t0 = Date.now();
  await page.goto(`${BASE}${path}`, { waitUntil: 'domcontentloaded', timeout: 120000 });
  while (Date.now() - t0 < 240000) {
    await page.waitForTimeout(500);
    if (pending.size === 0 && Date.now() - last > 3000 && Date.now() - t0 > 5000 && (reqs.length > 0 || Date.now() - t0 > 60000)) break;
  }
  const loadMs = last - t0;
  const errors = await page.$$eval('[data-testid*="Panel status error"], [aria-label*="Panel status error"]', (e) => e.length).catch(() => -1);
  const res = { path, loadMs, nReq: reqs.length, failed, maxBody: Math.max(0, ...reqs.map((r) => r.body)), maxUrl: Math.max(0, ...reqs.map((r) => r.urlLen)),
    panelErrors: errors, anno: anno.map((a) => ({ url: a.url, method: a.method, body: a.body, urlLen: a.urlLen, ms: a.ms, status: a.status, n: a.texts ? a.texts.length : null, error: a.error, sqlLen: a.sql ? a.sql.length : 0 })),
    annoTexts: [...new Set(anno.flatMap((a) => a.texts || []))].sort(), annoSql: anno.map((a) => a.sql).filter(Boolean) };
  const s = JSON.stringify(res, null, 1);
  if (out) require('fs').writeFileSync(out, s);
  console.log(JSON.stringify({ path, loadMs, nReq: res.nReq, failed: failed.length, maxBody: res.maxBody, maxUrl: res.maxUrl, panelErrors: errors, annoN: res.annoTexts.length, anno: res.anno.map((a) => [a.method, a.ms, a.status, a.n, a.body || a.urlLen, a.error]) }));
  await browser.close();
})().catch((e) => { console.error(e); process.exit(1); });

function sqlOf(body) { try { return JSON.parse(body).queries.map((q) => q.rawSql || q.expr).join('\n'); } catch { return null; } }
function annoTexts(url, j) {
  if (url.includes('/api/annotations')) return j.map((a) => a.text);
  const texts = [];
  for (const res of Object.values(j.results || {})) {
    if (res.error) texts.push('ERROR:' + res.error);
    for (const f of res.frames || []) {
      const fields = f.schema.fields; const vals = f.data.values;
      const ti = fields.findIndex((x) => x.name === 'text');
      if (ti >= 0) { texts.push(...vals[ti]); continue; }
      const lab = fields[1] && fields[1].labels; if (lab && lab.text && vals[1].some((v) => v !== null && v > 0)) texts.push(lab.text);
    }
  }
  return texts;
}
