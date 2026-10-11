async page => {
  const fr = () => page.frames().find(f => f.url().includes('/graph/') && !f.url().includes('/pmm-ui/'));
  const go = async (path, w = 1440, h = 900) => { await page.setViewportSize({ width: w, height: h }); await page.goto('https://pmm.kubine.cc/graph/' + path); await page.waitForTimeout(8000); return fr(); };
  const log = {};
  // narrow viewports
  await go('add-instance', 768, 1000); await page.screenshot({ path: 'adv/w768-picker.png' });
  await go('add-instance/mysql', 768, 1000); await page.screenshot({ path: 'adv/w768-mysql.png' });
  await go('add-instance/mysql', 400, 860); await page.screenshot({ path: 'adv/w400-mysql.png' });
  log.w400scroll = await fr().evaluate(() => [document.documentElement.scrollWidth, document.documentElement.clientWidth]);
  // light theme
  let f = await go('add-instance/mysql?theme=light');
  await page.screenshot({ path: 'adv/light-mysql.png' });
  // data loss: fill then browser back, forward
  f = await go('add-instance/mysql');
  await f.fill('input[name=address]', 'db-prod-1.internal');
  await f.fill('input[name=username]', 'pmm');
  await f.fill('input[name=environment]', 'prod');
  await page.goBack(); await page.waitForTimeout(3000);
  log.afterBack = page.url();
  await page.screenshot({ path: 'adv/back.png' });
  await page.goForward(); await page.waitForTimeout(5000);
  f = fr();
  log.afterForward = { url: page.url(), address: await f.inputValue('input[name=address]').catch(e => 'n/a'), env: await f.inputValue('input[name=environment]').catch(() => 'n/a') };
  // cancel on step 2 with data
  await f.fill('input[name=address]', 'db-prod-1.internal').catch(() => {});
  await f.getByRole('button', { name: 'Cancel' }).click();
  await page.waitForTimeout(3000);
  log.afterCancel = page.url();
  await page.screenshot({ path: 'adv/cancel.png' });
  // refresh mid-form
  f = await go('add-instance/postgresql');
  await f.fill('input[name=address]', 'pg-orders.internal');
  await page.reload(); await page.waitForTimeout(8000);
  log.afterReload = await fr().inputValue('input[name=address]').catch(() => 'n/a');
  // invalid inputs
  f = await go('add-instance/mysql', 1440, 2400);
  await f.fill('input[name=serviceName]', 'x'.repeat(300));
  await f.fill('input[name=address]', 'http://db prod:3306/');
  await f.fill('input[name=port]', '99999');
  await f.fill('input[name=username]', 'pmm');
  await f.fill('input[name=password]', 'pmm');
  await f.fill('input[name=connection_timeout]', '5 minutes');
  await f.fill('input[name=maxQueryLength]', 'abc');
  await f.fill('textarea[name=custom_labels]', 'team payments\nfoo:bar:baz');
  await f.getByRole('button', { name: 'Add service' }).click();
  await page.waitForTimeout(4000);
  await page.screenshot({ path: 'adv/invalid.png' });
  log.invalidErrors = await f.evaluate(() => [...document.querySelectorAll('[data-testid$="error-message"]')].map(e => e.getAttribute('data-testid') + ': ' + e.innerText).filter(s => !s.endsWith(': ')));
  // deep links
  f = await go('add-instance/foo'); await page.screenshot({ path: 'adv/deeplink-foo.png' });
  log.foo = await f.evaluate(() => document.querySelector('h3')?.innerText || document.body.innerText.slice(0, 200));
  return log;
}
