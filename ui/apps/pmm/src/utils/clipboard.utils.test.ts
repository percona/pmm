import { describe, expect, it, vi } from 'vitest';
import { copyToClipboard } from 'utils/clipboard.utils';

describe('copyToClipboard', () => {
  it('writes text and resolves true when the clipboard is available', async () => {
    const writeText = vi.mocked(navigator.clipboard.writeText);
    writeText.mockResolvedValueOnce(undefined);

    await expect(copyToClipboard('hello')).resolves.toBe(true);
    expect(writeText).toHaveBeenCalledWith('hello');
  });

  it('resolves false when the context is not secure', async () => {
    const original = window.isSecureContext;
    Object.defineProperty(window, 'isSecureContext', {
      value: false,
      configurable: true,
    });

    await expect(copyToClipboard('hello')).resolves.toBe(false);

    Object.defineProperty(window, 'isSecureContext', {
      value: original,
      configurable: true,
    });
  });

  it('resolves false when writing fails', async () => {
    const writeText = vi.mocked(navigator.clipboard.writeText);
    writeText.mockRejectedValueOnce(new Error('denied'));

    await expect(copyToClipboard('hello')).resolves.toBe(false);
  });
});
