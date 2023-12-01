import { test, expect } from '@playwright/test';
import * as cli from '@helpers/cliHelper';
import { verifyPmmServerProperties } from '@helpers/customAssertions';

test.describe.configure({ mode: 'parallel' });

test.describe('PMM Server Install tests', async () => {
  const defaultAdminPassword = 'admin';
  const defaultServImage = 'perconalab/pmm-server:3-dev-latest';
  const defaultContainerName = 'pmm-server';
  const defaultVolumeName = 'pmm-data';
  const adminPassword = 'admin123';

  test('PMM-T1570 "pmm server docker install" works with no flags @pmm-cli', async ({ }) => {
    const output = await cli.exec('pmm server docker install');
    await output.assertSuccess();
    expect(output.stderr, 'stderr should contain \'Starting PMM Server\'').toContain('Starting PMM Server');

    await verifyPmmServerProperties({
      containerName: defaultContainerName,
      imageName: defaultServImage,
      volumeName: defaultVolumeName,
      httpPort: 80,
      httpsPort: 443,
      adminPassword: defaultAdminPassword,
    });
  });

  test('PMM-T1610 "pmm server docker install" respects relevant flags @pmm-cli', async ({ }) => {
    const containerName = 'pmm-server-install-test';
    const volumeName = 'pmm-data-install-test';
    // TODO: add getHttpPort() getHttpsPort() methods to remove manual attention.
    const httpPort = 1080;
    const httpsPort = 1443;

    const output = await cli.exec(`
      pmm server docker install 
        --admin-password=${adminPassword}
        --docker-image="${process.env.server_image}"
        --https-listen-port=${httpsPort}
        --http-listen-port=${httpPort}
        --container-name=${containerName}
        --volume-name=${volumeName}`);
    await output.assertSuccess();
    expect(output.stderr, 'stderr should contain "Starting PMM Server"').toContain('Starting PMM Server');

    await verifyPmmServerProperties({
      containerName,
      imageName: process.env.server_image,
      volumeName,
      httpPort,
      httpsPort,
      adminPassword,
    });
  });

  test('PMM-T1660 "pmm server docker install" shows error for short password @pmm-cli', async ({ }) => {
    const containerName = 'pmm-server-short-pass';
    const volumeName = 'pmm-data-short-pass';
    const httpPort = 1081;
    const httpsPort = 1444;

    const output = await cli.exec(`
      pmm server docker install
        --admin-password="test"
        --https-listen-port=${httpsPort}
        --http-listen-port=${httpPort}
        --container-name=${containerName}
        --volume-name=${volumeName}`);
    await output.assertSuccess();
    await output.outContains('Error: ✗ new password is too short');
    await output.errContainsMany([
      'Starting PMM Server',
      'Password change exit code: 1',
      'Password change failed. Use the default password "admin"',
    ]);

    await verifyPmmServerProperties({
      httpPort,
      httpsPort,
      adminPassword: defaultAdminPassword,
    });
  });

  test('PMM-T1575 "pmm server docker install" exits if volume already exists @pmm-cli', async ({ }) => {
    const volumeName = 'pmm-data-exists';
    await (await cli.exec(`docker volume create ${volumeName}`)).assertSuccess();
    const output = await cli.exec(`
      pmm server docker install 
        --volume-name=${volumeName}`);
    await output.exitCodeEquals(1);
    await output.outContains(`VolumeExists: docker volume with name "${volumeName}" already exists`);
  });

  test('PMM-T1576 "pmm server docker install" exits if docker container is already present @pmm-cli', async ({ }) => {
    const containerName = 'pmm-server-exists';
    const httpsPort = 1445;
    const httpPort = 1082;
    await (await cli.exec(`
      sudo pmm server docker install 
        --https-listen-port=${httpsPort}
        --http-listen-port=${httpPort}
        --container-name=${containerName}
        --volume-name=pmm-data-123`)).assertSuccess();
    const output = await cli.exec(`
      pmm server docker install
        --container-name=${containerName}
        --volume-name=pmm-data-124`);
    await output.exitCodeEquals(1);
    await output.outContains(`Error response from daemon: Conflict. The container name "/${containerName}" is already in use by container`);
  });

  // TODO: PMM-T1616 scenario requires a review. Why this flag is actually needed?
  test('PMM-T1616 "pmm server docker install" flag --skip-change-password is respected @pmm-cli', async ({ }) => {
    const containerName = 'pmm-server-skip-pass';
    const volumeName = 'pmm-data-skip-pass';
    const httpsPort = 1446;
    const httpPort = 1888;
    const output = await cli.exec(`
      pmm server docker install 
        --skip-change-password
        --https-listen-port=${httpsPort}
        --http-listen-port=${httpPort}
        --container-name=${containerName}
        --volume-name=${volumeName}`);
    await output.assertSuccess();
    expect(output.stderr, 'stderr should contain "Starting PMM Server"').toContain('Starting PMM Server');
    expect(output.stdout, 'Stdout should not contain "Changing password"!').not.toContain('Changing password');

    await verifyPmmServerProperties({
      httpPort,
      httpsPort,
      adminPassword: defaultAdminPassword,
    });
  });

  test('PMM-T1616 "pmm server docker install" flag --skip-change-password is respected with present password change flag'
      + ' @pmm-cli', async ({ }) => {
    const containerName = 'pmm-server-skip-flag';
    const volumeName = 'pmm-data-skip-flag';
    const httpsPort = 1447;
    const httpPort = 1889;
    const output = await cli.exec(`
      pmm server docker install 
        --skip-change-password
        --admin-password=${adminPassword}
        --https-listen-port=${httpsPort}
        --http-listen-port=${httpPort}
        --container-name=${containerName}
        --volume-name=${volumeName}`);
    await output.assertSuccess();
    expect(output.stderr, 'stderr should contain "Starting PMM Server"').toContain('Starting PMM Server');
    expect(output.stdout, 'Stdout should not contain "Changing password"!').not.toContain('Changing password');

    await verifyPmmServerProperties({
      httpPort,
      httpsPort,
      adminPassword: defaultAdminPassword,
    });
  });
});
