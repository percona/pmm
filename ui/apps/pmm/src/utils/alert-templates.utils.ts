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

export const getTemplateExportFilename = (template: Template): string =>
  `${template.name}.yaml`;

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
