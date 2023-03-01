import { test, expect } from '@playwright/test'
import * as cli from '@helpers/cliHelper'
import PMMRestClient from '@tests/support/types/request'
import { teardown } from '@tests/helpers/containers'

test.describe.configure({ mode: 'parallel' })

test.describe('Upgrade PMM Server', async () => {
  test('shall respect relevant flags', async () => {
    const oldContainerName = 'pmm-server-upgrade-1'
    const newContainerName = 'pmm-server-upgrade-1-new'
    const oldImage = 'percona/pmm-server:2.32.0'
    const newImage = 'percona/pmm-server:2.33.0'
    const volumeName = 'pmm-data-upgrade-1'

    try {
      let output = await cli.exec(`
        pmm server docker install 
          --json
          --docker-image="${oldImage}"
          --http-listen-port=3080
          --https-listen-port=3443
          --container-name=${oldContainerName}
          --volume-name=${volumeName}`
      )
      await output.assertSuccess()

      output = await cli.exec(`
        pmm server docker upgrade
          --json
          --docker-image="${newImage}"
          --container-id=${oldContainerName}
          --new-container-name=${newContainerName}`
      )
      await output.assertSuccess()
      expect(output.stderr).toContain('Starting PMM Server')

      // Container name      
      output = await cli.exec(`docker ps --format="{{.Names}}" | grep "^${newContainerName}$" | wc -l`)
      expect(output.stdout.trim()).toEqual('1')

      // Volume name
      output = await cli.exec(`docker volume ls --format="{{.Name}}" | grep "^${volumeName}$" | wc -l`)
      expect(output.stdout.trim()).toEqual('1')

      // Docker image
      output = await cli.exec(`docker ps --format="{{.Names}} {{.Image}}" | grep -E "^${newContainerName} ${newImage}$" | wc -l`)
      expect(output.stdout.trim()).toEqual('1')

      const client = new PMMRestClient('admin', 'admin', 3080)
      await client.works()
    } finally {
      await teardown(['^pmm-server-upgrade-1'], [`^${volumeName}`])
    }
  })

  test('shall respect container name prefix', async () => {
    const oldContainerName = 'pmm-server-upgrade-2'
    const newContainerName = 'pmm-server-upgrade-2-new'
    const oldImage = 'percona/pmm-server:2.32.0'
    const newImage = 'percona/pmm-server:2.33.0'
    const volumeName = 'pmm-data-upgrade-2'

    try {
      let output = await cli.exec(`
        pmm server docker install 
          --json
          --docker-image="${oldImage}"
          --http-listen-port=4080
          --https-listen-port=4443
          --container-name=${oldContainerName}
          --volume-name=${volumeName}`
      )
      await output.assertSuccess()

      output = await cli.exec(`
        pmm server docker upgrade
          --json
          --docker-image="${newImage}"
          --container-id=${oldContainerName}
          --new-container-name-prefix=${newContainerName}`
      )
      await output.assertSuccess()
      expect(output.stderr).toContain('Starting PMM Server')

      // Container name      
      output = await cli.exec(`docker ps --format="{{.Names}}" | grep -E "^${newContainerName}.+$" | wc -l`)
      expect(output.stdout.trim()).toEqual('1')

      // Volume name
      output = await cli.exec(`docker volume ls --format="{{.Name}}" | grep "^${volumeName}$" | wc -l`)
      expect(output.stdout.trim()).toEqual('1')

      // Docker image
      output = await cli.exec(`docker ps --format="{{.Names}} {{.Image}}" | grep -E "^${newContainerName}.+ ${newImage}$" | wc -l`)
      expect(output.stdout.trim()).toEqual('1')

      const client = new PMMRestClient('admin', 'admin', 4080)
      await client.works()
    } finally {
      await teardown(['^pmm-server-upgrade-2'], [`^${volumeName}`])
    }
  })
})
