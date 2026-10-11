async page => {
  const fr = () => page.frames().find(f => f.url().includes('/graph/') && !f.url().includes('/pmm-ui/'));
  const log = {};
  for (const w of [400, 768]) {
    await page.setViewportSize({ width: w, height: 900 });
    await page.goto('https://pmm.kubine.cc/graph/add-instance/mysql');
    await page.waitForTimeout(8000);
    // collapse nav: the « button in the shell
    const btn = page.locator('button[aria-label*="ollapse" i], button:has(svg[data-testid*="KeyboardDoubleArrowLeft"])').first();
    await btn.click({ timeout: 3000 }).catch(e => { log['nav' + w] = 'no collapse button found'; });
    await page.waitForTimeout(1200);
    await page.screenshot({ path: `adv/w${w}-mysql-navclosed.png` });
    log['scroll' + w] = await fr().evaluate(() => [document.documentElement.scrollWidth, document.documentElement.clientWidth]);
  }
  // below-the-fold error
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto('https://pmm.kubine.cc/graph/add-instance/mysql');
  await page.waitForTimeout(8000);
  const f = fr();
  await f.fill('input[name=address]', 'db-prod-1.internal');
  await f.fill('input[name=username]', 'pmm');
  await f.fill('input[name=password]', 'pmm');
  await f.fill('textarea[name=custom_labels]', 'team payments');
  await f.evaluate(() => window.scrollTo(0, 0));
  await f.evaluate(() => document.querySelector('.scrollbar-view, main')?.scrollTo?.(0, 0));
  await f.getByRole('button', { name: 'Add service' }).click();
  await page.waitForTimeout(3000);
  await page.screenshot({ path: 'adv/fold-error.png' });
  log.foldErrorVisible = await f.evaluate(() => { const e = document.querySelector('[data-testid="custom_labels-field-error-message"]'); if (!e) return 'no error el'; const r = e.getBoundingClientRect(); return { text: e.innerText, top: Math.round(r.top), inView: r.top >= 0 && r.bottom <= window.innerHeight }; });
  return log;
}
