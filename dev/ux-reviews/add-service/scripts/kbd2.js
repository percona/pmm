async page => {
  const fr = () => page.frames().find(f => f.url().includes('/graph/') && !f.url().includes('/pmm-ui/'));
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto('https://pmm.kubine.cc/graph/add-instance/mysql');
  await page.waitForTimeout(8000);
  const f = fr();
  await f.locator('input[name=serviceName]').focus();
  const steps = [];
  for (let i = 0; i < 40; i++) {
    const d = await f.evaluate(() => {
      const e = document.activeElement;
      if (!e || e === document.body) return 'body';
      const lab = e.getAttribute('name') || e.getAttribute('aria-label') || (e.innerText || '').trim().slice(0, 30) || e.getAttribute('data-testid') || '';
      return `${e.tagName.toLowerCase()}${e.type ? ':' + e.type : ''} ${lab.replace(/\n/g, ' ')}`;
    });
    steps.push(i + ' ' + d);
    if (i === 3) await page.screenshot({ path: 'adv/kbd-mysql-focus.png' });
    await page.keyboard.press('Tab');
    await page.waitForTimeout(120);
  }
  // tooltip icons: focusable?
  const icons = await f.evaluate(() => [...document.querySelectorAll('label svg, label [class*="icon"], [data-testid*="tooltip"]')].slice(0, 5).map(i => ({ tag: i.tagName, tabindex: i.closest('[tabindex]')?.getAttribute('tabindex') ?? null, aria: i.getAttribute('aria-label') })));
  // hover a tooltip
  const info = f.locator('label:has-text("Connection timeout") svg').first();
  await info.hover().catch(() => {});
  await page.waitForTimeout(800);
  await page.screenshot({ path: 'adv/tooltip-hover.png' });
  return { steps, icons };
}
