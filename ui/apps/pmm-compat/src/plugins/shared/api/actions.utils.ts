import { getAction } from './actions';
import {
  ACTION_POLL_INITIAL_DELAY_MS,
  ACTION_POLL_MAX_ATTEMPTS,
} from './actions.constants';
import { Messages } from './actions.messages';
import type { ActionResult } from '../types/actions.types';

const wait = (ms: number) =>
  new Promise<void>((resolve) => setTimeout(resolve, ms));

// Polls with exponential backoff until the action is done or failed.
export const getActionResult = async (
  actionId: string
): Promise<ActionResult> => {
  let delay = ACTION_POLL_INITIAL_DELAY_MS;

  for (let attempt = 1; attempt <= ACTION_POLL_MAX_ATTEMPTS; attempt++) {
    try {
      const { done, error, output } = await getAction(actionId);

      if (done || error) {
        return { value: output, error: error || '' };
      }
    } catch {
      // a failed poll counts as an attempt, like a pending one
    }

    if (attempt < ACTION_POLL_MAX_ATTEMPTS) {
      await wait(delay);
      delay *= 2;
    }
  }

  return { value: null, error: Messages.unknownError };
};
