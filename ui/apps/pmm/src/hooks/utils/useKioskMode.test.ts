import { describe, it, expect, vi } from 'vitest';
import { useKioskMode } from './useKioskMode';

const setup = (params: string) => {
  const mockUseSearchParams = vi
    .fn()
    .mockReturnValue([new URLSearchParams(params)]);

  vi.spyOn(require('react-router-dom'), 'useSearchParams').mockImplementation(
    mockUseSearchParams
  );
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
