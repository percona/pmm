import { test, expect } from '@playwright/test'
import * as cli from '@helpers/cliHelper'
import PMMRestClient from '@tests/support/types/request'
import { teardown } from '@tests/helpers/containers'
import PromiseRetry from 'promise-retry';

test.describe.configure({ mode: 'parallel' })

test.describe('Self-update', async () => {
  // beforeAll runs in every worker
  // TODO: migrate to config.projects.dependencies with playwright 1.31 to run only once
  // https://playwright.dev/docs/next/api/class-testproject#test-project-dependencies
  test.beforeAll(async () => {
    let output = await cli.exec(`make -C ../admin build-docker PMM_RELEASE_VERSION=first`)
    await output.assertSuccess()

    output = await cli.exec(`make -C ../admin build-docker PMM_RELEASE_VERSION=latest`)
    await output.assertSuccess()
  })

  test('shall not update - running latest version', async () => {
    let output = await cli.exec(`docker tag percona/pmm-server-upgrade:latest psu-no-update:latest`)
    await output.assertSuccess()
    
    output = await cli.exec(`docker run -d
      --name psu-no-update
      -v /srv
      -v /var/run/docker.sock:/var/run/docker.sock
      psu-no-update:latest
      pmm-server-upgrade
        run
        --self-update-trigger-on-start
        --self-update-disable-image-pull
        --self-update-docker-image=psu-no-update
    `)
    await output.assertSuccess()

    try {
      await PromiseRetry(async retry => {
        try {
          const output = await cli.exec(`docker logs psu-no-update 2>&1`)
          await output.assertSuccess()
          await output.containsMany(['Already running the latest version'])
        } catch(err) {
          return retry(err)
        }
      }, {
        retries: 30,
        minTimeout: 1000,
        maxTimeout: 1000,
      })
    } finally {
      await teardown(['^psu-no-update'])
    }
  })

  test('shall update to the latest version', async () => {
    let output = await cli.exec(`docker tag percona/pmm-server-upgrade:first psu-update:first`)
    await output.assertSuccess()
    
    output = await cli.exec(`docker tag percona/pmm-server-upgrade:latest psu-update:latest`)
    await output.assertSuccess()

    output = await cli.exec(`docker run -d
      --name psu-update
      -v /srv
      -v /var/run/docker.sock:/var/run/docker.sock
      psu-update:first
      pmm-server-upgrade
        run
        --self-update-trigger-on-start
        --self-update-disable-image-pull
        --self-update-docker-image=psu-update
        --self-update-container-name-prefix=psu-update`)
    await output.assertSuccess()

    try {
      await PromiseRetry(async retry => {
        try {
          const output = await cli.exec(`docker ps -f="status=exited" -f="name=psu-update" -q | wc -l`)
          await output.assertSuccess()
          expect(output.stdout.trim()).toBe('1')
        } catch(err) {
          return retry(err)
        }
      }, {
        retries: 30,
        minTimeout: 1000,
        maxTimeout: 1000,
      })
    } finally {
      await teardown(['^psu-update'])
    }
  })

  test('shall restore API server on update failure', async () => {
    let output = await cli.exec(`docker tag percona/pmm-server-upgrade:latest psu-failed-update:latest`)
    await output.assertSuccess()
    
    output = await cli.exec(`docker run -d
      --name psu-failed-update
      -v /srv
      -v /var/run/docker.sock:/var/run/docker.sock
      psu-failed-update
      pmm-server-upgrade
        run
        --self-update-trigger-on-start
        --self-update-docker-image=gcr.io/distroless/base-debian11
        --self-update-container-name-prefix=psu-failed-update`)
    await output.assertSuccess()

    try {
      await PromiseRetry(async retry => {
        try {
          let output = await cli.exec(`docker logs psu-failed-update 2>&1`)
          await output.assertSuccess()
          await output.containsMany([
            'Stopping API server',
            'Restarting API server after self-update error',
          ])

          output = await cli.exec(`docker inspect psu-failed-update -f '{{ json .State.Health.Status }}'`)
          await output.assertSuccess()
          await output.containsMany(['"healthy"'])
        } catch(err) {
          return retry(err)
        }
      }, {
        retries: 30,
        minTimeout: 1000,
        maxTimeout: 1000,
      })
    } finally {
      await teardown(['^psu-failed-update'])
    }
  })
})
