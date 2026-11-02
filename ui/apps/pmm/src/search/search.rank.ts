import { PMM_NEW_NAV_GRAFANA_PATH } from 'lib/constants';
import {
  GROUP_CAPS,
  PROMOTABLE_GROUPS,
  RECENT_SHOWN,
  REMOTE_MIN_QUERY_LENGTH,
  ZERO_QUERY_ACTIONS,
} from './search.constants';
import { Messages } from './search.messages';
import { canonicalKey, normalizeText, tokenize } from './search.text';
import type {
  RemoteState,
  SearchGroupId,
  SearchItem,
  SearchResultGroup,
} from './search.types';

export interface PreparedItem {
  item: SearchItem;
  title: string;
  titleWords: string[];
  initials: string;
  keywordPhrases: string[];
  keywordWords: string[];
  haystack: string;
}

export interface PreparedQuery {
  q: string;
  tokens: string[];
}

export interface ItemScore {
  base: number;
  total: number;
}

export interface SearchGroupsContext {
  recentIds?: string[];
  dashboards?: RemoteState;
  inventory?: RemoteState;
}

const EXACT = 1000;
// A title prefix or better.
const PROMOTION_MIN = 900;
const FUZZY_CEILING = 200;
const FUZZY_GATE = 500;
const RECENT_BOOSTED = 10;
// With a one-character term only these tiers count: no substring or fuzzy matches.
const SHORT_TOKEN_TIERS = new Set([1000, 900, 800, 700]);

const STARTER_URLS = [
  `${PMM_NEW_NAV_GRAFANA_PATH}/`,
  `${PMM_NEW_NAV_GRAFANA_PATH}/d/pmm-qan/pmm-query-analytics`,
  `${PMM_NEW_NAV_GRAFANA_PATH}/dashboards`,
  `${PMM_NEW_NAV_GRAFANA_PATH}/advisors/insights`,
];

// Shown with an empty query, in this order, when the user can see them.
const STARTER_ACTION_IDS = [
  'action:inventory-services-action',
  'action:create-alert-rule',
  'action:theme-toggle',
  'action:start-tour',
];

const STARTER_HELP_IDS = ['page:help', 'help:pmm-docs'];

const splitWords = (text: string) => (text ? text.split(' ') : []);

const joinDescription = (description: SearchItem['description']) =>
  Array.isArray(description) ? description.join(' ') : (description ?? '');

const prepareItem = (item: SearchItem): PreparedItem => {
  const title = normalizeText(item.label);
  const titleWords = splitWords(title);
  const keywordPhrases = item.keywords.map(normalizeText).filter(Boolean);
  const aliasPhrases = (item.aliases ?? []).map(normalizeText).filter(Boolean);

  return {
    item,
    title,
    titleWords,
    initials: titleWords.map((word) => word[0]).join(''),
    keywordPhrases,
    keywordWords: [...keywordPhrases, ...aliasPhrases].flatMap(splitWords),
    haystack: [
      title,
      ...keywordPhrases,
      ...aliasPhrases,
      normalizeText(joinDescription(item.description)),
    ].join(' '),
  };
};

export const prepareItems = (items: SearchItem[]) => items.map(prepareItem);

export const prepareQuery = (query: string): PreparedQuery => {
  const tokens = tokenize(query);

  return { q: tokens.join(' '), tokens };
};

// The shortest stretch of `text` holding `query` as a subsequence, or null.
const subsequenceSpan = (text: string, query: string) => {
  let best: number | null = null;

  for (
    let start = text.indexOf(query[0]);
    start !== -1;
    start = text.indexOf(query[0], start + 1)
  ) {
    let matched = 1;
    let index = start + 1;

    for (; index < text.length && matched < query.length; index++) {
      if (text[index] === query[matched]) {
        matched++;
      }
    }

    if (matched < query.length) {
      break;
    }

    best = best === null ? index - start : Math.min(best, index - start);
  }

  return best;
};

const prefixesAny = (token: string, words: string[]) =>
  words.some((word) => word.startsWith(token));

const tierScore = (p: PreparedItem, { q, tokens }: PreparedQuery) => {
  const { title, titleWords, keywordWords } = p;

  if (title === q || p.keywordPhrases.includes(q)) {
    return EXACT;
  }

  if (title.startsWith(q)) {
    return 900;
  }

  if (tokens.every((token) => prefixesAny(token, titleWords))) {
    return 800;
  }

  if (tokens.length === 1 && q.length >= 2 && q === p.initials) {
    return 750;
  }

  if (
    tokens.every(
      (token) =>
        prefixesAny(token, titleWords) || prefixesAny(token, keywordWords)
    )
  ) {
    return 700;
  }

  if (q.length >= 2 && title.includes(q)) {
    return 500;
  }

  if (
    tokens.every((token) => token.length >= 2 && p.haystack.includes(token))
  ) {
    return 400;
  }

  if (tokens.length === 1 && q.length >= 3) {
    const span = subsequenceSpan(title, q);

    if (span !== null && span <= 3 * q.length) {
      return Math.max(101, FUZZY_CEILING - (span - q.length));
    }
  }

  return 0;
};

const baseScore = (p: PreparedItem, query: PreparedQuery) => {
  if (!query.tokens.length) {
    return 0;
  }

  const base = tierScore(p, query);
  const hasShortToken = query.tokens.some((token) => token.length === 1);

  return hasShortToken && !SHORT_TOKEN_TIERS.has(base) ? 0 : base;
};

// At most 41 in all, below the smallest gap between tiers, so boosts never reorder across tiers.
const boostFor = (item: SearchItem, recentRank?: number) => {
  let boost = 0;

  if (recentRank !== undefined && recentRank < RECENT_BOOSTED) {
    boost += 25 - 2 * recentRank;
  }

  if (item.starred) {
    boost += 10;
  }

  if (
    item.kind === 'page' ||
    item.kind === 'setting' ||
    item.kind === 'action'
  ) {
    boost += 6;
  }

  if (item.kind === 'dashboard' && item.curated) {
    boost += 3;
  }

  if (item.kind === 'help') {
    boost += 2;
  }

  return boost;
};

export const scoreItem = (
  p: PreparedItem,
  query: PreparedQuery,
  recentRank?: number
): ItemScore => {
  const base = baseScore(p, query);

  return base
    ? { base, total: base + boostFor(p.item, recentRank) }
    : { base: 0, total: 0 };
};

// The first item for each destination wins; the others leave their names and stars on it.
export const mergeIndex = (lists: SearchItem[][]): SearchItem[] => {
  const kept = new Map<string, SearchItem>();
  const merged: SearchItem[] = [];

  lists.forEach((list) =>
    list.forEach((item) => {
      const existing = kept.get(item.dedupeKey);

      if (!existing) {
        const copy = {
          ...item,
          keywords: [...item.keywords],
          order: merged.length,
        };
        kept.set(item.dedupeKey, copy);
        merged.push(copy);
        return;
      }

      [item.label, ...item.keywords].forEach((keyword) => {
        if (
          keyword !== existing.label &&
          !existing.keywords.includes(keyword)
        ) {
          existing.keywords.push(keyword);
        }
      });

      const aliases = (item.aliases ?? []).filter(
        (alias) => !existing.aliases?.includes(alias)
      );

      if (aliases.length) {
        existing.aliases = [...(existing.aliases ?? []), ...aliases];
      }

      if (item.starred) {
        existing.starred = true;
      }
    })
  );

  return merged;
};

interface ScoredItem extends ItemScore {
  item: SearchItem;
}

const byScore = (a: ScoredItem, b: ScoredItem) =>
  b.total - a.total ||
  a.item.label.length - b.item.label.length ||
  a.item.order - b.item.order;

const rowItem = (
  fields: Pick<SearchItem, 'id' | 'kind' | 'group' | 'label'> &
    Partial<SearchItem>
): SearchItem => ({
  keywords: [],
  dedupeKey: fields.id,
  order: Number.MAX_SAFE_INTEGER,
  ...fields,
});

const showAllRow = (query: string, count: number) =>
  rowItem({
    id: 'link:all-dashboards',
    kind: 'link',
    group: 'dashboards',
    label: Messages.showAllDashboards(count),
    to: `${PMM_NEW_NAV_GRAFANA_PATH}/dashboards?query=${encodeURIComponent(
      query.trim()
    )}`,
  });

const retryRow = (
  group: 'dashboards' | 'inventory',
  label: string,
  refetch: () => void
) =>
  rowItem({
    id: `retry:${group}`,
    kind: 'retry',
    group,
    label,
    description: Messages.retryDescription,
    status: 'error',
    perform: refetch,
  });

export const buildSearchGroups = (
  prepared: PreparedItem[],
  query: string,
  { recentIds = [], dashboards, inventory }: SearchGroupsContext = {}
): SearchResultGroup[] => {
  const preparedQuery = prepareQuery(query);

  if (!preparedQuery.tokens.length) {
    return [];
  }

  const remoteAllowed = preparedQuery.q.length >= REMOTE_MIN_QUERY_LENGTH;
  const recentRanks = new Map(recentIds.map((id, rank) => [id, rank]));
  const scored = prepared.flatMap((p): ScoredItem[] => {
    if (p.item.remote && !remoteAllowed) {
      return [];
    }

    const score = scoreItem(p, preparedQuery, recentRanks.get(p.item.id));

    return score.base ? [{ item: p.item, ...score }] : [];
  });
  const gated = scored.some((s) => s.base >= FUZZY_GATE)
    ? scored.filter((s) => s.base > FUZZY_CEILING)
    : scored;
  const byGroup = new Map<SearchGroupId, ScoredItem[]>();

  gated.sort(byScore).forEach((s) => {
    const list = byGroup.get(s.item.group) ?? [];
    list.push(s);
    byGroup.set(s.item.group, list);
  });

  const groups = GROUP_CAPS.map(([id, cap]) => {
    const matches = byGroup.get(id) ?? [];
    const items = matches.slice(0, cap).map((s) => s.item);

    if (id === 'dashboards' && matches.length > cap) {
      items.push(showAllRow(query, matches.length));
    }

    if (
      id === 'dashboards' &&
      remoteAllowed &&
      dashboards?.status === 'error'
    ) {
      items.push(
        retryRow('dashboards', Messages.retryDashboards, dashboards.refetch)
      );
    }

    if (id === 'inventory' && remoteAllowed && inventory?.status === 'error') {
      items.push(
        retryRow('inventory', Messages.retryInventory, inventory.refetch)
      );
    }

    return {
      group: { id, label: Messages.groups[id], items },
      best: matches[0]?.base ?? 0,
    };
  }).filter(({ group }) => group.items.length);

  const pagesBest = groups.find(({ group }) => group.id === 'pages')?.best ?? 0;
  const promoted = groups
    .filter(
      ({ group, best }) =>
        best >= PROMOTION_MIN &&
        best > pagesBest &&
        PROMOTABLE_GROUPS.includes(group.id)
    )
    .sort((a, b) => b.best - a.best);

  return [
    ...promoted,
    ...groups.filter((entry) => !promoted.includes(entry)),
  ].map(({ group }) => group);
};

export const buildZeroQueryGroups = (
  items: SearchItem[],
  recent: SearchItem[]
): SearchResultGroup[] => {
  const byId = new Map(items.map((item) => [item.id, item]));
  const byKey = new Map(items.map((item) => [item.dedupeKey, item]));
  // Overview is keyed by the home dashboard its URL redirects to, so it is found by the path it opens.
  const byUrl = (url: string) => {
    const key = canonicalKey(url) ?? url;

    return (
      byKey.get(key) ?? items.find((item) => canonicalKey(item.to) === key)
    );
  };
  const recentIds = new Set(recent.map((item) => item.id));
  const pick = (candidates: (SearchItem | undefined)[], max: number) =>
    candidates
      .filter((item): item is SearchItem => !!item && !recentIds.has(item.id))
      .slice(0, max);

  const first: SearchResultGroup = recent.length
    ? {
        id: 'recent',
        label: Messages.groups.recent,
        items: recent.slice(0, RECENT_SHOWN),
      }
    : {
        id: 'pages',
        label: Messages.groups.pages,
        items: pick(STARTER_URLS.map(byUrl), STARTER_URLS.length),
      };

  return [
    first,
    {
      id: 'actions' as const,
      label: Messages.groups.actions,
      items: pick(
        STARTER_ACTION_IDS.map((id) => byId.get(id)),
        ZERO_QUERY_ACTIONS
      ),
    },
    {
      id: 'help' as const,
      label: Messages.groups.help,
      items: pick(
        STARTER_HELP_IDS.map((id) => byId.get(id)),
        STARTER_HELP_IDS.length
      ),
    },
  ].filter((group) => group.items.length);
};
