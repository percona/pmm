import { expect } from '@playwright/test';
import PmmRestClient from '@support/types/PmmRestClient';
import * as cli from './cliHelper';

/**
 * Encapsulates all checks for running "PMM Server" container in docker.
 * All verifications are taken based on specified parameters.
 * Object properties are optional and verifies each "pmm docker command" flag:
 * {
 *   containerName: "verifies --container-name",
 *   imageName: "verifies --docker-image, requires containerName",
 *   volumeName: "verifies --volume-name",
 *   httpPort: "verifies --http-listen-port, requires adminPassword",
 *   httpsPort: "verifies --https-listen-port, requires adminPassword",
 *   adminPassword: "verifies --admin-password, embedded into ports check"
 * }
 *
 * @param   checks  Object with checks to execute:
 */
export const verifyPmmServerProperties = async (checks: {
  containerName?: string,
  imageName?: string,
  volumeName?: string,
  httpPort?: number,
  httpsPort?: number,
  adminPassword?: string }) => {
  // verify --container-name
  if (checks.containerName !== undefined) {
    await (await cli.exec('docker ps --format="{{.Names}}"')).outHasLine(checks.containerName);
  }

  // verify --volume-name
  if (checks.volumeName !== undefined) {
    await (await cli.exec('docker volume ls --format="{{.Name}}"')).outHasLine(checks.volumeName);
  }

  // verify --docker-image
  if (checks.imageName !== undefined && checks.containerName !== undefined) {
    await (await cli.exec('docker ps --format="{{.Names}} {{.Image}}"'))
      .outHasLine(`${checks.containerName} ${checks.imageName}`);
  }

  // verify --http-listen-port
  if (checks.httpPort !== undefined && checks.adminPassword !== undefined) {
    const httpClient = new PmmRestClient('admin', checks.adminPassword, checks.httpPort);

    await expect(async () => {
      const resp = await httpClient.post('/v1/settings/Get', {});
      await expect(resp, `http ${checks.httpPort} port and password should work`).toBeOK();
      expect(await resp.json(), 'response body should have "settings"').toHaveProperty('settings');
    }).toPass({
      // Probe, wait 1s, probe, wait 2s, probe, wait 2s, probe, wait 2s, probe, ....
      intervals: [1_000, 2_000, 2_000],
      timeout: 60_000,
    });
  }

  // verify --https-listen-port
  if (checks.httpsPort !== undefined && checks.adminPassword !== undefined) {
    const httpsClient = new PmmRestClient('admin', checks.adminPassword, checks.httpsPort, 'https');

    await expect(async () => {
      const resp = await httpsClient.post('/v1/settings/Get', {});
      await expect(resp, `https ${checks.httpsPort} port and password should work`).toBeOK();
      expect(await resp.json(), 'response body should have "settings"').toHaveProperty('settings');
    }).toPass({
      // Probe, wait 1s, probe, wait 1s, probe, wait 2s, probe, wait 2s, probe, ....
      intervals: [1_000, 2_000, 2_000],
      timeout: 30_000,
    });
  }
};
