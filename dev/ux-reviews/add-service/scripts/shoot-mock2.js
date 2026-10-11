async page => {
  const AXE = await (await page.request.get('https://cdnjs.cloudflare.com/ajax/libs/axe-core/4.10.2/axe.min.js')).text();
  const sizes = {}, axe = {};
  for (const s of ['select', 'mysql', 'mysql-ok', 'mysql-fail', 'postgresql', 'rds', 'success']) {
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto('http://localhost:5199/?s=' + s);
    await page.waitForTimeout(1300);
    sizes[s] = await page.evaluate(() => document.documentElement.scrollHeight);
    await page.screenshot({ path: `adv/v2-mock-${s}.png`, fullPage: true });
    for (const mode of ['dark', 'light']) {
      await page.goto(`http://localhost:5199/?s=${s}&mode=${mode}`); await page.waitForTimeout(900);
      await page.evaluate(AXE);
      axe[mode + ':' + s] = await page.evaluate(async () => (await window.axe.run(document, { runOnly: ['wcag2a', 'wcag2aa', 'wcag21aa'] })).violations.map(v => `${v.impact} ${v.id} x${v.nodes.length} :: ${(v.nodes[0].target.join(' ')).slice(-50)} :: ${(v.nodes[0].failureSummary || '').split('\n').slice(1,2).join('').slice(0,140)}`));
    }
  }
  await page.setViewportSize({ width: 400, height: 860 });
  await page.goto('http://localhost:5199/?s=mysql-ok'); await page.waitForTimeout(1200);
  await page.screenshot({ path: 'adv/v2-mock-w400.png' });
  const w400 = await page.evaluate(() => [document.documentElement.scrollWidth, document.documentElement.clientWidth]);
  return { sizes, w400, axe };
}
