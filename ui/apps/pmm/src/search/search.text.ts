import { PMM_NEW_NAV_GRAFANA_PATH } from 'lib/constants';
import { MAX_QUERY_TOKENS } from './search.constants';

const DASHBOARD_PATH = new RegExp(`^${PMM_NEW_NAV_GRAFANA_PATH}/d/([^/?#]+)`);

// Words kept capitalised after a technology name, because they are names.
const PROPER_NOUNS = new Set(['Patroni', 'Galera']);

export const normalizeText = (text: string) =>
  text
    .normalize('NFKD')
    .replace(/\p{M}/gu, '')
    .toLowerCase()
    .replace(/[^\p{L}\p{N}]+/gu, ' ')
    .trim();

export const tokenize = (query: string) => {
  const normalized = normalizeText(query);

  return normalized ? normalized.split(' ').slice(0, MAX_QUERY_TOKENS) : [];
};

export const stripQuery = (url: string) => url.split(/[?#]/)[0];

// One key per destination: a dashboard by its uid, whatever its slug or query; anything else by its path.
export const canonicalKey = (url?: string) => {
  if (!url) {
    return undefined;
  }

  const path = stripQuery(url);
  const dashboard = DASHBOARD_PATH.exec(path);

  if (dashboard) {
    return `dashboard:${dashboard[1]}`;
  }

  return path.length > 1 ? path.replace(/\/+$/, '') || '/' : path;
};

export const carriesVariables = (to?: string) =>
  !!to && DASHBOARD_PATH.test(to) && !/[?#]/.test(to);

export const techLabel = (tech: string, child: string) => {
  if (normalizeText(child).startsWith(normalizeText(tech))) {
    return child;
  }

  const rest = child
    .split(' ')
    .map((word) =>
      /^[A-Z][a-z]+$/.test(word) && !PROPER_NOUNS.has(word)
        ? word.toLowerCase()
        : word
    )
    .join(' ');

  return `${tech} ${rest}`;
};
