/**
 * Why this file exists:
 * - Some UI code and libraries (e.g. MUI) expect browser APIs that jsdom does not fully implement.
 * - Tests run in Node, so we provide light polyfills to match what the components need.
 *
 * What this file adds:
 * 1) Extends Jest-DOM matchers for better assertions in @testing-library/react tests.
 * 2) Polyfills `window.matchMedia` so MUI and responsive code paths do not crash in jsdom.
 * 3) Provides `TextEncoder`/`TextDecoder` on Node globals because some libs access them.
 * 4) Adds a minimal `ResizeObserver` stub for components that read layout changes.
 *
 * Scope:
 * - Only testing environment is affected. Production code is unchanged.
 */
import '@testing-library/jest-dom';

/** Minimal matchMedia polyfill for jsdom/MUI. */
if (typeof window !== 'undefined' && !window.matchMedia) {
  window.matchMedia = (query: string): MediaQueryList => {
    return {
      matches: false,
      media: query,
      onchange: null,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => false,
    };
  };
}

/** TextEncoder/TextDecoder for Node test env (some libs expect them). */
import { TextEncoder as NodeTextEncoder, TextDecoder as NodeTextDecoder } from 'util';
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const g: any = globalThis;
if (typeof g.TextEncoder === 'undefined') g.TextEncoder = NodeTextEncoder;
if (typeof g.TextDecoder === 'undefined') g.TextDecoder = NodeTextDecoder;

/** Optional: ResizeObserver stub if components expect it. */
declare global {
  // Provide a minimal type to avoid "any"
  interface Window {
    ResizeObserver?: new () => {
      observe: (target: Element) => void;
      unobserve: (target: Element) => void;
      disconnect: () => void;
    };
  }
}

if (typeof window !== 'undefined' && !window.ResizeObserver) {
  window.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
}
