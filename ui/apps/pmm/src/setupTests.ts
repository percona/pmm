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
