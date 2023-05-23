import { expect, test } from '@playwright/test';
import * as cli from '@helpers/cliHelper';

/**
 * Test are chained in the order to keep docker working after both CI/CD and local runs:
 * 1. Uninstall docker from OS -> check error
 * 2. Docker installed by pmm bin
 * 3. Remove $USER's privileges to use docker -> check error
 * 4. Restore $USER's privileges
 */
test.describe('PMM Server Install Docker specific tests', async () => {
  const expectedErrorMessage = 'DockerNoAccess: docker is either not running or this user has no access to Docker. Try running as root';

  test.afterAll(async () => {
    const output = await cli.exec('sudo usermod -a -G docker $USER');
  });

  test.skip('PMM-T1615 "pmm server docker install" flag --skip-docker-install is respected @pmm-cli', async ({ }) => {
    // Remove docker for test
    // dpkg -l | grep -i docker
    // sudo apt-get purge -y docker-engine docker docker.io docker-ce docker-ce-cli docker-compose-plugin || true
    // sudo apt-get autoremove -y --purge docker-engine docker docker.io docker-ce docker-compose-plugin
    const output = await cli.exec('pmm server docker install --json --skip-docker-install');
    await output.exitCodeEquals(1);
    await output.outContains(expectedErrorMessage);
  });

  test.skip('PMM-T1569 "pmm server docker install" installs server when docker not installed @pmm-cli', async ({ }) => {
    // remove docker and run pmm 'pmm server docker install'
    const output = await cli.exec('pmm server docker install --json');
    await output.assertSuccess();
    expect(output.stderr, 'stderr should contain "Starting PMM Server"').toContain('Starting PMM Server');
  });

  test.skip('PMM-T1574 "pmm server docker install" displays error if user does not have privileges to use docker @pmm-cli', async ({ }) => {
    // change privileges for test
    await (await cli.exec('sudo gpasswd -d $USER docker')).assertSuccess();
    await (await cli.exec('exec newgrp $USER')).assertSuccess();

    const output = await cli.exec('pmm server docker install --skip-docker-install');
    await output.exitCodeEquals(1);
    await output.outContains(expectedErrorMessage);
  });
});
