import type { SxProps, Theme } from '@mui/material';
import { describe, expect, it } from 'vitest';
import { flattenSx, mergeSx } from './sx';

const theme = { palette: { primary: { main: '#000' } } } as unknown as Theme;

describe('flattenSx', () => {
  it('returns an empty object for undefined-ish values', () => {
    expect(flattenSx(undefined as unknown as SxProps<Theme>, theme)).toEqual({});
    expect(flattenSx(null as unknown as SxProps<Theme>, theme)).toEqual({});
  });

  it('passes through a plain style object', () => {
    const sx = { color: 'red', mx: 2 };
    expect(flattenSx(sx, theme)).toEqual({ color: 'red', mx: 2 });
  });

  it('resolves a callback-style sx by calling it with the theme', () => {
    const sx: SxProps<Theme> = (t) => ({ color: t.palette.primary.main });
    expect(flattenSx(sx, theme)).toEqual({ color: '#000' });
  });

  it('merges an array of style objects left-to-right', () => {
    const sx: SxProps<Theme> = [{ color: 'red' }, { color: 'blue', mx: 1 }];
    expect(flattenSx(sx, theme)).toEqual({ color: 'blue', mx: 1 });
  });

  it('handles a mixed array of objects, callbacks, and falsy values', () => {
    const sx: SxProps<Theme> = [
      { color: 'red' },
      false,
      (t: Theme) => ({ bg: t.palette.primary.main }),
      null,
      { mx: 4 },
    ];
    expect(flattenSx(sx as SxProps<Theme>, theme)).toEqual({
      color: 'red',
      bg: '#000',
      mx: 4,
    });
  });

  it('handles nested arrays', () => {
    const sx = [[{ color: 'red' }, { mx: 1 }], { py: 2 }];
    expect(flattenSx(sx as unknown as SxProps<Theme>, theme)).toEqual({
      color: 'red',
      mx: 1,
      py: 2,
    });
  });
});

describe('mergeSx', () => {
  it('returns defaultSx when consumerSx is undefined', () => {
    const defaultSx: SxProps<Theme> = { color: 'red' };
    expect(mergeSx(defaultSx, undefined)).toBe(defaultSx);
  });

  it('returns consumerSx when defaultSx is undefined', () => {
    const consumerSx: SxProps<Theme> = { mx: 2 };
    expect(mergeSx(undefined, consumerSx)).toBe(consumerSx);
  });

  it('returns undefined when both are undefined', () => {
    expect(mergeSx(undefined, undefined)).toBeUndefined();
  });

  it('merges two plain objects with consumer taking precedence', () => {
    const result = mergeSx({ color: 'red', mx: 1 }, { color: 'blue', py: 2 });

    expect(typeof result).toBe('function');
    const resolved = (result as (t: Theme) => Record<string, unknown>)(theme);
    expect(resolved).toEqual({ color: 'blue', mx: 1, py: 2 });
  });

  it('merges a callback default with a plain consumer object', () => {
    const defaultSx: SxProps<Theme> = (t) => ({ color: t.palette.primary.main });
    const consumerSx: SxProps<Theme> = { mx: 3 };

    const result = mergeSx(defaultSx, consumerSx);
    const resolved = (result as (t: Theme) => Record<string, unknown>)(theme);
    expect(resolved).toEqual({ color: '#000', mx: 3 });
  });

  it('lets consumer callbacks override default values', () => {
    const defaultSx: SxProps<Theme> = { color: 'red', mx: 1 };
    const consumerSx: SxProps<Theme> = () => ({ color: 'green', py: 5 });

    const result = mergeSx(defaultSx, consumerSx);
    const resolved = (result as (t: Theme) => Record<string, unknown>)(theme);
    expect(resolved).toEqual({ color: 'green', mx: 1, py: 5 });
  });

  it('merges array-style sx props', () => {
    const defaultSx: SxProps<Theme> = [{ color: 'red' }, { mx: 1 }];
    const consumerSx: SxProps<Theme> = [{ py: 2 }];

    const result = mergeSx(defaultSx, consumerSx);
    const resolved = (result as (t: Theme) => Record<string, unknown>)(theme);
    expect(resolved).toEqual({ color: 'red', mx: 1, py: 2 });
  });
});
