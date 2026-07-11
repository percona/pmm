import { describe, expect, it } from 'vitest';
import {
  beautifyUnit,
  durationToSeconds,
  isTemplateEditable,
  paramValueKey,
  secondsToDuration,
} from 'utils/alert-templates.utils';
import {
  ParamType,
  ParamUnit,
  Severity,
  Template,
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

  describe('paramValueKey', () => {
    it('maps param types to oneof value keys', () => {
      expect(paramValueKey(ParamType.BOOL)).toBe('bool');
      expect(paramValueKey(ParamType.FLOAT)).toBe('float');
      expect(paramValueKey(ParamType.STRING)).toBe('string');
      expect(paramValueKey(ParamType.UNSPECIFIED)).toBeNull();
    });
  });
});
