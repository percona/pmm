async page => {
  const fr = () => page.frames().find(f => f.url().includes('/graph/') && !f.url().includes('/pmm-ui/'));
  const ver = await (await page.request.get('https://pmm.kubine.cc/v1/server/version')).json().catch(e => String(e));
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto('https://pmm.kubine.cc/graph/add-instance/rds');
  await page.waitForTimeout(12000);
  const f = fr();
  await f.fill('input[name=aws_access_key]', 'AKIAEXAMPLEEXAMPLE1');
  await f.fill('input[name=aws_secret_key]', 'bogus-secret');
  await f.getByRole('button', { name: 'Discover', exact: true }).click();
  const shots = [];
  for (const t of [1500, 4000, 9000]) {
    await page.waitForTimeout(t - (shots.length ? [1500, 4000, 9000][shots.length - 1] : 0));
    await page.screenshot({ path: `adv/rds-bad-${t}.png` });
    shots.push(t);
  }
  return { version: ver.version || ver, full: ver.server?.full_version, shots };
}
