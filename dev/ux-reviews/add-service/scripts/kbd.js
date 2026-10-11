async page => {
  const fr = () => page.frames().find(f => f.url().includes('/graph/') && !f.url().includes('/pmm-ui/'));
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto('https://pmm.kubine.cc/graph/add-instance');
  await page.waitForTimeout(8000);
  const f = fr();
  // focus inside frame body first
  await f.locator('body').click({ position: { x: 5, y: 5 } }).catch(() => {});
  const steps = [];
  for (let i = 0; i < 25; i++) {
    await page.keyboard.press('Tab');
    await page.waitForTimeout(150);
    const d = await f.evaluate(() => {
      const e = document.activeElement;
      if (!e || e === document.body) return 'body';
      return `${e.tagName.toLowerCase()}${e.getAttribute('role') ? '[role=' + e.getAttribute('role') + ']' : ''} "${(e.getAttribute('aria-label') || e.innerText || e.getAttribute('data-testid') || '').trim().slice(0, 40).replace(/\n/g, ' ')}" testid=${e.getAttribute('data-testid') || ''}`;
    }).catch(e => 'outside-frame');
    steps.push(i + ': ' + d);
  }
  await page.screenshot({ path: 'adv/kbd-picker.png' });
  // are cards focusable at all?
  const cards = await f.evaluate(() => [...document.querySelectorAll('[data-testid$="-instance"]')].map(c => ({ id: c.getAttribute('data-testid'), tag: c.tagName, tabindex: c.getAttribute('tabindex'), role: c.getAttribute('role'), focusables: c.querySelectorAll('button,a,[tabindex]').length })));
  return { steps, cards };
}
