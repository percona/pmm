import { test, expect } from '@playwright/test'
import * as cli from '@helpers/cliHelper'
import PMMRestClient from '@tests/support/types/request'
import { teardown } from '@tests/helpers/containers'

test.describe.configure({ mode: 'parallel' })

test.describe('Install PMM Server', async () => {
  test('shall respect relevant flags', async ({ }) => {
    const adminPw = 'admin123'
    const containerName = 'pmm-server-install-test'
    const imageName = 'percona/pmm-server:2.32.0'
    const volumeName = 'pmm-data-install-test'

    try {
      let output = await cli.exec(`
        pmm server docker install 
          --json
          --admin-password=${adminPw}
          --docker-image="${imageName}"
          --https-listen-port=1443
          --http-listen-port=1080
          --container-name=${containerName}
          --volume-name=${volumeName}`
      )

      // Output
      await output.assertSuccess()
      expect(output.stderr).toContain('Starting PMM Server')

      // http client
      let client = new PMMRestClient('admin', adminPw, 1080)
      await client.works()

      // https client
      client = new PMMRestClient('admin', adminPw, 1443, {
        baseURL: 'https://localhost:1443',
        ignoreHTTPSErrors: true,
      })
      let resp = await client.doPost('/v1/Settings/Get')
      let respBody = await resp.json()

      expect(resp.ok()).toBeTruthy()
      expect(respBody).toHaveProperty('settings')

      // Container name      
      output = await cli.exec(`docker ps --format="{{.Names}}" | grep "^${containerName}$" | wc -l`)
      expect(output.stdout.trim()).toEqual('1')

      // Volume name
      output = await cli.exec(`docker volume ls --format="{{.Name}}" | grep "^${volumeName}$" | wc -l`)
      expect(output.stdout.trim()).toEqual('1')

      // Docker image
      output = await cli.exec(`docker ps --format="{{.Names}} {{.Image}}" | grep -E "^${containerName} ${imageName}$" | wc -l`)
      expect(output.stdout.trim()).toEqual('1')
    } finally {
      await teardown([`^${containerName}$`], [volumeName])
    }
  })
})
