import * as cli from './cliHelper';

export const teardown = async (containers: string[], volumes: string[]): Promise<void> => {
  const cmds: string[] = [];

  for (const container of containers) {
    cmds.push(`docker ps -a --format="{{.Names}}" | grep -E "${container}" | xargs --no-run-if-empty docker rm -f`);
  }

  for (const volume of volumes) {
    cmds.push(`docker volume ls -q | grep -E "${volume}" | xargs --no-run-if-empty docker volume rm`);
  }

  await Promise.all(cmds.map((cmd) => cli.exec(cmd)))
    .catch((err) => console.error(err));
};

/**
 * Removes docker containers and volumes found by default name patterns: "*pmm-server*" and "*pmm-data*"
 */
export const cleanUpDocker = async (): Promise<void> => {
  const cmds: string[] = [];

  for (const container of
    cli.execute('docker ps -a --format="{{.Names}}" | grep -E "pmm-server"').getStdOutLines()) {
    cmds.push(`docker ps -a --format="{{.Names}}" | grep -E "^${container}$" | xargs --no-run-if-empty docker rm -f`);
  }

  for (const volume of cli.execute('docker volume ls -q | grep -E "pmm-data"').getStdOutLines()) {
    cmds.push(`docker volume ls -q | grep -E "^${volume}$" | xargs --no-run-if-empty docker volume rm`);
  }
  await Promise.all(cmds.map((cmd) => cli.execute(cmd)))
    .catch((err) => console.error(err));
};

/**
 * Removes code duplication in "pmm-cli/server/upgrade.spec.ts"
 * Runs pmm-server using docker command with specified parameters.
 *
 * @param   httpPort        binds port 80 redirect
 * @param   httpsPort       binds port 443 redirect
 * @param   volumeName      name for the docker volume
 * @param   containerName   name for the container
 */
export const runOldPmmServer = async (httpPort: number, httpsPort: number, volumeName: string, containerName: string) => {
  await (await cli.exec(`docker run --detach --restart always
        --publish ${httpPort}:80 
        --publish ${httpsPort}:443 
        -v ${volumeName}:/srv 
        --name ${containerName} 
        ${process.env.old_server_image}`)).assertSuccess();
};
