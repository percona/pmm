import { parseDuration } from 'utils/duration.utils';
import {
  ParamType,
  ParamUnit,
  Template,
  TemplateSource,
} from 'types/alert-templates.types';

// Only templates created via the API can be edited or deleted.
export const isTemplateEditable = (template: Template): boolean =>
  template.source === TemplateSource.USER_API;

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
