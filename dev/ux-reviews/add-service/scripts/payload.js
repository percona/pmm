async page => {
  const fr = () => page.frames().find(f => f.url().includes('/graph/') && !f.url().includes('/pmm-ui/'));
  let body = null;
  page.on('request', r => { if (r.method() === 'POST' && r.url().includes('/v1/management/services')) body = r.postData(); });
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto('https://pmm.kubine.cc/graph/add-instance/mysql');
  await page.waitForTimeout(8000);
  const f = fr();
  await f.fill('input[name=address]', 'db.invalid.example');
  await f.fill('input[name=username]', 'pmm');
  await f.fill('input[name=password]', 'x');
  await f.fill('input[name=region]', 'eu-west-1');
  await f.fill('input[name=az]', 'eu-west-1a');
  await f.getByRole('button', { name: 'Add service' }).click();
  await page.waitForTimeout(3000);
  const j = JSON.parse(body || '{}');
  const m = j.mysql || j;
  return { topKeys: Object.keys(m).filter(k => /region|az|add_node|node_id|pmm_agent_id|metrics_mode/.test(k)), region: m.region, az: m.az, add_node: m.add_node, pmm_agent_id: m.pmm_agent_id, metrics_mode: m.metrics_mode };
}
