async page => {
  const fr = () => page.frames().find(f => f.url().includes('/graph/') && !f.url().includes('/pmm-ui/'));
  await page.setViewportSize({ width: 400, height: 860 });
  await page.goto('https://pmm.kubine.cc/graph/add-instance/mysql');
  await page.waitForTimeout(8000);
  const btns = await page.evaluate(() => [...document.querySelectorAll('button')].slice(0, 12).map(b => (b.getAttribute('aria-label') || b.innerText || b.getAttribute('data-testid') || '').slice(0, 40)));
  // click the first button near top-left of the drawer header (the « icon)
  const box = await page.evaluate(() => { const b = [...document.querySelectorAll('button')].find(b => { const r = b.getBoundingClientRect(); return r.top < 60 && r.left > 250 && r.left < 320; }); if (!b) return null; b.click(); return b.outerHTML.slice(0, 150); });
  await page.waitForTimeout(1500);
  await page.screenshot({ path: 'adv/w400-navclosed.png' });
  const f = fr();
  await f.evaluate(() => document.querySelector('main')?.scrollTo?.(0, 0));
  return { btns, box, scroll: await f.evaluate(() => [document.documentElement.scrollWidth, document.documentElement.clientWidth]) };
}
