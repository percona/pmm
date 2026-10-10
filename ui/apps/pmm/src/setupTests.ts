import '@testing-library/jest-dom';

// jsdom's :has() engine (nwsapi) throws SyntaxError when a selector contains an
// id React's useId generated (":r23:"). React wraps those ids in colons on
// purpose so they cannot be used as CSS selectors, and MUI passes them to
// label/aria attributes, so any percona-ui theme rule using :has() blows up as
// soon as a MUI input is on the page. Browsers treat an unparseable selector as
// simply not matching, so restore that: swallow SyntaxError only, and let every
// other error through.
const emptyNodeList = document.createDocumentFragment().querySelectorAll('*');

// nwsapi and jsdom build the error in their own realm, so `instanceof
// SyntaxError` is unreliable here - match on the name instead.
const isSyntaxError = (error: unknown) =>
  typeof error === 'object' &&
  error !== null &&
  (error as { name?: string }).name === 'SyntaxError';

(
  [
    ['matches', () => false],
    ['closest', () => null],
    ['querySelector', () => null],
    ['querySelectorAll', () => emptyNodeList],
  ] as const
).forEach(([method, fallback]) => {
  const original = Element.prototype[method] as (
    this: Element,
    selector: string
  ) => unknown;

  Element.prototype[method] = function (this: Element, selector: string) {
    try {
      return original.call(this, selector);
    } catch (error) {
      if (isSyntaxError(error)) {
        return fallback();
      }
      throw error;
    }
  } as never;
});

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
