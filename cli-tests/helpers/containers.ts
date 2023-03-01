import * as cli from "./cliHelper"

export async function teardown(containers: string[], volumes: string[]): Promise<void> {
    const cmds = []
    for(const container of containers) {
        cmds.push(`docker ps -a --format="{{.Names}}" | grep -E "${container}" | xargs --no-run-if-empty docker rm -f`)
    }

    for(const volume of volumes) {
        cmds.push(`docker volume ls -q | grep -E "${volume}" | xargs --no-run-if-empty docker volume rm`)
    }

    await Promise.all(cmds.map(cmd => cli.exec(cmd)))
        .catch(err => console.error(err))
}
