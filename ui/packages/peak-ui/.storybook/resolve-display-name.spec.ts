import { forwardRef, memo, createElement } from 'react';
import type { ReactElement } from 'react';
import IconButton from '@mui/material/IconButton';
import Button from '@mui/material/Button';
import CachedOutlined from '@mui/icons-material/CachedOutlined';
import { describe, expect, it } from 'vitest';
import { resolveDisplayName } from './resolve-display-name';

const asElement = (type: unknown): ReactElement =>
  ({ type }) as unknown as ReactElement;

describe('resolveDisplayName', () => {
  it('returns the tag name for host elements', () => {
    expect(resolveDisplayName(asElement('div'))).toBe('div');
  });

  it('returns React.Fragment for fragment elements', () => {
    expect(resolveDisplayName(asElement(Symbol.for('react.fragment')))).toBe('React.Fragment');
  });

  it('uses an explicit displayName on a forwardRef wrapper (design-system components)', () => {
    const Wrapped = forwardRef<HTMLDivElement>((_, ref) =>
      createElement('div', { ref })
    );
    Wrapped.displayName = 'Chip';
    expect(resolveDisplayName(asElement(Wrapped))).toBe('Chip');
  });

  it('falls back to the inner render function name for unnamed forwardRef wrappers', () => {
    const Wrapped = forwardRef<HTMLDivElement>(function MyWidget(_, ref) {
      return createElement('div', { ref });
    });
    expect(resolveDisplayName(asElement(Wrapped))).toBe('MyWidget');
  });

  it('resolves a bare MUI forwardRef component to its component name', () => {
    expect(resolveDisplayName(asElement(IconButton))).toBe('IconButton');
    expect(resolveDisplayName(asElement(Button))).toBe('Button');
  });

  it('unwraps a memo component to the underlying name', () => {
    const Inner = function Panel() {
      return createElement('div');
    };
    expect(resolveDisplayName(asElement(memo(Inner)))).toBe('Panel');
  });

  it('resolves a MUI icon (memo + forwardRef) without emitting React.Memo', () => {
    const name = resolveDisplayName(asElement(CachedOutlined));
    expect(name).not.toBe('React.Memo');
    expect(name).not.toBe('React.ForwardRef');
  });

  it('resolves plain function components by name', () => {
    const Plain = function PlainThing() {
      return createElement('div');
    };
    expect(resolveDisplayName(asElement(Plain))).toBe('PlainThing');
  });

  it('does not leak generic inner names such as "Component"', () => {
    const Wrapped = forwardRef<HTMLDivElement>(function Component(_, ref) {
      return createElement('div', { ref });
    });
    expect(resolveDisplayName(asElement(Wrapped))).toBe('React.ForwardRef');
  });

  it('strips a bundler dedupe suffix from compound camelCase names', () => {
    const Wrapped = forwardRef<HTMLDivElement>(function IconButton2(_, ref) {
      return createElement('div', { ref });
    });
    expect(resolveDisplayName(asElement(Wrapped))).toBe('IconButton');
  });

  it('keeps deliberate numeric suffixes on single-word names (e.g. Grid2)', () => {
    const Wrapped = forwardRef<HTMLDivElement>(function Grid2(_, ref) {
      return createElement('div', { ref });
    });
    expect(resolveDisplayName(asElement(Wrapped))).toBe('Grid2');
  });

  it('strips a bundler dedupe suffix that comes from an explicit displayName', () => {
    const Wrapped = forwardRef<HTMLDivElement>((_, ref) => createElement('div', { ref }));
    Wrapped.displayName = 'IconButton2';
    expect(resolveDisplayName(asElement(Wrapped))).toBe('IconButton');
  });
});
