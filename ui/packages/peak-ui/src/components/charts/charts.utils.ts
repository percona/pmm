import type { SparklinePoint } from './sparkline';

const START = Date.UTC(2026, 9, 1, 12, 0);

export const shareToPercent = (share: number): number =>
  Math.round(share * 1000) / 10;

export const formatShare = (share: number): string => {
  if (share > 0 && share < 0.001) return '<0.1%';
  return `${shareToPercent(share)}%`;
};

const random = (seed: number) => {
  let state = seed >>> 0;
  return () => {
    state = (state + 0x6d2b79f5) >>> 0;
    let t = state;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
};

export const makeSeries = ({
  seed = 1,
  points = 30,
  base = 100,
  spread = 40,
  stepMs = 60_000,
}: {
  seed?: number;
  points?: number;
  base?: number;
  spread?: number;
  stepMs?: number;
} = {}): SparklinePoint[] => {
  const next = random(seed);
  let level = base;
  return Array.from({ length: points }, (_, index) => {
    level = Math.max(0, level + (next() - 0.5) * spread);
    return {
      timestamp: START + index * stepMs,
      value: Math.round(level * 100) / 100,
    };
  });
};

export const timeFormatter = (time: Date) =>
  time.toLocaleTimeString('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    timeZone: 'UTC',
  });
