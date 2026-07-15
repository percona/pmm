import { parseDuration } from 'utils/duration.utils';
import { formatTimestampWithTime } from 'utils/datetime.utils';
import {
  ParamType,
  ParamUnit,
  Template,
  TemplateSource,
} from 'types/alert-templates.types';

const EMPTY_VALUE = '—';

// Built-in / SaaS templates carry no creation time; the backend serializes it
// as Go's zero time (0001-01-01...) rather than an empty string.
export const formatCreatedAt = (createdAt?: string): string => {
  if (!createdAt) {
    return EMPTY_VALUE;
  }
  const date = new Date(createdAt);
  if (Number.isNaN(date.getTime()) || date.getUTCFullYear() <= 1) {
    return EMPTY_VALUE;
  }
  return formatTimestampWithTime(createdAt);
};

// Only templates created via the API can be edited or deleted.
export const isTemplateEditable = (template: Template): boolean =>
  template.source === TemplateSource.USER_API;

// Category inferred from the template name/summary (heuristic for now).
export const TEMPLATE_CATEGORY_PMM = 'PMM';

const CATEGORY_PATTERNS: { label: string; test: RegExp }[] = [
  { label: 'MySQL', test: /mysql/i },
  { label: 'MongoDB', test: /mongo/i },
  { label: 'PostgreSQL', test: /postgres/i },
  { label: 'ProxySQL', test: /proxysql/i },
  { label: 'HAProxy', test: /haproxy/i },
  { label: 'Valkey', test: /valkey|redis/i },
];

export const getTemplateCategory = (template: Template): string => {
  const haystack = `${template.name} ${template.summary}`;
  const match = CATEGORY_PATTERNS.find(({ test }) => test.test(haystack));
  return match ? match.label : TEMPLATE_CATEGORY_PMM;
};

export const getTemplateCategories = (templates: Template[]): string[] =>
  Array.from(new Set(templates.map(getTemplateCategory))).sort();

export const secondsToDuration = (seconds: number): string => `${seconds}s`;

export const durationToSeconds = (duration: string): number =>
  parseDuration(duration) / 1000;

export const beautifyUnit = (unit: ParamUnit): string => {
  switch (unit) {
    case ParamUnit.PERCENTAGE:
      return '%';
    case ParamUnit.SECONDS:
      return 's';
    default:
      return '';
  }
};

export const paramValueKey = (
  type: ParamType
): 'bool' | 'float' | 'string' | null => {
  switch (type) {
    case ParamType.BOOL:
      return 'bool';
    case ParamType.FLOAT:
      return 'float';
    case ParamType.STRING:
      return 'string';
    default:
      return null;
  }
};
