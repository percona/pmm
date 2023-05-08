import {expect, test} from '@playwright/test';
import * as cli from '@helpers/cliHelper';

test.describe('PMM Server Install Docker specific tests', async () => {
  const expectedErrorMessage = 'DockerNoAccess: docker is either not running or this user has no access to Docker. Try running as root';

  test('PMM-T1615 "pmm server docker install" flag --skip-docker-install is respected @pmm-cli', async ({ }) => {
    const output = await cli.exec('pmm server docker install --json --skip-docker-install');
    await output.exitCodeEquals(1);
    await output.outContains(expectedErrorMessage);
  });

  // next tests in chain to keep docker working after tests on local run:
  // test('PMM-T1574 "pmm server docker install" displays error if user does not have privileges to use docker @pmm-cli', async ({ }) => {
  //   // change privileges for test
  //   const output = await cli.exec('pmm server docker install --json --skip-docker-install');
  //   await output.exitCodeEquals(1);
  //   await output.outContains(expectedErrorMessage);
  // });

  // test('PMM-T1569 "pmm server docker install" installs server when docker not installed @pmm-cli', async ({ }) => {
  //   // remove docker and run pmm 'pmm server docker install'
  //   const output = await cli.exec('pmm server docker install --json');
  //   await output.assertSuccess();
  //   expect(output.stderr, 'stderr should contain "Starting PMM Server"').toContain('Starting PMM Server');
  //
  //   await verifyPmmServerProperties({
  //     containerName: defaultContainerName,
  //     imageName: defaultServImage,
  //     volumeName: defaultVolumeName,
  //     httpPort: 80,
  //     httpsPort: 443,
  //     adminPassword: defaultAdminPassword,
  //   });
  // });
});
