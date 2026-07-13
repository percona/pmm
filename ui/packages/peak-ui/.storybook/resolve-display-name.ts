import type { ReactElement } from 'react';

const MEMO_TYPE = Symbol.for('react.memo');
const FORWARD_REF_TYPE = Symbol.for('react.forward_ref');
const FRAGMENT_TYPE = Symbol.for('react.fragment');

const GENERIC_RENDER_NAMES = new Set(['', 'Component', '_default', 'render']);
const REACT_MEMO_FALLBACK = 'React.Memo';
const REACT_FORWARD_REF_FALLBACK = 'React.ForwardRef';

// Minifiers append a numeric suffix when several identifiers share a name
// (e.g. MUI's `IconButton` render fn is emitted as `IconButton2`). Strip it so
// the snippet stays copy-pasteable — but only for compound camelCase names to
// avoid mangling deliberately number-suffixed components such as MUI's `Grid2`.
const stripBundlerSuffix = (name: string): string =>
  /[a-z][A-Z]/.test(name) ? name.replace(/\d+$/, '') : name;

const cleanName = (name: unknown): string | undefined => {
  if (typeof name !== 'string' || GENERIC_RENDER_NAMES.has(name)) return undefined;
  const stripped = stripBundlerSuffix(name);
  return GENERIC_RENDER_NAMES.has(stripped) ? undefined : stripped;
};

const resolveType = (type: unknown): string | undefined => {
  if (type == null) return undefined;
  if (typeof type === 'string') return type;
  if (typeof type === 'symbol') return type === FRAGMENT_TYPE ? 'React.Fragment' : undefined;

  const candidate = type as {
    $$typeof?: symbol;
    displayName?: string;
    name?: string;
    render?: { displayName?: string; name?: string };
    type?: unknown;
  };

  const cleanedDisplayName = cleanName(candidate.displayName);
  if (cleanedDisplayName) return cleanedDisplayName;

  switch (candidate.$$typeof) {
    case MEMO_TYPE:
      return resolveType(candidate.type) ?? REACT_MEMO_FALLBACK;
    case FORWARD_REF_TYPE:
      return (
        cleanName(candidate.render?.displayName) ??
        cleanName(candidate.render?.name) ??
        REACT_FORWARD_REF_FALLBACK
      );
    case FRAGMENT_TYPE:
      return 'React.Fragment';
    default:
      break;
  }

  if (typeof type === 'function') {
    return cleanName(candidate.displayName) ?? cleanName(candidate.name);
  }

  return undefined;
};

/**
 * Storybook's default JSX renderer reaches the `forwardRef`/`memo` symbol branch
 * before it inspects the wrapped `render.name`, so wrappers without an explicit
 * `displayName` (e.g. bare MUI components) print as `React.ForwardRef`/`React.Memo`.
 * This resolver unwraps those exotic types so the code panel shows a usable name.
 */
export const resolveDisplayName = (element: ReactElement): string =>
  resolveType(element.type) ?? REACT_FORWARD_REF_FALLBACK;
