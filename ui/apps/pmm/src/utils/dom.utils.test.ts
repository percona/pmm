import { waitForVisible } from './dom.utils';

describe('waitForVisible', () => {
  beforeEach(() => {
    document.body.innerHTML = '';
  });

  afterEach(() => {
    vi.clearAllTimers();
  });

  it('resolves immediately if element already exists in DOM', async () => {
    const testElement = document.createElement('div');
    testElement.id = 'test-element';
    document.body.appendChild(testElement);

    const result = await waitForVisible('#test-element');

    expect(result).toBe(true);
  });

  it('resolves when element is added to DOM', async () => {
    const promise = waitForVisible('#dynamic-element');

    // Simulate async DOM manipulation
    setTimeout(() => {
      const testElement = document.createElement('div');
      testElement.id = 'dynamic-element';
      document.body.appendChild(testElement);
    }, 100);

    const result = await promise;

    expect(result).toBe(true);
  });

  it('rejects when element is not found within timeout', async () => {
    await expect(
      waitForVisible('#non-existent-element', 100)
    ).rejects.toBeUndefined();
  });

  it('uses default timeout of 5000ms if not specified', async () => {
    const startTime = Date.now();

    try {
      await waitForVisible('#non-existent-element');
    } catch (error) {
      const elapsed = Date.now() - startTime;
      expect(elapsed).toBeGreaterThanOrEqual(4900); // Allow some margin
      expect(elapsed).toBeLessThan(5200);
    }
  });

  it('uses custom timeout when provided', async () => {
    const customTimeout = 1000;
    const startTime = Date.now();

    try {
      await waitForVisible('#non-existent-element', customTimeout);
    } catch (error) {
      const elapsed = Date.now() - startTime;
      expect(elapsed).toBeGreaterThanOrEqual(900); // Allow some margin
      expect(elapsed).toBeLessThan(1200);
    }
  });

  it('resolves for nested elements', async () => {
    const promise = waitForVisible('#parent #child');

    setTimeout(() => {
      const parent = document.createElement('div');
      parent.id = 'parent';
      const child = document.createElement('div');
      child.id = 'child';
      parent.appendChild(child);
      document.body.appendChild(parent);
    }, 50);

    const result = await promise;

    expect(result).toBe(true);
  });

  it('works with class selectors', async () => {
    const promise = waitForVisible('.test-class');

    setTimeout(() => {
      const testElement = document.createElement('div');
      testElement.className = 'test-class';
      document.body.appendChild(testElement);
    }, 50);

    const result = await promise;

    expect(result).toBe(true);
  });

  it('works with complex selectors', async () => {
    const promise = waitForVisible('div[data-testid="complex-element"]');

    setTimeout(() => {
      const testElement = document.createElement('div');
      testElement.setAttribute('data-testid', 'complex-element');
      document.body.appendChild(testElement);
    }, 50);

    const result = await promise;

    expect(result).toBe(true);
  });

  it('disconnects observer after element is found', async () => {
    const disconnectSpy = vi.spyOn(MutationObserver.prototype, 'disconnect');

    const promise = waitForVisible('#cleanup-test');

    setTimeout(() => {
      const testElement = document.createElement('div');
      testElement.id = 'cleanup-test';
      document.body.appendChild(testElement);
    }, 50);

    await promise;

    expect(disconnectSpy).toHaveBeenCalled();
    disconnectSpy.mockRestore();
  });

  it('clears timeout when element is found before timeout', async () => {
    const clearTimeoutSpy = vi.spyOn(global, 'clearTimeout');

    const promise = waitForVisible('#timeout-test', 5000);

    setTimeout(() => {
      const testElement = document.createElement('div');
      testElement.id = 'timeout-test';
      document.body.appendChild(testElement);
    }, 50);

    await promise;

    expect(clearTimeoutSpy).toHaveBeenCalled();
    clearTimeoutSpy.mockRestore();
  });
});
