import { test, expect } from '@playwright/test';
import { DateTime } from 'luxon';

class Output {
  command: string;
  code: number;
  stdout: string;
  stderr: string;

  constructor(command: string, exitCode: number, stdOut: string, stdErr: string) {
    this.command = command;
    this.code = exitCode;
    this.stdout = stdOut;
    this.stderr = stdErr;
  }

  getStdOutLines(): string[] {
    return this.stdout.trim().split('\n').filter((item) => item.trim().length > 0);
  }

  getStdErrLines(): string[] {
    return this.stderr.trim().split('\n').filter((item) => item.trim().length > 0);
  }

  /**
   * Encapsulates composition of auto generated container name.
   * The name is based on container start date and time,
   * which is parsed from specified output lines produced by 'pmm server docker upgrade --json'
   * Designed to parse: {@code (await cli.exec('pmm server docker upgrade --json')).getStdErrLines();}
   *
   * Example:
   * > logs line: {"level":"info","msg":"Starting PMM Server","time":"2023-05-04T12:47:49-04:00"}
   * > returns:   'pmm-server-2023-05-04-12-47-49'
   *
   * @param   logs    shell logs lines array {@link Output#getStdErrLines()}
   * @param   prefix  name prefix to generate format: 'prefix-YYYY-MM-DD-HH-MM-SS'
   * @return          container name {@code string} in format: 'pmm-server-YYYY-MM-DD-HH-MM-SS'
   */
  generateContainerNameFromLogs(prefix = 'pmm-server') {
    const foundLine = this.getStdErrLines().find((item) => item.includes('"Starting PMM Server","time":'));
    expect(foundLine, 'Output logs must be "json lines" and have "Starting PMM Server" with "time"').not.toBeUndefined();
    type LogLine = { level: string, msg: string, time: string };
    const startDateTime: string = (JSON.parse(foundLine.trim()) as LogLine).time;
    return `${prefix}-${DateTime.fromISO(startDateTime).toFormat('yyyy-MM-dd-HH-mm-ss')}`;
  }

  async assertSuccess() {
    await test.step(`Verify "${this.command}" command executed successfully`, async () => {
      expect(this.code, `"${this.command}" expected to exit with 0!\nStdout: ${this.stdout}\nStderr: "${this.stderr}"`).toEqual(0);
    });
  }

  async exitCodeEquals(expectedValue: number) {
    await test.step(`Verify "${this.command}" command exit code is ${expectedValue}`, async () => {
      expect(this.code, `"${this.command}" expected to exit with ${expectedValue}! Output: "${this.stdout}"`).toEqual(expectedValue);
    });
  }

  async outContains(expectedValue: string) {
    await test.step(`Verify command output contains ${expectedValue}`, async () => {
      expect(this.stdout, `Stdout should contain ${expectedValue}!`).toContain(expectedValue);
    });
  }

  async outContainsMany(expectedValues: string[]) {
    for (const val of expectedValues) {
      await test.step(`Verify command output contains ${val}`, async () => {
        expect.soft(this.stdout, `Stdout should contain '${val}'`).toContain(val);
      });
    }
    expect(
      test.info().errors,
      `'Contains all elements' failed with ${test.info().errors.length} error(s):\n${this.getErrors()}`,
    ).toHaveLength(0);
  }

  async outHasLine(expectedValue: string) {
    await test.step(`Verify command output has line: '${expectedValue}'`, async () => {
      expect(this.getStdOutLines(), `Stdout must have line: '${expectedValue}'`).toContainEqual(expectedValue);
    });
  }

  async errContainsMany(expectedValues: string[]) {
    for (const val of expectedValues) {
      expect.soft(this.stderr, `Stderr should contain '${val}'`).toContain(val);
    }
    expect(
      test.info().errors,
      `'Contains all elements' failed with ${test.info().errors.length} error(s):\n${this.getErrors()}`,
    ).toHaveLength(0);
  }

  private getErrors(): string {
    const errors: string[] = [];
    for (const obj of test.info().errors) {
      errors.push(`\t${obj.message.split('\n')[0]}`);
    }
    return errors.join('\n');
  }
}

export default Output;
