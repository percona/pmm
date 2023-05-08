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
