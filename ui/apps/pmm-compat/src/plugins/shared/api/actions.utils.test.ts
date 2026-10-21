import { getAction } from './actions';
import { ACTION_POLL_MAX_ATTEMPTS } from './actions.constants';
import { getActionResult } from './actions.utils';

jest.mock('./actions');

const getActionMock = jest.mocked(getAction);

const response = (done: boolean, output = '', error = '') => ({
  action_id: 'a1',
  pmm_agent_id: 'agent',
  output,
  done,
  error,
});

describe('getActionResult', () => {
  beforeEach(() => {
    jest.useFakeTimers();
    getActionMock.mockReset();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('should poll until the action is done', async () => {
    getActionMock
      .mockResolvedValueOnce(response(false))
      .mockRejectedValueOnce(new Error('network'))
      .mockResolvedValueOnce(response(true, 'summary'));

    const result = getActionResult('a1');
    await jest.runAllTimersAsync();

    expect(await result).toEqual({ value: 'summary', error: '' });
    expect(getActionMock).toHaveBeenCalledTimes(3);
  });

  it('should return the action error', async () => {
    getActionMock.mockResolvedValue(response(false, 'partial', 'failed'));

    const result = getActionResult('a1');
    await jest.runAllTimersAsync();

    expect(await result).toEqual({ value: 'partial', error: 'failed' });
  });

  it('should give up after the maximum number of attempts', async () => {
    getActionMock.mockResolvedValue(response(false));

    const result = getActionResult('a1');
    await jest.runAllTimersAsync();

    expect(await result).toEqual({ value: null, error: 'Unknown error' });
    expect(getActionMock).toHaveBeenCalledTimes(ACTION_POLL_MAX_ATTEMPTS);
  });
});
