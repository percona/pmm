import { test } from '@playwright/test';
import * as cli from '@helpers/cliHelper';

test.describe('PMM Server Install Docker specific tests', async () => {
  const expectedErrorMessage = 'DockerNoAccess: docker is either not running or this user has no access to Docker. Try running as root';

  test('PMM-T1615 "pmm server docker install" flag --skip-docker-install is respected @pmm-cli', async ({ }) => {
    const output = await cli.exec('pmm server docker install --skip-docker-install');
    await output.exitCodeEquals(1);
    await output.outContains(expectedErrorMessage);
  });
});
