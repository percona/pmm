import { test } from '@playwright/test';
import * as cli from '@helpers/cliHelper';

test.describe('PMM Server Install Docker specific tests', async () => {
  const expectedErrorMessage = 'DockerNoAccess: docker is either not running or this user has no access to Docker. Try running as root';

  test('PMM-T1615 "pmm server docker install" flag --skip-docker-install is respected @pmm-cli', async ({ }) => {
    const output = await cli.exec('pmm server docker install --skip-docker-install');
    await output.exitCodeEquals(1);
    await output.outContains(expectedErrorMessage);
  });

  // next tests in chain to keep docker working after tests on local run:
  // PMM-T1574 Verifying server will not install if user does not have privileges to use docker
  // change privileges for test

  // PMM-T1569 CLI installation of server using command pmm (docker not installed)
  // remove docker and run pmm 'pmm server docker install'
});
