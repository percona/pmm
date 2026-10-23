import '@testing-library/jest-dom';

const mockClipboard = {
  writeText: vi.fn(),
  readText: vi.fn(),
};

Object.defineProperty(navigator, 'clipboard', {
  value: mockClipboard,
  writable: true,
  configurable: true,
});

Object.defineProperty(window, 'isSecureContext', {
  value: true,
  writable: true,
  configurable: true,
});

beforeEach(() => {
  mockClipboard.readText.mockClear();
  mockClipboard.writeText.mockClear();
});

// jsdom has no IntersectionObserver; tests that need one install their own.
class IntersectionObserverStub {
  observe = vi.fn();
  unobserve = vi.fn();
  disconnect = vi.fn();
  takeRecords = vi.fn(() => []);
}

Object.defineProperty(window, 'IntersectionObserver', {
  value: IntersectionObserverStub,
  writable: true,
  configurable: true,
});
