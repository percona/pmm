import { describe, it, expect } from 'vitest';
import {
  removeUnits,
  addUnits,
  getResolutionPreset,
} from './MetricsResolution.utils';

describe('removeUnits', () => {
  it('strips the trailing s from all fields', () => {
    expect(removeUnits({ hr: '5s', mr: '10s', lr: '60s' })).toEqual({
      hr: '5',
      mr: '10',
      lr: '60',
    });
  });

  it('leaves values without a trailing s unchanged', () => {
    expect(removeUnits({ hr: '5', mr: '10', lr: '60' })).toEqual({
      hr: '5',
      mr: '10',
      lr: '60',
    });
  });
});

describe('addUnits', () => {
  it('appends s to all fields', () => {
    expect(addUnits({ hr: '5', mr: '10', lr: '60' })).toEqual({
      hr: '5s',
      mr: '10s',
      lr: '60s',
    });
  });
});

describe('removeUnits / addUnits roundtrip', () => {
  it('is lossless', () => {
    const original = { hr: '1s', mr: '5s', lr: '30s' };
    expect(addUnits(removeUnits(original))).toEqual(original);
  });
});

describe('getResolutionPreset', () => {
  it('identifies the rare preset', () => {
    expect(getResolutionPreset({ hr: '60s', mr: '180s', lr: '300s' })).toBe(
      'rare'
    );
  });

  it('identifies the standard preset', () => {
    expect(getResolutionPreset({ hr: '5s', mr: '10s', lr: '60s' })).toBe(
      'standard'
    );
  });

  it('identifies the frequent preset', () => {
    expect(getResolutionPreset({ hr: '1s', mr: '5s', lr: '30s' })).toBe(
      'frequent'
    );
  });

  it('returns custom for non-matching values', () => {
    expect(getResolutionPreset({ hr: '3s', mr: '7s', lr: '45s' })).toBe(
      'custom'
    );
  });

  it('returns custom when metricsResolutions is undefined', () => {
    expect(getResolutionPreset(undefined)).toBe('custom');
  });
});
