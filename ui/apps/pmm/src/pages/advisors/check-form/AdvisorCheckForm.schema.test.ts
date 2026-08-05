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

const valid: AdvisorCheckFormValues = {
  name: 'custom_my_check',
  summary: 'My check',
  description: 'Checks something',
  category: 'Custom',
  subcategory: 'General',
  technology: AdvisorTechnology.mysql,
  interval: AdvisorInterval.standard,
  queries: [{ type: 'MYSQL_SHOW', query: '' }],
  script: 'def check_context(docs, context):\n    return []',
};

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
});

describe('toInput', () => {
  it('maps form values to an API payload', () => {
    expect(toInput(valid)).toEqual({
      name: 'custom_my_check',
      summary: 'My check',
      description: 'Checks something',
      category: 'Custom',
      subcategory: 'General',
      technology: AdvisorTechnology.mysql,
      interval: AdvisorInterval.standard,
      queries: [{ type: 'MYSQL_SHOW', query: '' }],
      script: valid.script,
    });
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
    subcategory: 'Sub',
    userDefined: true,
    queries: [{ type: 'POSTGRESQL_SELECT', query: 'SELECT 1' }],
    script: 'print(1)',
  };

  it('maps a check into form values', () => {
    expect(toFormValues(check)).toEqual({
      name: 'existing_check',
      summary: 'Existing',
      description: 'desc',
      category: 'Cat',
      subcategory: 'Sub',
      technology: AdvisorTechnology.postgresql,
      interval: AdvisorInterval.rare,
      queries: [{ type: 'POSTGRESQL_SELECT', query: 'SELECT 1' }],
      script: 'print(1)',
    });
  });

  it('prefills the name with the prefixed source name when cloning', () => {
    expect(toFormValues(check, true).name).toBe(
      `${USER_CHECK_NAME_PREFIX}existing_check`
    );
  });
});
