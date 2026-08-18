import { describe, expect, it } from 'vitest';
import {
  beautifyUnit,
  durationToSeconds,
  formatCreatedAt,
  getTemplateExportFilename,
  isTemplateEditable,
  paramValueKey,
  secondsToDuration,
} from 'utils/alert-templates.utils';
import {
  ParamType,
  ParamUnit,
  Severity,
  Template,
  TemplateCategory,
  TemplateSource,
} from 'types/alert-templates.types';

const makeTemplate = (source: TemplateSource): Template => ({
  name: 'test',
  summary: 'Test template',
  expr: 'up == 0',
  params: [],
  for: '60s',
  severity: Severity.WARNING,
  labels: {},
  annotations: {},
  source,
  yaml: '',
  category: TemplateCategory.UNSPECIFIED,
});

describe('alert-templates.utils', () => {
  describe('isTemplateEditable', () => {
    it('returns true only for USER_API templates', () => {
      expect(isTemplateEditable(makeTemplate(TemplateSource.USER_API))).toBe(
        true
      );
      expect(isTemplateEditable(makeTemplate(TemplateSource.BUILT_IN))).toBe(
        false
      );
      expect(isTemplateEditable(makeTemplate(TemplateSource.USER_FILE))).toBe(
        false
      );
      expect(isTemplateEditable(makeTemplate(TemplateSource.SAAS))).toBe(false);
    });
  });

  describe('duration helpers', () => {
    it('formats seconds as a duration string', () => {
      expect(secondsToDuration(60)).toBe('60s');
    });

    it('parses a duration string back to seconds', () => {
      expect(durationToSeconds('60s')).toBe(60);
      expect(durationToSeconds('2m')).toBe(120);
      expect(durationToSeconds('1h')).toBe(3600);
    });
  });

  describe('beautifyUnit', () => {
    it('maps known units and falls back to empty string', () => {
      expect(beautifyUnit(ParamUnit.PERCENTAGE)).toBe('%');
      expect(beautifyUnit(ParamUnit.SECONDS)).toBe('s');
      expect(beautifyUnit(ParamUnit.UNSPECIFIED)).toBe('');
    });
  });

  describe('formatCreatedAt', () => {
    it('returns a dash for empty, zero-time, or invalid values', () => {
      expect(formatCreatedAt(undefined)).toBe('—');
      expect(formatCreatedAt('')).toBe('—');
      expect(formatCreatedAt('0001-01-01T00:00:00Z')).toBe('—');
      expect(formatCreatedAt('not-a-date')).toBe('—');
    });

    it('formats a real timestamp with date and time', () => {
      const result = formatCreatedAt('2026-06-30T14:25:00Z');
      expect(result).not.toBe('—');
      expect(result).toMatch(/2026/);
    });
  });

  describe('getTemplateExportFilename', () => {
    it('appends .yaml to the template name', () => {
      const template = {
        ...makeTemplate(TemplateSource.USER_API),
        name: 'my_template',
      };
      expect(getTemplateExportFilename(template)).toBe('my_template.yaml');
    });
  });

  describe('paramValueKey', () => {
    it('maps param types to oneof value keys', () => {
      expect(paramValueKey(ParamType.BOOL)).toBe('bool');
      expect(paramValueKey(ParamType.FLOAT)).toBe('float');
      expect(paramValueKey(ParamType.STRING)).toBe('string');
      expect(paramValueKey(ParamType.UNSPECIFIED)).toBeNull();
    });
  });
});
