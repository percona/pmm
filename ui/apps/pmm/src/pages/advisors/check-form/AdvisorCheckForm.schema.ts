import { z } from 'zod';
import {
  AdvisorCheck,
  AdvisorCheckInput,
  AdvisorTechnology,
  AdvisorInterval,
} from 'types/advisors.types';
import { Messages } from './AdvisorCheckForm.messages';

const NAME_RE = /^[a-zA-Z_][a-zA-Z0-9_]*$/;

// user-check names carry a reserved prefix so they can never collide with
// current or future Percona-shipped check names (enforced server-side too)
export const USER_CHECK_NAME_PREFIX = 'custom_';

const querySchema = z.object({
  type: z.string().min(1, Messages.validation.queryType),
  // may be empty for parameterless query types (SHOW / getParameter)
  query: z.string(),
});

export const advisorCheckFormSchema = z.object({
  name: z
    .string()
    .regex(NAME_RE, Messages.validation.name)
    .max(128, Messages.validation.nameMax)
    .startsWith(USER_CHECK_NAME_PREFIX, Messages.validation.namePrefix),
  summary: z.string().min(1, Messages.validation.required),
  description: z.string().min(1, Messages.validation.required),
  category: z.string().min(1, Messages.validation.required),
  subcategory: z.string().min(1, Messages.validation.required),
  // the technology select never offers "unspecified"; an empty technology is rejected server-side
  technology: z.nativeEnum(AdvisorTechnology),
  interval: z.nativeEnum(AdvisorInterval),
  queries: z.array(querySchema).min(1, Messages.validation.queriesRequired),
  script: z.string().min(1, Messages.validation.required),
});

export type AdvisorCheckFormValues = z.infer<typeof advisorCheckFormSchema>;

export const emptyFormValues: AdvisorCheckFormValues = {
  name: USER_CHECK_NAME_PREFIX,
  summary: '',
  description: '',
  category: '',
  subcategory: '',
  technology: AdvisorTechnology.mysql,
  interval: AdvisorInterval.standard,
  queries: [{ type: 'MYSQL_SHOW', query: '' }],
  script: '',
};

// toFormValues maps a fetched check into form values. When cloneName is true
// (clone), the name is prefilled as "custom_<source check name>" so the clone
// starts with a valid, recognizable name the user can adjust.
export const toFormValues = (
  check: AdvisorCheck,
  cloneName = false
): AdvisorCheckFormValues => ({
  name: cloneName ? `${USER_CHECK_NAME_PREFIX}${check.name}` : check.name,
  summary: check.summary,
  description: check.description,
  category: check.category,
  subcategory: check.subcategory,
  technology:
    check.technology === AdvisorTechnology.unspecified
      ? AdvisorTechnology.mysql
      : check.technology,
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
  technology: values.technology,
  interval: values.interval,
  queries: values.queries.map((q) => ({ type: q.type, query: q.query })),
  script: values.script,
});
