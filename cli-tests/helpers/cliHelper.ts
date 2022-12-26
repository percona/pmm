import assert from 'assert';
import { test } from '@playwright/test';
import Output from '@support/types/output';
import * as shell from 'shelljs';

export function verifyCommand(command: string, result = 'pass', getError = false): string {
  // eslint-disable-next-line @typescript-eslint/no-unsafe-call,@typescript-eslint/no-unsafe-member-access,@typescript-eslint/no-unsafe-assignment
  const { stdout, stderr, code } = shell.exec(command.replace(/(\r\n|\n|\r)/gm, ''), { silent: true });
  if (result === 'pass') {
    assert.ok(code === 0, `The command ${command} was expected to run without any errors, the error found ${stderr}`);
  } else {
    assert.ok(code !== 0, `The command ${command} was expected to return with failure but found to be executing without any error, the return code found ${code}`);
  }

  if (!getError) return stdout as string;

  return stderr as string;
}

/**
 * Shell(sh) exec() wrapper to return handy {@link Output} object.
 *
 * @param       command   sh command to execute
 * @return      {@link Output} instance
 */
export async function exec(command: string): Promise<Output> {
  // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
  const { stdout, stderr, code } = await test.step(`Run "${command}" command`, async () => {
    // eslint-disable-next-line @typescript-eslint/no-unsafe-call,@typescript-eslint/no-unsafe-member-access,@typescript-eslint/no-unsafe-return
    return shell.exec(command.replace(/(\r\n|\n|\r)/gm, ''), { silent: false });
  });

  return new Output(command, code as number, stdout as string, stderr as string);
}
