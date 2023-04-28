import { test, expect } from '@playwright/test';
import * as cli from '@helpers/cliHelper';
import PMMRestClient from '@tests/support/types/request';
import { teardown } from '@tests/helpers/containers';

test.describe.configure({ mode: 'parallel' });
test.describe('PMM Server Install tests', async () => {
  const adminPassword = 'admin123';
  const containerName = 'pmm-server-install-test';
  const imageName = 'percona/pmm-server:2.32.0';
  const volumeName = 'pmm-data-install-test';

  test.afterAll(async ({}) => {
    await teardown([`^${containerName}$`], [volumeName]);
  });

  test('"pmm server docker install" respects relevant flags', async ({ }) => {
    // TODO: add "docker pull" test images as tests configuration to speed up tests
    let output = await cli.exec(`
      pmm server docker install 
        --json
        --admin-password=${adminPassword}
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

    // verify --http-listen-port
    await (new PMMRestClient('admin', adminPassword, 1080)).works();

    // verify --https-listen-port
    const client = new PMMRestClient('admin', adminPassword, 1443, {
      baseURL: 'https://localhost:1443',
      ignoreHTTPSErrors: true,
    });
    const resp = await client.doPost('/v1/Settings/Get');

    await expect(resp).toBeOK();
    expect(await resp.json()).toHaveProperty('settings');

    // verify --container-name
    output = await cli.exec('docker ps --format="{{.Names}}"');
    expect(output.getStdOutLines()).toContainEqual(containerName);

    // verify --volume-name
    output = await cli.exec('docker volume ls --format="{{.Name}}"');
    await output.outContains(volumeName);

    // verify --docker-image
    output = await cli.exec('docker ps --format="{{.Names}} {{.Image}}"');
    await output.outContains(`${containerName} ${imageName}`);
  });
});
