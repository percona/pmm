import {
  CreateRulePayload,
  ParamType,
  ParamValue,
  Template,
} from 'types/alert-templates.types';
import {
  durationToSeconds,
  paramValueKey,
  secondsToDuration,
} from 'utils/alert-templates.utils';
import { CreateRuleFormValues } from './CreateAlertFromTemplate.types';
import { Messages } from './CreateAlertFromTemplate.messages';
import {
  DEFAULT_DURATION_SECONDS,
  DEFAULT_INTERVAL,
  INTERVAL_STEP_SECONDS,
  MIN_INTERVAL_SECONDS,
} from './CreateAlertFromTemplate.constants';

// Grafana's evaluation-interval validation: a valid Prometheus duration
// string, >= 10s, and a multiple of 10s. Returns an error message or
// undefined. Shared by the form schema and the new-group modal.
export const getIntervalError = (value: string): string | undefined => {
  const seconds = durationToSeconds(value);
  if (!value || Number.isNaN(seconds) || seconds <= 0) {
    return Messages.validation.invalidInterval;
  }
  if (seconds < MIN_INTERVAL_SECONDS) {
    return Messages.validation.intervalMin;
  }
  if (seconds % INTERVAL_STEP_SECONDS !== 0) {
    return Messages.validation.intervalMultiple;
  }
  return undefined;
};

export const getParamDefault = (
  param: Template['params'][number]
): number | boolean | string => {
  switch (param.type) {
    case ParamType.FLOAT:
      return param.float?.default ?? 0;
    case ParamType.BOOL:
      return param.bool?.default ?? false;
    case ParamType.STRING:
      return param.string?.default ?? '';
    default:
      return '';
  }
};

export const getTemplateDefaults = (
  template: Template
): Partial<CreateRuleFormValues> => ({
  template: template.name,
  name: `${template.summary || template.name} alerting rule`,
  severity: template.severity,
  duration: String(
    template.for
      ? durationToSeconds(template.for) || DEFAULT_DURATION_SECONDS
      : DEFAULT_DURATION_SECONDS
  ),
  interval: DEFAULT_INTERVAL,
  params: template.params.reduce<Record<string, number | boolean | string>>(
    (acc, param) => {
      acc[param.name] = getParamDefault(param);
      return acc;
    },
    {}
  ),
});

const coerceParamValue = (
  type: ParamType,
  raw: number | boolean | string
): Partial<ParamValue> => {
  const key = paramValueKey(type);
  if (!key) {
    return {};
  }
  switch (type) {
    case ParamType.FLOAT:
      return { float: Number(raw) };
    case ParamType.BOOL:
      return { bool: Boolean(raw) };
    case ParamType.STRING:
      return { string: String(raw) };
    default:
      return {};
  }
};

export const buildCreateRulePayload = (
  values: CreateRuleFormValues,
  template: Template
): CreateRulePayload => ({
  templateName: template.name,
  name: values.name.trim(),
  group: values.group.trim(),
  folderUid: values.folderUid,
  severity: values.severity,
  for: secondsToDuration(Number(values.duration)),
  interval: secondsToDuration(durationToSeconds(values.interval)),
  params: template.params.map<ParamValue>((param) => ({
    name: param.name,
    type: param.type,
    ...coerceParamValue(param.type, values.params[param.name]),
  })),
  filters: values.filters
    .filter((filter) => filter.label.trim() !== '')
    .map((filter) => ({
      type: filter.type,
      label: filter.label.trim(),
      regexp: filter.regexp,
    })),
});
