async page => {
  const fr = () => page.frames().find(f => f.url().includes('/graph/') && !f.url().includes('/pmm-ui/'));
  const log = {};
  const svc = await (await page.request.get('https://pmm.kubine.cc/v1/inventory/services')).json().catch(() => ({}));
  const names = Object.values(svc).flat().filter(s => s && s.service_name).map(s => s.service_name);
  log.existing = names.slice(0, 5);
  const posts = [];
  page.on('request', r => { if (r.method() === 'POST' && r.url().includes('/v1/management/services')) posts.push(r.url()); });
  // 1) double submit with unresolvable host
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto('https://pmm.kubine.cc/graph/add-instance/mysql');
  await page.waitForTimeout(8000);
  let f = fr();
  await f.fill('input[name=address]', 'db.invalid.example');
  await f.fill('input[name=username]', 'pmm');
  await f.fill('input[name=password]', 'pmm');
  const b = f.getByRole('button', { name: 'Add service' });
  await b.dblclick();
  await page.waitForTimeout(400);
  log.btnDisabledDuring = await b.isDisabled();
  await page.waitForTimeout(2500);
  log.doublePosts = posts.length;
  await page.screenshot({ path: 'adv/double-submit.png' });
  // 2) duplicate name
  posts.length = 0;
  if (names[0]) {
    await page.goto('https://pmm.kubine.cc/graph/add-instance/postgresql');
    await page.waitForTimeout(8000);
    f = fr();
    await f.fill('input[name=serviceName]', names[0]);
    await f.fill('input[name=address]', 'db.invalid.example');
    await f.fill('input[name=username]', 'pmm');
    await f.fill('input[name=password]', 'pmm');
    await f.getByRole('button', { name: 'Add service' }).click();
    await page.waitForTimeout(2500);
    await page.screenshot({ path: 'adv/duplicate-name.png' });
  }
  // 3) wrong password against PMM Server's own PostgreSQL
  await page.goto('https://pmm.kubine.cc/graph/add-instance/postgresql');
  await page.waitForTimeout(8000);
  f = fr();
  await f.fill('input[name=serviceName]', 'adv-review-auth-test');
  await f.fill('input[name=address]', '127.0.0.1');
  await f.fill('input[name=username]', 'pmm-adv-review');
  await f.fill('input[name=password]', 'definitely-wrong');
  await f.getByRole('button', { name: 'Add service' }).click();
  await page.waitForTimeout(3000);
  await page.screenshot({ path: 'adv/auth-fail.png' });
  log.toasts = await f.evaluate(() => [...document.querySelectorAll('[role=status], [data-testid*="alert"], .page-alert-list *')].map(e => e.innerText).filter(Boolean).slice(0, 4)).catch(() => []);
  // confirm nothing was created
  const after = await (await page.request.get('https://pmm.kubine.cc/v1/inventory/services')).json().catch(() => ({}));
  log.createdTest = Object.values(after).flat().filter(s => s && /adv-review|invalid/.test(s.service_name || '')).map(s => s.service_name);
  return log;
}
