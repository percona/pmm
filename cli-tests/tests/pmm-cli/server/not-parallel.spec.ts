import { test, expect } from '@playwright/test'
import * as cli from '@helpers/cliHelper'
import PMMRestClient from '@tests/support/types/request'
import { teardown } from '@tests/helpers/containers'

test.describe('Install PMM Server - not parallel', async () => {
    test('shall install with no flags', async ({ }) => {
      try {
        const output = await cli.exec(`pmm server docker install --json`)
        await output.assertSuccess()
        expect(output.stderr).toContain('Starting PMM Server')

        // http client
        const client = new PMMRestClient('admin', 'admin')
        await client.works()
      } finally {
        await teardown(['^pmm-server$'], ['pmm-data'])
      }
    })
})

test.describe('Upgrade PMM Server - not parallel', async () => {
  test('shall upgrade with no flags', async () => {
    const oldImage = 'percona/pmm-server:2.32.0'

    try {
      let output = await cli.exec(`
        pmm server docker install 
          --json
          --docker-image="${oldImage}"`
      )
      await output.assertSuccess()

      output = await cli.exec(`pmm server docker upgrade --json`)
      await output.assertSuccess()
      expect(output.stderr).toContain('Starting PMM Server')

      // Docker image
      output = await cli.exec(`docker ps --format="{{.Image}}" | grep "^percona/pmm-server:2$" | wc -l`)
      expect(output.stdout.trim()).toEqual('1')

      const client = new PMMRestClient('admin', 'admin')
      await client.works()
    } finally {
      await teardown(['^pmm-server$', '^pmm-server-[-0-9]+$'], ['pmm-data'])
    }
  })
})
