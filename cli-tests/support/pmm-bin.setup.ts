import { test as setup } from '@playwright/test';

const oldImage = 'percona/pmm-server:2.32.0';
const newImage = 'percona/pmm-server:2.33.0';

setup.describe.configure({ mode: 'parallel' });

/**
 * Extension point un-hardcode versions using environment variables
 * TODO: add detection of latest released and RC versions and previous release
 */
setup('Set default env.VARs', async () => {
  process.env.server_image = newImage;
  process.env.server_image_old = oldImage;
});
