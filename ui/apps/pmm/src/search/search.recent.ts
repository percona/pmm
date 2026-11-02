import type { User } from 'types/user.types';
import { safeLocalStorage } from 'utils/storage.utils';
import { RECENT_PREFIX, RECENT_SHOWN, RECENT_STORED } from './search.constants';
import { canonicalKey, carriesVariables } from './search.text';
import type { RecentEntry, SearchItem, SearchItemKind } from './search.types';

const RECORDED_KINDS = new Set<SearchItemKind>([
  'page',
  'dashboard',
  'folder',
  'setting',
  'service',
  'node',
  'help',
]);

const STORAGE_VERSION = 1;

export const getRecentKey = (user: Pick<User, 'orgId' | 'id'>) =>
  `${RECENT_PREFIX}.${user.orgId}.${user.id}`;

export const isRecordable = (item: SearchItem) =>
  !!item.to && RECORDED_KINDS.has(item.kind);

const isRecentEntry = (value: unknown): value is RecentEntry => {
  const entry = value as RecentEntry | null;

  return (
    typeof entry?.id === 'string' &&
    typeof entry.label === 'string' &&
    typeof entry.to === 'string' &&
    RECORDED_KINDS.has(entry.kind)
  );
};

export const readRecent = (key: string): RecentEntry[] => {
  try {
    const stored = JSON.parse(safeLocalStorage.getItem(key) ?? 'null');

    return stored?.v === STORAGE_VERSION && Array.isArray(stored.items)
      ? stored.items.filter(isRecentEntry)
      : [];
  } catch {
    return [];
  }
};

export const pushRecent = (
  key: string,
  item: SearchItem,
  now = Date.now()
): RecentEntry[] => {
  if (!isRecordable(item) || !item.to) {
    return readRecent(key);
  }

  const entry: RecentEntry = {
    id: item.id,
    kind: item.kind,
    label: item.label,
    description: item.description,
    to: item.to,
    at: now,
  };
  const items = [
    entry,
    ...readRecent(key).filter((recent) => recent.id !== item.id),
  ].slice(0, RECENT_STORED);

  safeLocalStorage.setItem(key, JSON.stringify({ v: STORAGE_VERSION, items }));

  return items;
};

// Live items replace their snapshots, so a renamed dashboard shows its current name, and each destination shows once.
export const resolveRecent = (
  entries: RecentEntry[],
  items: SearchItem[],
  pathname: string
): SearchItem[] => {
  const byId = new Map(items.map((item) => [item.id, item]));
  const current = canonicalKey(pathname);
  const shown = new Set<string>();

  return entries
    .map(
      (entry, order): SearchItem =>
        byId.get(entry.id) ?? {
          id: entry.id,
          kind: entry.kind,
          group: 'recent',
          label: entry.label,
          description: entry.description,
          keywords: [],
          to: entry.to,
          carryVariables: carriesVariables(entry.to),
          dedupeKey: canonicalKey(entry.to) ?? entry.id,
          order,
        }
    )
    .filter(({ dedupeKey, to }) => {
      const fresh =
        dedupeKey !== current &&
        canonicalKey(to) !== current &&
        !shown.has(dedupeKey);

      shown.add(dedupeKey);

      return fresh;
    })
    .slice(0, RECENT_SHOWN);
};
