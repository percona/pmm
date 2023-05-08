import { expect, test } from '@playwright/test';
import * as cli from '@helpers/cliHelper';
import PMMRestClient from '@tests/support/types/request';
import { DateTime } from 'luxon';

const defaultAdminPassword = 'admin';
const defaultServImage = 'percona/pmm-server:2';
const defaultVolumeName = 'pmm-data';

/**
 * Encapsulates composition of auto generated container name.
 * The name is based on container start date and time,
 * which is parsed from specified output lines produced by 'pmm server docker upgrade --json'
 * Designed to parse: {@code (await cli.exec('pmm server docker upgrade --json')).getStdErrLines();}
 *
 * Example:
 * > logs line: {"level":"info","msg":"Starting PMM Server","time":"2023-05-04T12:47:49-04:00"}
 * > returns:   'pmm-server-2023-05-04-12-47-49'
 *
 * @param   logs    shell logs lines array {@link Output#getStdErrLines()}
 * @param   prefix  name prefix to generate format: 'prefix-YYYY-MM-DD-HH-MM-SS'
 * @return          container name {@code string} in format: 'pmm-server-YYYY-MM-DD-HH-MM-SS'
 */
const generateContainerNameFromLogs = (logs: string[], prefix = 'pmm-server'): string => {
  const foundLine = logs.find((item) => item.includes('"Starting PMM Server","time":')).trim();
  expect(foundLine, 'Specified logs should have "Starting PMM Server" with time').not.toBeUndefined();
  type LogLine = { level: string, msg: string, time: string };
  const startDateTime: string = (JSON.parse(foundLine) as LogLine).time;
  return `${prefix}-${DateTime.fromISO(startDateTime).toFormat('yyyy-MM-dd-hh-mm-ss')}`;
};

/**
 * Encapsulates all running "PMM Server" container in docker.
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
const verifyPmmServerProperties = async (checks: {
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
  // TODO: improve error messaging and logs for exapmple for incorrect password
  if (checks.httpPort !== undefined && checks.adminPassword !== undefined) {
    await (new PMMRestClient('admin', checks.adminPassword, checks.httpPort)).works();
  }
  // verify --https-listen-port
  if (checks.httpsPort !== undefined && checks.adminPassword !== undefined) {
    const client = new PMMRestClient('admin', checks.adminPassword, checks.httpsPort, {
      baseURL: `https://localhost:${checks.httpsPort}`,
      ignoreHTTPSErrors: true,
    });
    const resp = await client.doPost('/v1/Settings/Get');

    await expect(resp, 'https port and password should work').toBeOK();
    expect(await resp.json()).toHaveProperty('settings');
  }
};

test.describe.configure({ mode: 'parallel' });
test.describe('pmm-bin: server upgrade tests', async () => {
  const adminPassword = 'admin123';

  test('"pmm server docker upgrade" works with no flags', async () => {
    let output = await cli.exec(`
      pmm server docker install 
        --json
        --docker-image="${process.env.old_server_image}"`);
    await output.assertSuccess();

    output = await cli.exec('pmm server docker upgrade --json');
    await output.assertSuccess();
    expect(output.stderr, 'stderr should contain "Starting PMM Server"').toContain('Starting PMM Server');

    await verifyPmmServerProperties({
      containerName: generateContainerNameFromLogs(output.getStdErrLines()),
      imageName: defaultServImage,
      volumeName: defaultVolumeName,
      httpPort: 80,
      httpsPort: 443,
      adminPassword: defaultAdminPassword,
    });
  });

  test('"pmm server docker upgrade" respects relevant flags', async () => {
    const oldContainerName = 'pmm-server-upgrade-1';
    const newContainerName = 'pmm-server-upgrade-1-new';
    const volumeName = 'pmm-data-upgrade-1';
    const httpPort = 3080;
    const httpsPort = 3443;

    await (await cli.exec(`
      pmm server docker install 
        --json
        --admin-password=${adminPassword}
        --docker-image="${process.env.old_server_image}"
        --http-listen-port=${httpPort}
        --https-listen-port=${httpsPort}
        --container-name=${oldContainerName}
        --volume-name=${volumeName}`)).assertSuccess();

    const output = await cli.exec(`
      pmm server docker upgrade
        --json
        --docker-image="${process.env.server_image}"
        --container-id=${oldContainerName}
        --new-container-name=${newContainerName}`);
    await output.assertSuccess();
    expect(output.stderr, 'stderr should contain "Starting PMM Server"').toContain('Starting PMM Server');

    await verifyPmmServerProperties({
      containerName: newContainerName,
      imageName: process.env.server_image,
      volumeName,
      httpPort,
      httpsPort,
      adminPassword,
    });
  });

  test('"pmm server docker upgrade" respects container name prefix', async () => {
    const oldContainerName = 'pmm-server-upgrade-2';
    const newContainerPrefix = 'pmm-server-upg';
    const volumeName = 'pmm-data-upgrade-2';
    const httpPort = 4080;
    const httpsPort = 4443;

    await (await cli.exec(`
      pmm server docker install 
        --json
        --docker-image="${process.env.old_server_image}"
        --http-listen-port=${httpPort}
        --https-listen-port=${httpsPort}
        --container-name=${oldContainerName}
        --volume-name=${volumeName}`)).assertSuccess();

    const output = await cli.exec(`
      pmm server docker upgrade
        --json
        --docker-image="${process.env.server_image}"
        --container-id=${oldContainerName}
        --new-container-name-prefix=${newContainerPrefix}`);
    await output.assertSuccess();
    expect(output.stderr, 'stderr should contain "Starting PMM Server"').toContain('Starting PMM Server');

    await verifyPmmServerProperties({
      containerName: generateContainerNameFromLogs(output.getStdErrLines(), newContainerPrefix),
      imageName: process.env.server_image,
      volumeName,
      httpPort,
      httpsPort,
      adminPassword: defaultAdminPassword,
    });
  });

  // PMM-T1680 Verify pmm server docker upgrade will upgrade non-CLI containers
  // PMM-T1682 Verify pmm server docker upgrade will give warning for upgrade of non-CLI containers
  // PMM-T1685 Verify CLI command "pmm server docker upgrade" flags are respected for non-CLI server
  // PMM-T1687 Verify pmm server docker upgrade flag "--new-container-name-prefix" non-CLI server
  // PMM-T1702 General functionality test for pmm CLI upgrade of non-cli installed pmm server
});
