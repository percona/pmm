import { test, expect } from '@playwright/test'
import * as cli from '@helpers/cliHelper'
import PMMRestClient from '@tests/support/types/request'
import PromiseRetry from 'promise-retry';
import { teardown } from '@tests/helpers/containers'

test.describe.configure({ mode: 'parallel' })

// This test requires the following Docker images to be already built
// - percona/pmm-server-upgrade:latest
//
// CI builds them automatically.
// You can build them locally with the following commands run from the repo root:
// - make -C admin build-docker PMM_RELEASE_VERSION=latest
test.describe('Upgrade PMM Server', async () => {
  test.describe('Installed via pmm cli', () => {
    test('shall upgrade properly', async () => {
      const oldContainerName = 'upgrader-1'
      const newContainerName = 'upgrader-1-new'
      const oldImage = process.env.UPGRADER_UPGRADE_OLD_IMAGE ?? 'percona/pmm-server:2.32.0'
      const upgraderName = 'pmm-server-upgrade-1'
      const volumeName = 'upgrader-data-1'
  
      try {
        let output = await cli.exec(`
          pmm server docker install 
            --json
            --disable-image-pull
            --docker-image="${oldImage}"
            --http-listen-port=6080
            --https-listen-port=6443
            --container-name=${oldContainerName}
            --volume-name=${volumeName}`
        )
        await output.assertSuccess()
  
        output = await cli.exec(`
          docker run -d \
            --name ${upgraderName}
            --volumes-from ${oldContainerName}
            -v /var/run/docker.sock:/var/run/docker.sock
            percona/pmm-server-upgrade
            pmm-server-upgrade
              run
              --debug
              --new-container-name-prefix ${newContainerName}`)
        await output.assertSuccess()
        
        await startAndVerifyUpgrade(6080, oldContainerName)
      } finally {
        await teardown([`^${oldContainerName}`, upgraderName], [`^${volumeName}`])
      }
    })
  })

  test.describe('Installed via Docker', () => {
    test('shall upgrade properly', async () => {
      const oldContainerName = 'upgrader-docker-1'
      const newContainerName = 'upgrader-docker-1-new'
      const oldImage = process.env.UPGRADER_UPGRADE_OLD_IMAGE ?? 'percona/pmm-server:2.32.0'
      const upgraderName = 'pmm-server-upgrade-docker-1'
      const volumeName = 'upgrader-docker-data-1'
  
      try {
        let output = await cli.exec(`docker volume create ${volumeName}`)
        await output.assertSuccess()

        output = await cli.exec(`
          docker run -d
            -p 7080:80
            -v ${volumeName}:/srv
            --name ${oldContainerName}
            ${oldImage}
        `)
        await output.assertSuccess()

        output = await cli.exec(`
          docker run -d \
            --name ${upgraderName}
            --volumes-from ${oldContainerName}
            -v /var/run/docker.sock:/var/run/docker.sock
            percona/pmm-server-upgrade
            pmm-server-upgrade
              run
              --debug
              --new-container-name-prefix ${newContainerName}`)
        await output.assertSuccess()

        await startAndVerifyUpgrade(7080, oldContainerName)
      } finally {
        await teardown([`^${oldContainerName}`, upgraderName], [`^${volumeName}`])
      }
    })
  })
})

async function startAndVerifyUpgrade(port, oldContainerName) {
  const client = new PMMRestClient('admin', 'admin', port)
  await client.works()

  const res = await client.doPost('/v1/Updates/Start', {method: 'PMM_SERVER_UPGRADE'})
  expect(res.ok()).toBeTruthy()

  // Wait until old container is stopped
  // Upgrade will download the latest image in the background. This can take a bit.
  await PromiseRetry(async retry => {
    try {
      const output = await cli.exec(`docker inspect ${oldContainerName} --format="{{ .State.Status }}"`)
      await output.assertSuccess()
      expect(output.stdout.trim()).toBe('exited')
    } catch(err) {
      return retry(err)
    }
  }, {
    retries: 120,
    minTimeout: 1000,
    maxTimeout: 1000,
  })

  await client.works()
}
