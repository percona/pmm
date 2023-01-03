import { test, expect } from '@playwright/test'
import * as cli from '@helpers/cliHelper'

test.describe('PMM binary tests @pmm-cli', async () => {
  test('--version', async ({}) => {
    const output = await cli.exec('pmm --version')
    await output.assertSuccess()
  })
})
