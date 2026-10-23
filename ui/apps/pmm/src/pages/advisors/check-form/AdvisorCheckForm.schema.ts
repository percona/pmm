import { z } from 'zod';
import {
  AdvisorCheck,
  AdvisorCheckInput,
  AdvisorCheckQuery,
  AdvisorTechnology,
  AdvisorInterval,
} from 'types/advisors.types';
import { Messages } from './AdvisorCheckForm.messages';

const NAME_RE = /^[a-zA-Z_][a-zA-Z0-9_]*$/;

// user-check names carry a reserved prefix so they can never collide with
// current or future Percona-shipped check names (enforced server-side too)
export const USER_CHECK_NAME_PREFIX = 'custom_';

// a cleared name autocomplete sets null, hence the type error message
const parameterSchema = z.object({
  name: z
    .string({ error: Messages.validation.parameterName })
    .min(1, Messages.validation.parameterName),
  value: z.string(),
});

const querySchema = z.object({
  type: z.string().min(1, Messages.validation.queryType),
  // may be empty for parameterless query types (SHOW / getParameter)
  query: z.string(),
  // the API takes a map, so a repeated name would silently drop a row
  parameters: z.array(parameterSchema).superRefine((params, ctx) => {
    params.forEach(({ name }, index) => {
      if (name && params.findIndex((p) => p.name === name) < index) {
        ctx.addIssue({
          code: 'custom',
          message: Messages.validation.parameterDuplicate,
          path: [index, 'name'],
        });
      }
    });
  }),
});

export const advisorCheckFormSchema = z.object({
  name: z
    .string()
    .regex(NAME_RE, Messages.validation.name)
    .max(128, Messages.validation.nameMax)
    .startsWith(USER_CHECK_NAME_PREFIX, Messages.validation.namePrefix),
  summary: z.string().min(1, Messages.validation.required),
  description: z.string().min(1, Messages.validation.required),
  // a cleared autocomplete sets null, hence the type error message
  category: z
    .string({ error: Messages.validation.required })
    .min(1, Messages.validation.required),
  // the technology select never offers "unspecified"; an empty technology is rejected server-side
  technology: z.nativeEnum(AdvisorTechnology),
  interval: z.nativeEnum(AdvisorInterval),
  queries: z.array(querySchema).min(1, Messages.validation.queriesRequired),
  script: z.string().min(1, Messages.validation.required),
});

export type AdvisorCheckFormValues = z.infer<typeof advisorCheckFormSchema>;
type QueryFormValues = AdvisorCheckFormValues['queries'][number];

export const emptyFormValues: AdvisorCheckFormValues = {
  name: USER_CHECK_NAME_PREFIX,
  summary: '',
  description: '',
  category: '',
  technology: AdvisorTechnology.mysql,
  interval: AdvisorInterval.standard,
  queries: [{ type: 'MYSQL_SHOW', query: '', parameters: [] }],
  script: '',
};

const toQueryFormValues = ({
  type,
  query,
  parameters = {},
}: AdvisorCheckQuery): QueryFormValues => ({
  type,
  query,
  parameters: Object.entries(parameters).map(([name, value]) => ({
    name,
    value,
  })),
});

const toQueryInput = ({
  type,
  query,
  parameters,
}: QueryFormValues): AdvisorCheckQuery =>
  parameters.length === 0
    ? { type, query }
    : {
        type,
        query,
        parameters: Object.fromEntries(
          parameters.map(({ name, value }) => [name, value])
        ),
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
  technology:
    check.technology === AdvisorTechnology.unspecified
      ? AdvisorTechnology.mysql
      : check.technology,
  interval:
    check.interval === AdvisorInterval.unspecified
      ? AdvisorInterval.standard
      : check.interval,
  queries: (check.queries ?? []).map(toQueryFormValues),
  script: check.script ?? '',
});

export const toInput = (values: AdvisorCheckFormValues): AdvisorCheckInput => ({
  name: values.name,
  summary: values.summary,
  description: values.description,
  category: values.category,
  technology: values.technology,
  interval: values.interval,
  queries: values.queries.map(toQueryInput),
  script: values.script,
});
