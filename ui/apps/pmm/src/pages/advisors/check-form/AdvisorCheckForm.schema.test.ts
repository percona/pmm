import { describe, it, expect } from 'vitest';
import {
  AdvisorCheck,
  AdvisorTechnology,
  AdvisorInterval,
} from 'types/advisors.types';
import {
  AdvisorCheckFormValues,
  advisorCheckFormSchema,
  toFormValues,
  toInput,
  USER_CHECK_NAME_PREFIX,
} from './AdvisorCheckForm.schema';
import { Messages } from './AdvisorCheckForm.messages';

const valid: AdvisorCheckFormValues = {
  name: 'custom_my_check',
  summary: 'My check',
  description: 'Checks something',
  category: 'Custom',
  technology: AdvisorTechnology.mysql,
  interval: AdvisorInterval.standard,
  queries: [{ type: 'MYSQL_SHOW', query: '', parameters: [] }],
  script: 'def check_context(docs, context):\n    return []',
};

const rangeQuery = (
  parameters: AdvisorCheckFormValues['queries'][number]['parameters']
): AdvisorCheckFormValues => ({
  ...valid,
  queries: [{ type: 'METRICS_RANGE', query: 'up', parameters }],
});

describe('advisorCheckFormSchema', () => {
  it('accepts a valid check', () => {
    expect(advisorCheckFormSchema.safeParse(valid).success).toBe(true);
  });

  it('rejects an invalid name', () => {
    expect(
      advisorCheckFormSchema.safeParse({ ...valid, name: '1 bad name' }).success
    ).toBe(false);
  });

  it('rejects a name longer than 128 characters', () => {
    expect(
      advisorCheckFormSchema.safeParse({
        ...valid,
        name: `custom_${'a'.repeat(128)}`,
      }).success
    ).toBe(false);
  });

  it('rejects a name without the reserved prefix', () => {
    expect(
      advisorCheckFormSchema.safeParse({ ...valid, name: 'my_check' }).success
    ).toBe(false);
  });

  it('requires a summary', () => {
    expect(
      advisorCheckFormSchema.safeParse({ ...valid, summary: '' }).success
    ).toBe(false);
  });

  it('asks for a cleared category', () => {
    const result = advisorCheckFormSchema.safeParse({
      ...valid,
      category: null,
    });
    expect(result.error?.issues).toEqual([
      expect.objectContaining({
        path: ['category'],
        message: Messages.validation.required,
      }),
    ]);
  });

  it('requires at least one query', () => {
    expect(
      advisorCheckFormSchema.safeParse({ ...valid, queries: [] }).success
    ).toBe(false);
  });

  it('requires a script', () => {
    expect(
      advisorCheckFormSchema.safeParse({ ...valid, script: '' }).success
    ).toBe(false);
  });

  it('allows an empty query text (parameterless types)', () => {
    expect(advisorCheckFormSchema.safeParse(valid).success).toBe(true);
  });

  it('accepts query parameters', () => {
    expect(
      advisorCheckFormSchema.safeParse(
        rangeQuery([
          { name: 'range', value: '1h' },
          { name: 'step', value: '5m' },
        ])
      ).success
    ).toBe(true);
  });

  it('rejects an empty parameter name', () => {
    const result = advisorCheckFormSchema.safeParse(
      rangeQuery([{ name: '', value: '1h' }])
    );
    expect(result.error?.issues).toEqual([
      expect.objectContaining({
        path: ['queries', 0, 'parameters', 0, 'name'],
        message: Messages.validation.parameterName,
      }),
    ]);
  });

  it('rejects a cleared parameter name', () => {
    const result = advisorCheckFormSchema.safeParse(
      rangeQuery([{ name: null as unknown as string, value: '1h' }])
    );
    expect(result.error?.issues).toEqual([
      expect.objectContaining({
        path: ['queries', 0, 'parameters', 0, 'name'],
        message: Messages.validation.parameterName,
      }),
    ]);
  });

  it('rejects a parameter name repeated within a query', () => {
    const result = advisorCheckFormSchema.safeParse(
      rangeQuery([
        { name: 'range', value: '1h' },
        { name: 'step', value: '5m' },
        { name: 'range', value: '2h' },
      ])
    );
    expect(result.error?.issues).toEqual([
      expect.objectContaining({
        path: ['queries', 0, 'parameters', 2, 'name'],
        message: Messages.validation.parameterDuplicate,
      }),
    ]);
  });
});

describe('toInput', () => {
  it('maps form values to an API payload', () => {
    expect(toInput(valid)).toStrictEqual({
      name: 'custom_my_check',
      summary: 'My check',
      description: 'Checks something',
      category: 'Custom',
      technology: AdvisorTechnology.mysql,
      interval: AdvisorInterval.standard,
      queries: [{ type: 'MYSQL_SHOW', query: '' }],
      script: valid.script,
    });
  });

  it('sends query parameters as a map', () => {
    expect(
      toInput(
        rangeQuery([
          { name: 'range', value: '1h' },
          { name: 'step', value: '5m' },
        ])
      ).queries
    ).toStrictEqual([
      {
        type: 'METRICS_RANGE',
        query: 'up',
        parameters: { range: '1h', step: '5m' },
      },
    ]);
  });
});

describe('toFormValues', () => {
  const check: AdvisorCheck = {
    name: 'existing_check',
    enabled: true,
    summary: 'Existing',
    description: 'desc',
    interval: AdvisorInterval.rare,
    technology: AdvisorTechnology.postgresql,
    category: 'Cat',
    userDefined: true,
    queries: [{ type: 'POSTGRESQL_SELECT', query: 'SELECT 1' }],
    script: 'print(1)',
  };

  it('maps a check into form values', () => {
    expect(toFormValues(check)).toStrictEqual({
      name: 'existing_check',
      summary: 'Existing',
      description: 'desc',
      category: 'Cat',
      technology: AdvisorTechnology.postgresql,
      interval: AdvisorInterval.rare,
      queries: [
        { type: 'POSTGRESQL_SELECT', query: 'SELECT 1', parameters: [] },
      ],
      script: 'print(1)',
    });
  });

  it('keeps query parameters', () => {
    const withParameters: AdvisorCheck = {
      ...check,
      queries: [
        {
          type: 'POSTGRESQL_SELECT',
          query: 'SELECT 1',
          parameters: { all_dbs: 'true' },
        },
        {
          type: 'METRICS_RANGE',
          query: 'up',
          parameters: { range: '1h', step: '5m' },
        },
      ],
    };

    expect(toFormValues(withParameters).queries).toStrictEqual([
      {
        type: 'POSTGRESQL_SELECT',
        query: 'SELECT 1',
        parameters: [{ name: 'all_dbs', value: 'true' }],
      },
      {
        type: 'METRICS_RANGE',
        query: 'up',
        parameters: [
          { name: 'range', value: '1h' },
          { name: 'step', value: '5m' },
        ],
      },
    ]);
    // clone and edit save what was fetched
    expect(toInput(toFormValues(withParameters, true)).queries).toStrictEqual([
      {
        type: 'POSTGRESQL_SELECT',
        query: 'SELECT 1',
        parameters: { all_dbs: 'true' },
      },
      {
        type: 'METRICS_RANGE',
        query: 'up',
        parameters: { range: '1h', step: '5m' },
      },
    ]);
  });

  it('prefills the name with the prefixed source name when cloning', () => {
    expect(toFormValues(check, true).name).toBe(
      `${USER_CHECK_NAME_PREFIX}existing_check`
    );
  });
});
