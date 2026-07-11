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
import {
  DEFAULT_DURATION_SECONDS,
  DEFAULT_INTERVAL_SECONDS,
} from './CreateAlertFromTemplate.constants';

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
  interval: String(DEFAULT_INTERVAL_SECONDS),
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
  interval: secondsToDuration(Number(values.interval)),
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
