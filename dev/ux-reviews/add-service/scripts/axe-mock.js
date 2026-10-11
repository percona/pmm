async page => {
  const AXE = await (await page.request.get('https://cdnjs.cloudflare.com/ajax/libs/axe-core/4.10.2/axe.min.js')).text();
  const out = {};
  for (const mode of ['dark', 'light']) for (const s of ['select', 'mysql-ok', 'mysql-fail', 'postgresql', 'rds', 'success']) {
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto(`http://localhost:5199/?s=${s}&mode=${mode}`);
    await page.waitForTimeout(1200);
    await page.evaluate(AXE);
    out[mode + ':' + s] = await page.evaluate(async () => (await window.axe.run(document, { runOnly: ['wcag2a', 'wcag2aa', 'wcag21aa', 'best-practice'] })).violations.map(v => ({ id: v.id, impact: v.impact, n: v.nodes.length, ex: v.nodes.slice(0, 2).map(n => (n.target.join(' ').slice(-60) + ' :: ' + (n.failureSummary || '').split('\n').slice(1, 2).join(' ')).slice(0, 200)) })));
  }
  await page.goto('http://localhost:5199/?s=mysql&mode=light'); await page.waitForTimeout(1200);
  await page.screenshot({ path: 'adv/mock-light-mysql.png', fullPage: true });
  await page.setViewportSize({ width: 400, height: 860 });
  await page.goto('http://localhost:5199/?s=mysql-ok'); await page.waitForTimeout(1200);
  await page.screenshot({ path: 'adv/mock-w400-mysql.png' });
  out.w400 = await page.evaluate(() => [document.documentElement.scrollWidth, document.documentElement.clientWidth]);
  return out;
}
