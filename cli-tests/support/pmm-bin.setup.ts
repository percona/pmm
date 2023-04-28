import { test as setup } from '@playwright/test';
import * as cli from '@helpers/cliHelper';

const oldImage = 'percona/pmm-server:2.32.0';
const newImage = 'percona/pmm-server:2.33.0';

setup.describe.configure({ mode: 'parallel' });
setup(`pull ${newImage}`, async () => {
  process.env.server_image = newImage;
  const output = await cli.exec(`docker pull ${newImage}`);
  await output.assertSuccess();
});

setup(`pull ${oldImage}`, async () => {
  process.env.server_image_old = oldImage;
  const output = await cli.exec(`docker pull ${oldImage}`);
  await output.assertSuccess();
});
