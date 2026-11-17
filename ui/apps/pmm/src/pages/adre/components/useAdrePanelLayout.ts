import { useCallback, useEffect, useState } from 'react';

const STORAGE_KEY = 'pmm-assistant-layout';

export const CHATS_WIDTH = { min: 200, max: 440, initial: 260 };
export const RAIL_WIDTH = { min: 280, max: 560, initial: 340 };

export interface AdrePanelLayout {
  chatsWidth: number;
  chatsOpen: boolean;
  railWidth: number;
  railOpen: boolean;
}

const DEFAULT_LAYOUT: AdrePanelLayout = {
  chatsWidth: CHATS_WIDTH.initial,
  chatsOpen: true,
  railWidth: RAIL_WIDTH.initial,
  railOpen: true,
};

const width = (
  value: unknown,
  bounds: { min: number; max: number; initial: number }
) =>
  typeof value === 'number' && Number.isFinite(value)
    ? Math.min(bounds.max, Math.max(bounds.min, value))
    : bounds.initial;

// Storage can throw in private windows or with site data blocked; the defaults then apply.
const load = (): AdrePanelLayout => {
  try {
    const saved = JSON.parse(localStorage.getItem(STORAGE_KEY) ?? '{}');
    return {
      chatsWidth: width(saved.chatsWidth, CHATS_WIDTH),
      chatsOpen: saved.chatsOpen !== false,
      railWidth: width(saved.railWidth, RAIL_WIDTH),
      railOpen: saved.railOpen !== false,
    };
  } catch {
    return DEFAULT_LAYOUT;
  }
};

/** The Assistant page's panel widths and which panels are open, remembered in this browser. */
export const useAdrePanelLayout = () => {
  const [layout, setLayout] = useState(load);
  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(layout));
    } catch {
      /* not remembered */
    }
  }, [layout]);
  const update = useCallback(
    (patch: Partial<AdrePanelLayout>) =>
      setLayout((prev) => ({ ...prev, ...patch })),
    []
  );
  return [layout, update] as const;
};
