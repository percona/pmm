import { test } from '@playwright/test';
import Output from '@support/types/output';
import * as shell from 'shelljs';

/**
 * Shell(sh) exec() wrapper to use outside outside {@link test}
 * returns handy {@link Output} object.
 *
 * @param       command   sh command to execute
 * @return      {@link Output} instance
 */
export const execute = (command: string): Output => {
  console.log(`exec: "${command}"`);
  const { stdout, stderr, code } = shell.exec(command.replace(/(\r\n|\n|\r)/gm, ''), { silent: false });
  if (stdout.length > 0) console.log(`Out: "${stdout}"`);
  if (stderr.length > 0) console.log(`Error: "${stderr}"`);
  return new Output(command, code, stdout, stderr);
};

/**
 * Shell(sh) exec() wrapper to return handy {@link Output} object.
 *
 * @param       command   sh command to execute
 * @return      {@link Output} instance
 */
export const exec = async (command: string): Promise<Output> => {
  return test.step(`Run "${command}" command`, async () => {
    return execute(command);
  });
};
/**
 * Silent Shell(sh) exec() wrapper to return handy {@link Output} object.
 * Provides no logs to skip huge outputs.
 *
 * @param       command   sh command to execute
 * @return      {@link Output} instance
 */
export const execSilent = async (command: string): Promise<Output> => {
  const { stdout, stderr, code } = await test.step(`Run "${command}" command`, async () => {
    return shell.exec(command.replace(/(\r\n|\n|\r)/gm, ''), { silent: false });
  });
  return new Output(command, code, stdout, stderr);
};
