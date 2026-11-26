import { describe, it, expect, vi } from 'vitest';
import { useSearchParams } from 'react-router-dom';
import { useKioskMode } from './useKioskMode';

// react-router-dom v7 is ESM-only (frozen namespace) — can't vi.spyOn its exports,
// so mock the module instead.
vi.mock('react-router-dom', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router-dom')>();
  return { ...actual, useSearchParams: vi.fn() };
});

const setup = (params: string) => {
  vi.mocked(useSearchParams).mockReturnValue([
    new URLSearchParams(params),
    vi.fn(),
  ]);
};

describe('useKioskMode', () => {
  it('should return active as true when kiosk mode is enabled', () => {
    setup('kiosk=');
    const kioskMode = useKioskMode();
    expect(kioskMode.active).toBe(true);
  });

  it('should return active as true when kiosk mode is enabled (kiosk=true)', () => {
    setup('kiosk=true');
    const kioskMode = useKioskMode();
    expect(kioskMode.active).toBe(true);
  });

  it('should return active as false when kiosk mode is not enabled', () => {
    setup('');
    const kioskMode = useKioskMode();
    expect(kioskMode.active).toBe(false);
  });

  it('should return active as false when kiosk mode is not enabled (kiosk=false)', () => {
    setup('kiosk=false');
    const kioskMode = useKioskMode();
    expect(kioskMode.active).toBe(false);
  });
});
