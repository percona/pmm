import { test as setup } from '@playwright/test';
import * as cli from '@helpers/cliHelper';

const oldImage = 'perconalab/pmm-server:3-dev-latest';
const newImage = 'perconalab/pmm-server:3-dev-latest';

setup.describe.configure({ mode: 'parallel' });

/**
 * Extension point un-hardcode versions using environment variables
 */
setup('Set default env.VARs', async () => {
  // TODO: add detection of latest released and RC versions and previous release:
  //  convert bash into api call with JS object parsing instead of jq
  // const release_latest = (await cli.exec('wget -q https://registry.hub.docker.com/v2/repositories/perconalab/pmm-server/tags -O - | jq -r .results[].name  | grep -v latest | sort -V | tail -n1'))
  //     .stdout;
  // rc_latest=$(wget -q "https://registry.hub.docker.com/v2/repositories/perconalab/pmm-server/tags?page_size=25&name=rc" -O - | jq -r .results[].name  | grep 3.*.*-rc$ | sort -V | tail -n1)
  // rc_minor=$(echo $rc_latest | awk -F. '{print $2}')
  // dev_latest="2.$((++rc_minor)).0"
  await setup.step('Set pmm-server versions', async () => {
    process.env.server_image = newImage;
    process.env.old_server_image = oldImage;
  });

  // Download main images to reduce threads of download progress logs
  await cli.exec(`docker pull ${oldImage} -q`);
  await cli.exec(`docker pull ${newImage} -q`);
});
