import { test, expect } from '@playwright/test';
import * as cli from '@helpers/cliHelper';
import PMMRestClient from '@tests/support/types/request';
import { teardown } from '@tests/helpers/containers';

test.describe('Install PMM Server respects relevant flags', async () => {
  const adminPw = 'admin123';
  const containerName = 'pmm-server-install-test';
  const imageName = 'percona/pmm-server:2.32.0';
  const volumeName = 'pmm-data-install-test';

  test.beforeAll(async ({}) => {
    const output = await cli.exec(`
      pmm server docker install 
        --json
        --admin-password=${adminPw}
        --docker-image="${imageName}"
        --https-listen-port=1443
        --http-listen-port=1080
        --container-name=${containerName}
        --volume-name=${volumeName}`);
    await output.assertSuccess();

    await expect(async () => {
      expect(output.stderr).toContain('Starting PMM Server');
    }).toPass({
      // Probe, wait 1s, probe, wait 2s, probe, wait 2s, probe, wait 2s, probe, ....
      intervals: [1_000, 2_000, 2_000],
      timeout: 20_000,
    });
  });

  test.afterAll(async ({}) => {
    await teardown([`^${containerName}$`], [volumeName]);
  });

  test('http client', async ({ }) => {
    const client = new PMMRestClient('admin', adminPw, 1080);
    await client.works();
  });

  test('https client', async ({ }) => {
    const client = new PMMRestClient('admin', adminPw, 1443, {
      baseURL: 'https://localhost:1443',
      ignoreHTTPSErrors: true,
    });
    const resp = await client.doPost('/v1/Settings/Get');

    await expect(resp).toBeOK();
    expect(await resp.json()).toHaveProperty('settings');
  });

  test('Container name', async ({ }) => {
    const output = await cli.exec('docker ps --format="{{.Names}}"');
    await output.outContains(containerName);
  });

  test('Volume name', async ({ }) => {
    const output = await cli.exec('docker volume ls --format="{{.Name}}"');
    await output.outContains(volumeName);
  });

  test('Docker image', async ({ }) => {
    const output = await cli.exec('docker ps --format="{{.Names}} {{.Image}}"');
    await output.outContains(`${containerName} ${imageName}`);
  });
});
