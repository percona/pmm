import { describe, expect, it } from 'vitest';
import {
  buildCreateRulePayload,
  getTemplateDefaults,
} from './CreateAlertFromTemplate.utils';
import { CreateRuleFormValues } from './CreateAlertFromTemplate.types';
import { createRuleSchema } from './CreateAlertFromTemplate.schema';
import {
  FilterType,
  ParamType,
  ParamUnit,
  Severity,
  Template,
  TemplateCategory,
  TemplateSource,
} from 'types/alert-templates.types';

const template: Template = {
  name: 'pmm_mysql_down',
  summary: 'MySQL down',
  expr: 'mysql_up == 0',
  params: [
    {
      name: 'threshold',
      summary: 'Threshold',
      unit: ParamUnit.PERCENTAGE,
      type: ParamType.FLOAT,
      float: { default: 80, min: 0, max: 100 },
    },
    {
      name: 'enabled',
      summary: 'Enabled',
      unit: ParamUnit.UNSPECIFIED,
      type: ParamType.BOOL,
      bool: { default: true },
    },
  ],
  for: '300s',
  severity: Severity.CRITICAL,
  labels: {},
  annotations: {},
  source: TemplateSource.USER_API,
  yaml: '',
  category: TemplateCategory.MYSQL,
};

describe('CreateAlertFromTemplate utils', () => {
  describe('getTemplateDefaults', () => {
    it('seeds name, severity, duration and param defaults', () => {
      const defaults = getTemplateDefaults(template);
      expect(defaults.template).toBe('pmm_mysql_down');
      expect(defaults.name).toBe('MySQL down alerting rule');
      expect(defaults.severity).toBe(Severity.CRITICAL);
      expect(defaults.duration).toBe('300');
      expect(defaults.interval).toBe('1m');
      expect(defaults.params).toEqual({ threshold: 80, enabled: true });
    });
  });

  describe('buildCreateRulePayload', () => {
    const values: CreateRuleFormValues = {
      template: 'pmm_mysql_down',
      name: '  My rule  ',
      severity: Severity.WARNING,
      duration: '120',
      folderUid: 'folder-1',
      group: '  group-a  ',
      interval: '5m',
      filters: [
        { type: FilterType.MATCH, label: 'env', regexp: 'prod' },
        { type: FilterType.MISMATCH, label: '', regexp: 'skip-me' },
      ],
      params: { threshold: '95', enabled: false },
    };

    it('maps params to typed ParamValues and serializes durations', () => {
      const payload = buildCreateRulePayload(values, template);
      expect(payload.templateName).toBe('pmm_mysql_down');
      expect(payload.name).toBe('My rule');
      expect(payload.group).toBe('group-a');
      expect(payload.folderUid).toBe('folder-1');
      expect(payload.for).toBe('120s');
      // Grafana-style duration string is normalized to "<seconds>s".
      expect(payload.interval).toBe('300s');
      expect(payload.params).toEqual([
        { name: 'threshold', type: ParamType.FLOAT, float: 95 },
        { name: 'enabled', type: ParamType.BOOL, bool: false },
      ]);
    });

    it('drops filters with an empty label', () => {
      const payload = buildCreateRulePayload(values, template);
      expect(payload.filters).toEqual([
        { type: FilterType.MATCH, label: 'env', regexp: 'prod' },
      ]);
    });
  });

  describe('evaluation interval validation (mirrors Grafana)', () => {
    const baseValues: CreateRuleFormValues = {
      template: 'pmm_mysql_down',
      name: 'My rule',
      severity: Severity.WARNING,
      duration: '60',
      folderUid: 'folder-1',
      group: 'group-a',
      interval: '1m',
      filters: [],
      params: { threshold: 80, enabled: true },
    };

    const intervalError = (interval: string) => {
      const result = createRuleSchema.safeParse({ ...baseValues, interval });
      if (result.success) {
        return undefined;
      }
      return result.error.issues.find((issue) =>
        issue.path.includes('interval')
      )?.message;
    };

    it('accepts valid duration strings that are multiples of 10s', () => {
      expect(intervalError('1m')).toBeUndefined();
      expect(intervalError('30s')).toBeUndefined();
      expect(intervalError('1h')).toBeUndefined();
    });

    it('rejects invalid, sub-10s, and non-multiple-of-10 intervals', () => {
      expect(intervalError('abc')).toBeDefined();
      expect(intervalError('5s')).toBeDefined();
      expect(intervalError('15s')).toBeDefined();
    });
  });
});
