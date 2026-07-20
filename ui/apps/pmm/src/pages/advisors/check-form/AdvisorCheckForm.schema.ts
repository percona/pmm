import { z } from 'zod';
import {
  AdvisorCheck,
  AdvisorCheckInput,
  AdvisorFamily,
  AdvisorInterval,
} from 'types/advisors.types';
import { Messages } from './AdvisorCheckForm.messages';

const NAME_RE = /^[a-zA-Z_][a-zA-Z0-9_]*$/;

const querySchema = z.object({
  type: z.string().min(1, Messages.validation.queryType),
  // may be empty for parameterless query types (SHOW / getParameter)
  query: z.string(),
});

export const advisorCheckFormSchema = z.object({
  name: z
    .string()
    .regex(NAME_RE, Messages.validation.name)
    .max(128, Messages.validation.nameMax),
  summary: z.string().min(1, Messages.validation.required),
  description: z.string().min(1, Messages.validation.required),
  category: z.string().min(1, Messages.validation.required),
  subcategory: z.string().min(1, Messages.validation.required),
  // the family select never offers "unspecified"; an empty family is rejected server-side
  family: z.nativeEnum(AdvisorFamily),
  interval: z.nativeEnum(AdvisorInterval),
  queries: z.array(querySchema).min(1, Messages.validation.queriesRequired),
  script: z.string().min(1, Messages.validation.required),
});

export type AdvisorCheckFormValues = z.infer<typeof advisorCheckFormSchema>;

export const emptyFormValues: AdvisorCheckFormValues = {
  name: '',
  summary: '',
  description: '',
  category: '',
  subcategory: '',
  family: AdvisorFamily.mysql,
  interval: AdvisorInterval.standard,
  queries: [{ type: 'MYSQL_SHOW', query: '' }],
  script: '',
};

// toFormValues maps a fetched check into form values. When clearName is true
// (clone), the name is left blank so the user must provide a new one.
export const toFormValues = (
  check: AdvisorCheck,
  clearName = false
): AdvisorCheckFormValues => ({
  name: clearName ? '' : check.name,
  summary: check.summary,
  description: check.description,
  category: check.category,
  subcategory: check.subcategory,
  family:
    check.family === AdvisorFamily.unspecified
      ? AdvisorFamily.mysql
      : check.family,
  interval:
    check.interval === AdvisorInterval.unspecified
      ? AdvisorInterval.standard
      : check.interval,
  queries: (check.queries ?? []).map((q) => ({
    type: q.type,
    query: q.query,
  })),
  script: check.script ?? '',
});

export const toInput = (values: AdvisorCheckFormValues): AdvisorCheckInput => ({
  name: values.name,
  summary: values.summary,
  description: values.description,
  category: values.category,
  subcategory: values.subcategory,
  family: values.family,
  interval: values.interval,
  queries: values.queries.map((q) => ({ type: q.type, query: q.query })),
  script: values.script,
});
