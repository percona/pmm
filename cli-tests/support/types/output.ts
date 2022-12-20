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
      expect(this.code, `"${this.command}" expected to exit with 0! Error: "${this.stderr}"`).toEqual(0);
    });
  }

  async containsMany(expectedValues: string[]) {
    await test.step(`Verify "${this.command}" command output`, async () => {
      for (const val of expectedValues) {
        await test.step(`Verify command output contains ${val}`, async () => {
          expect(this.stdout).toContain(val);
        });
      }
    });
  }
}

export default Output;
