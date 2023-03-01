import { test, expect } from '@playwright/test';

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
      expect(this.stdout, `Stdout does not contain ${expectedValue}!`).toContain(expectedValue);
    })
  }

  async outContainsMany(expectedValues: string[]) {
    for (const val of expectedValues) {
      await test.step(`Verify command output contains ${val}`, async () => {
        expect.soft(this.stdout,`Stdout does not contain '${val}'!`).toContain(val);
      })
    }
    expect(
        test.info().errors,
        `'Contains all elements' failed with ${test.info().errors.length} error(s):\n${this.getErrors()}`
    ).toHaveLength(0);
  }

  private getErrors(): string {
    let errors: string[] = [];
    for (const obj of test.info().errors) {
      errors.push(`\t${obj.message.split('\n')[0]}`);
    }
    return errors.join('\n');
  }
}

export default Output;
