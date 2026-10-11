async page => {
  const fr = () => page.frames().find(f => f.url().includes('/graph/') && !f.url().includes('/pmm-ui/'));
  const AXE = await (await page.request.get('https://cdnjs.cloudflare.com/ajax/libs/axe-core/4.10.2/axe.min.js')).text();
  const out = {};
  for (const t of ['', 'mysql', 'postgresql', 'rds']) {
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto('https://pmm.kubine.cc/graph/add-instance' + (t ? '/' + t : ''));
    await page.waitForTimeout(8000);
    const f = fr();
    await f.evaluate(AXE);
    const r = await f.evaluate(async () => {
      const res = await window.axe.run(document.querySelector('main') || document, { runOnly: ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'best-practice'] });
      return res.violations.map(v => ({ id: v.id, impact: v.impact, help: v.help, n: v.nodes.length, ex: v.nodes.slice(0, 3).map(n => (n.target.join(' ') + ' :: ' + (n.failureSummary || '').split('\n').slice(1, 2).join(' ')).slice(0, 220)) }));
    });
    out[t || 'select'] = r;
  }
  return out;
}
