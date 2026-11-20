import { AxiosError, type AxiosResponse } from 'axios';
import { getApiErrorMessage } from './api.utils';

const failed = (data?: { message?: string }) => {
  const error = new AxiosError<{ message?: string }>(
    'Request failed with status code 500'
  );
  error.response = { status: 500, data } as AxiosResponse;
  return error;
};

describe('getApiErrorMessage', () => {
  it('shows the server message, trimmed', () => {
    expect(getApiErrorMessage(failed({ message: '  boom \n' }))).toBe('boom');
  });

  it('cuts a long message to fit a snackbar', () => {
    const message = getApiErrorMessage(failed({ message: 'x'.repeat(200) }));

    expect(message).toBe(`${'x'.repeat(120)}...`);
  });

  it('falls back when the server sends no message', () => {
    expect(getApiErrorMessage(failed())).toBe('Something went wrong');
    expect(getApiErrorMessage(failed({ message: '' }), 'Network Error')).toBe(
      'Network Error'
    );
  });
});
