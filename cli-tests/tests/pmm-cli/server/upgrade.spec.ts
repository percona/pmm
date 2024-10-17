import { expect, test } from '@playwright/test';
import * as cli from '@helpers/cliHelper';
import { verifyPmmServerProperties } from '@helpers/customAssertions';
import { runOldPmmServer } from '@helpers/containers';

const defaultAdminPassword = 'admin';
const defaultServImage = 'perconalab/pmm-server:3-dev-latest';
const defaultVolumeName = 'pmm-data';

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
      containerName: output.generateContainerNameFromLogs(),
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
      containerName: output.generateContainerNameFromLogs(newContainerPrefix),
      imageName: process.env.server_image,
      volumeName,
      httpPort,
      httpsPort,
      adminPassword: defaultAdminPassword,
    });
  });

  test('PMM-T1680 "pmm server docker upgrade" upgrades non-CLI containers', async () => {
    const oldContainerName = 'pmm-server-non-cli';
    const volumeName = 'pmm-data-non-cli';
    const httpPort = 4079;
    const httpsPort = 4444;
    await runOldPmmServer(httpPort, httpsPort, volumeName, oldContainerName);

    await cli.execSilent('sleep 1'); // to avoid same name
    const output = await cli.exec(`
      pmm server docker upgrade -y 
        --json
        --container-id=${oldContainerName}`);
    await output.assertSuccess();

    expect(output.stderr, 'stderr should contain "Starting PMM Server"').toContain('Starting PMM Server');

    await verifyPmmServerProperties({
      containerName: output.generateContainerNameFromLogs(),
      imageName: defaultServImage,
      volumeName,
      httpPort,
      httpsPort,
      adminPassword: defaultAdminPassword,
    });
  });

  test('PMM-T1682 "pmm server docker upgrade" gives warning for non-CLI containers', async () => {
    const oldContainerName = 'pmm-server-non-cli-warn';
    const newContainerPrefix = 'pmm-server-warn';
    const volumeName = 'pmm-data-non-cli-warn';
    const httpPort = 4078;
    const httpsPort = 4445;
    await runOldPmmServer(httpPort, httpsPort, volumeName, oldContainerName);

    const output = await cli.exec(`
      pmm server docker upgrade -y
        --new-container-name-prefix=${newContainerPrefix}
        --container-id=${oldContainerName}`);
    await output.assertSuccess();

    await output.outContainsMany([
      `PMM Server in the container "${oldContainerName}" was not installed via pmm cli.`,
      'We will attempt to upgrade the container and perform the following actions:',
      `- Stop the container "${oldContainerName}"`,
      `- Back up all volumes in "${oldContainerName}"`,
      `- Mount all volumes from "${oldContainerName}" in the new container`,
      `- Share the same network ports as in "${oldContainerName}"`,
      `The container "${oldContainerName}" will NOT be removed. You can remove it manually later, if needed.`,
    ]);
  });

  test('PMM-T1685 "pmm server docker upgrade" flags are respected for non-CLI containers', async () => {
    const oldContainerName = 'pmm-server-non-cli-flags';
    const newContainerName = 'pmm-server-non-cli-flags-new';
    const volumeName = 'pmm-data-non-cli-flags';
    const httpPort = 4081;
    const httpsPort = 4446;
    await runOldPmmServer(httpPort, httpsPort, volumeName, oldContainerName);

    const output = await cli.exec(`
      pmm server docker upgrade -y 
        --json
        --docker-image=${process.env.server_image} 
        --new-container-name=${newContainerName} 
        --container-id=${oldContainerName}`);
    await output.assertSuccess();
    expect(output.stderr, 'stderr should contain "Starting PMM Server"').toContain('Starting PMM Server');

    await verifyPmmServerProperties({
      containerName: newContainerName,
      imageName: process.env.server_image,
      volumeName,
      httpPort,
      httpsPort,
      adminPassword: defaultAdminPassword,
    });
  });

  test('PMM-T1687 "pmm server docker upgrade" respects container name prefix for non-CLI containers', async () => {
    const oldContainerName = 'pmm-server-non-cli-prefix';
    const newContainerPrefix = 'pmm-server-prefix';
    const volumeName = 'pmm-data-non-cli-prefix';
    const httpPort = 4083;
    const httpsPort = 4447;
    await runOldPmmServer(httpPort, httpsPort, volumeName, oldContainerName);

    const output = await cli.exec(`
      pmm server docker upgrade -y
        --json
        --new-container-name-prefix=${newContainerPrefix}
        --container-id=${oldContainerName}`);
    await output.assertSuccess();
    // TODO: include json format warning verification after PMM-12035 is done
    // await output.outContainsMany([
    //   `PMM Server in the container "${oldContainerName}" was not installed via pmm cli.`,
    //   'We will attempt to upgrade the container and perform the following actions:',
    //   `- Stop the container "${oldContainerName}"`,
    //   `- Back up all volumes in "${oldContainerName}"`,
    //   `- Mount all volumes from "${oldContainerName}" in the new container`,
    //   `- Share the same network ports as in "${oldContainerName}"`,
    //   `The container "${oldContainerName}" will NOT be removed. You can remove it manually later, if needed.`,
    // ]);
    await verifyPmmServerProperties({
      containerName: output.generateContainerNameFromLogs(newContainerPrefix),
      imageName: defaultServImage,
    });
  });
});
