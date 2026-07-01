/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { createMemoryRouter, RouterProvider } from 'react-router-dom';
import { useState, type ReactNode } from 'react';
import { useFormContext } from 'react-hook-form';
import { SchemaFormRenderer } from './SchemaFormRenderer';
import {
  buildValidationRules,
  coerceFormValues,
} from './utils/validationMapper';
import { evaluatePredicate, isPresent } from './utils/predicateEvaluator';
import type { FormSection, RenderFieldOverride } from './types';

const useAlertConfigMock = vi.fn();

vi.mock('@sep/api', () => ({
  apiClient: { get: vi.fn(), post: vi.fn() },
  useAlertConfig: () => useAlertConfigMock(),
}));
import { apiClient } from '@sep/api';
const mockedApi = apiClient as unknown as { get: ReturnType<typeof vi.fn> };

function renderWithProviders(ui: ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
  );
}

describe('buildValidationRules', () => {
  it('emits required + minLength + maxLength + pattern for string fields', () => {
    const rules = buildValidationRules({
      type: 'string',
      name: 'title',
      label: 'Title',
      required: true,
      min_length: 3,
      max_length: 10,
      pattern: '^[a-z]+$',
    });
    expect(rules.required).toBe('Title is required');
    expect(rules.minLength).toMatchObject({ value: 3 });
    expect(rules.maxLength).toMatchObject({ value: 10 });
    expect((rules.pattern as { value: RegExp }).value).toBeInstanceOf(RegExp);
  });

  it('emits min/max for integer ge/le', () => {
    const rules = buildValidationRules({
      type: 'integer',
      name: 'n',
      label: 'N',
      ge: 1,
      le: 5,
    });
    expect(rules.min).toMatchObject({ value: 1 });
    expect(rules.max).toMatchObject({ value: 5 });
  });

  it('emits a validate fn enforcing minItems/maxItems on multi_choice', () => {
    const rules = buildValidationRules({
      type: 'multi_choice',
      name: 'tags',
      label: 'Tags',
      choices: [{ label: 'A', value: 'a' }],
      min_items: 1,
      max_items: 2,
    });
    const validate = rules.validate as (value: unknown) => true | string;
    expect(validate([])).toMatch(/at least 1/);
    expect(validate(['a'])).toBe(true);
    expect(validate(['a', 'b', 'c'])).toMatch(/at most 2/);
  });
});

describe('coerceFormValues', () => {
  it('parses integer strings and coerces empty to undefined', () => {
    const out = coerceFormValues({ age: '42', weight: '3.14', empty: '' }, [
      { type: 'integer', name: 'age', label: 'Age' },
      { type: 'float', name: 'weight', label: 'Weight' },
      { type: 'integer', name: 'empty', label: 'Empty' },
    ]);
    expect(out.age).toBe(42);
    expect(out.weight).toBeCloseTo(3.14);
    expect(out.empty).toBeUndefined();
  });

  it('leaves non-numeric fields alone', () => {
    const out = coerceFormValues({ title: 'hi', flag: true }, [
      { type: 'string', name: 'title', label: 'Title' },
      { type: 'bool', name: 'flag', label: 'Flag' },
    ]);
    expect(out).toEqual({ title: 'hi', flag: true });
  });

  it('passes through extra keys absent from fields (capability-injected fields)', () => {
    const out = coerceFormValues({ title: 'x', alert_on_fail: true }, [
      { type: 'string', name: 'title', label: 'Title' },
    ]);
    expect(out.alert_on_fail).toBe(true);
  });

  it('coerces nested dotted field paths', () => {
    const out = coerceFormValues(
      { source: { source_db_id: '42', source_query: '' } },
      [
        { type: 'integer', name: 'source.source_db_id', label: 'Schema' },
        { type: 'string', name: 'source.source_query', label: 'Query' },
      ]
    );
    expect(out).toEqual({ source: { source_db_id: 42, source_query: '' } });
  });

  it('unwraps option objects from service/schema/table/host fields to scalar ids', () => {
    const out = coerceFormValues(
      {
        serviceId: { id: 7, name: 'svc', type: 'mysql' },
        schemaName: { id: 11, name: 'app_prod' },
        tbl: { id: 101, name: 'users' },
        hostId: { id: 'nomad-1', name: 'db-mysql-01', address: '10.0.0.1' },
        empty: '',
      },
      [
        {
          type: 'service',
          name: 'serviceId',
          label: 'Service',
          service_types: [],
        },
        {
          type: 'schema',
          name: 'schemaName',
          label: 'Schema',
          depends_on: 'serviceId',
        },
        {
          type: 'table',
          name: 'tbl',
          label: 'Table',
          depends_on: 'schemaName',
        },
        { type: 'host', name: 'hostId', label: 'Host' },
        { type: 'service', name: 'empty', label: 'Empty', service_types: [] },
      ]
    );
    expect(out.serviceId).toBe(7);
    expect(out.schemaName).toBe(11);
    expect(out.tbl).toBe(101);
    expect(out.hostId).toBe('nomad-1');
    expect(out.empty).toBeUndefined();
  });
});

describe('SchemaFormRenderer — field rendering', () => {
  beforeEach(() => {
    mockedApi.get.mockReset();
    mockedApi.get.mockResolvedValue({ data: [] });
  });

  const sections: FormSection[] = [
    {
      title: 'Basics',
      fields: [
        {
          type: 'string',
          name: 'title',
          label: 'Title',
          required: true,
          min_length: 3,
        },
        { type: 'integer', name: 'count', label: 'Count', ge: 1, le: 10 },
        {
          type: 'float',
          name: 'rate',
          label: 'Rate',
          ge: 0,
          le: 1,
          step: 0.01,
        },
        { type: 'bool', name: 'active', label: 'Active' },
        {
          type: 'choice',
          name: 'size',
          label: 'Size',
          choices: [
            { label: 'Small', value: 's' },
            { label: 'Medium', value: 'm' },
            { label: 'Large', value: 'l' },
            { label: 'Extra', value: 'xl' },
          ],
        },
        {
          type: 'multi_choice',
          name: 'tags',
          label: 'Tags',
          choices: [
            { label: 'Alpha', value: 'a' },
            { label: 'Beta', value: 'b' },
          ],
        },
        { type: 'textarea', name: 'notes', label: 'Notes' },
        { type: 'datetime', name: 'when', label: 'When' },
        { type: 'yaml', name: 'cfg', label: 'Config' },
        { type: 'file', name: 'upload', label: 'Upload' },
        {
          type: 'table',
          name: 'tbl',
          label: 'Table',
          depends_on: 'schemaName',
        },
        { type: 'host', name: 'hostId', label: 'Host' },
      ],
    },
  ];

  it('renders every declared field', () => {
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={() => {}} />
    );
    expect(screen.getByLabelText(/Title/)).toBeInTheDocument();
    expect(screen.getByLabelText(/Count/)).toBeInTheDocument();
    expect(screen.getByLabelText(/Rate/)).toBeInTheDocument();
    expect(screen.getByLabelText(/Active/)).toBeInTheDocument();
    // percona-ui's SelectInput does not expose the label as accessible name;
    // identify by the stable MUI-generated id instead.
    expect(document.getElementById('mui-component-select-size')).not.toBeNull();
    expect(document.getElementById('mui-component-select-tags')).not.toBeNull();
    expect(screen.getByLabelText(/Notes/)).toBeInTheDocument();
    expect(screen.getByLabelText(/When/)).toBeInTheDocument();
    expect(screen.getByLabelText(/Config/)).toBeInTheDocument();
    expect(screen.getByLabelText(/Upload/)).toBeInTheDocument();
    // ServiceSelector / SchemaSelector / TableSelector / HostSelector use
    // percona-ui's AutoCompleteInput which renders a TextField — assert by label.
    expect(screen.getByLabelText('Table')).toBeInTheDocument();
    expect(screen.getByLabelText('Host')).toBeInTheDocument();
  });

  it('groups fields under section titles as fieldset legends', () => {
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={() => {}} />
    );
    const legend = screen.getByText('Basics');
    expect(legend.tagName.toLowerCase()).toBe('legend');
  });
});

describe('SchemaFormRenderer — section layout controls', () => {
  const collapsedSections: FormSection[] = [
    {
      title: 'Script preview',
      collapsible: true,
      collapsed_by_default: true,
      fields: [{ type: 'string', name: 'probe', label: 'Probe' }],
    },
  ];

  it('renders render_after_submit sections after the submit button', () => {
    const sections: FormSection[] = [
      {
        title: 'Before submit',
        fields: [{ type: 'string', name: 'before', label: 'Before' }],
      },
      {
        title: 'After submit',
        render_after_submit: true,
        fields: [{ type: 'string', name: 'after', label: 'After' }],
      },
    ];

    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={() => {}} />
    );

    const beforeTitle = screen.getByText('Before submit');
    const runButton = screen.getByRole('button', { name: 'Run' });
    const afterTitle = screen.getByText('After submit');

    expect(
      beforeTitle.compareDocumentPosition(runButton) &
        Node.DOCUMENT_POSITION_FOLLOWING
    ).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
    expect(
      runButton.compareDocumentPosition(afterTitle) &
        Node.DOCUMENT_POSITION_FOLLOWING
    ).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
  });

  it('starts collapsed sections closed when collapsed_by_default is true', () => {
    renderWithProviders(
      <SchemaFormRenderer sections={collapsedSections} onSubmit={() => {}} />
    );

    expect(screen.queryByLabelText('Probe')).toBeNull();
  });

  it('mounts collapsible section content when expanded', async () => {
    const user = userEvent.setup();

    renderWithProviders(
      <SchemaFormRenderer sections={collapsedSections} onSubmit={() => {}} />
    );

    await user.click(screen.getByRole('button', { name: 'Script preview' }));
    expect(await screen.findByLabelText('Probe')).toBeInTheDocument();
  });
});

describe('SchemaFormRenderer — validation + submission', () => {
  it('blocks submission when a required field is empty and surfaces global banner', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    const sections: FormSection[] = [
      {
        title: 'Main',
        fields: [
          { type: 'string', name: 'title', label: 'Title', required: true },
        ],
      },
    ];
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={onSubmit} />
    );

    await user.click(screen.getByRole('button', { name: /Run/ }));
    expect(onSubmit).not.toHaveBeenCalled();
    expect(
      await screen.findByText(/Fix the highlighted fields/)
    ).toBeInTheDocument();
    expect(screen.getByText(/Title is required/)).toBeInTheDocument();
  });

  it('submits coerced numeric values on valid input', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    const sections: FormSection[] = [
      {
        title: 'Main',
        fields: [
          { type: 'string', name: 'title', label: 'Title', required: true },
          { type: 'integer', name: 'count', label: 'Count' },
        ],
      },
    ];
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={onSubmit} />
    );

    await user.type(screen.getByLabelText(/Title/), 'hello');
    await user.type(screen.getByLabelText(/Count/), '7');
    await user.click(screen.getByRole('button', { name: /Run/ }));

    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ title: 'hello', count: 7 })
    );
  });

  it('renders the server submitError banner when provided', () => {
    renderWithProviders(
      <SchemaFormRenderer
        sections={[
          { title: 'x', fields: [{ type: 'string', name: 'a', label: 'A' }] },
        ]}
        onSubmit={() => {}}
        submitError="Server exploded"
      />
    );
    expect(screen.getByText('Server exploded')).toBeInTheDocument();
  });
});

describe('SchemaFormRenderer — multi_choice minItems', () => {
  it('blocks submission when selection count is below minItems', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    const sections: FormSection[] = [
      {
        title: 'Main',
        fields: [
          {
            type: 'multi_choice',
            name: 'tags',
            label: 'Tags',
            choices: [
              { label: 'Alpha', value: 'a' },
              { label: 'Beta', value: 'b' },
            ],
            min_items: 1,
          },
        ],
      },
    ];
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={onSubmit} />
    );

    await user.click(screen.getByRole('button', { name: /Run/ }));
    expect(onSubmit).not.toHaveBeenCalled();
    expect(
      await screen.findByText(/Fix the highlighted fields/)
    ).toBeInTheDocument();
    // percona-ui's SelectInput does not render the validate message visibly,
    // but it does flip aria-invalid on the native input — use that as the
    // end-to-end proof that the minItems validate fn ran and blocked submit.
    await waitFor(() =>
      expect(screen.getByTestId('select-input-tags')).toHaveAttribute(
        'aria-invalid',
        'true'
      )
    );
  });
});

describe('SchemaFormRenderer — multi_choice required', () => {
  it('blocks submission when a required multi_choice has the default empty array', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    const sections: FormSection[] = [
      {
        title: 'Upload',
        fields: [
          {
            type: 'multi_choice',
            name: 'upload',
            label: 'Upload providers',
            required: true,
            choices: [
              { label: 'S3', value: 'S3' },
              { label: 'Rsync', value: 'RSYNC' },
            ],
          },
        ],
      },
    ];
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={onSubmit} />
    );

    await user.click(screen.getByRole('button', { name: /Run/ }));
    expect(onSubmit).not.toHaveBeenCalled();
    expect(
      await screen.findByText(/Fix the highlighted fields/)
    ).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.getByTestId('select-input-upload')).toHaveAttribute(
        'aria-invalid',
        'true'
      )
    );
  });
});

describe('SchemaFormRenderer — multi_choice empty-state placeholder', () => {
  it('renders "Select…" placeholder when nothing is selected', () => {
    const sections: FormSection[] = [
      {
        title: 'Upload',
        fields: [
          {
            type: 'multi_choice',
            name: 'upload',
            label: 'Upload providers',
            choices: [
              { label: 'S3', value: 'S3' },
              { label: 'Rsync', value: 'RSYNC' },
            ],
          },
        ],
      },
    ];
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={vi.fn()} />
    );
    expect(screen.getByText('Select…')).toBeVisible();
  });

  it('does NOT render placeholder when at least one value is selected', () => {
    const sections: FormSection[] = [
      {
        title: 'Upload',
        fields: [
          {
            type: 'multi_choice',
            name: 'upload',
            label: 'Upload providers',
            choices: [
              { label: 'S3', value: 'S3' },
              { label: 'Rsync', value: 'RSYNC' },
            ],
          },
        ],
      },
    ];
    renderWithProviders(
      <SchemaFormRenderer
        sections={sections}
        onSubmit={vi.fn()}
        defaultValues={{ upload: ['S3'] }}
      />
    );
    expect(screen.queryByText('Select…')).toBeNull();
    expect(screen.getByText('S3')).toBeVisible();
  });
});

describe('SchemaFormRenderer — choice (select mode) empty-state placeholder', () => {
  it('renders "Select…" when >3 choices and no value is chosen', () => {
    const sections: FormSection[] = [
      {
        title: 'Main',
        fields: [
          {
            type: 'choice',
            name: 'region',
            label: 'Region',
            choices: [
              { label: 'us-east-1', value: 'us-east-1' },
              { label: 'us-west-2', value: 'us-west-2' },
              { label: 'eu-west-1', value: 'eu-west-1' },
              { label: 'ap-south-1', value: 'ap-south-1' },
            ],
          },
        ],
      },
    ];
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={vi.fn()} />
    );
    expect(screen.getByText('Select…')).toBeVisible();
  });

  it('radio-mode branch (≤3 choices) does NOT add a placeholder', () => {
    const sections: FormSection[] = [
      {
        title: 'Main',
        fields: [
          {
            type: 'choice',
            name: 'mode',
            label: 'Mode',
            choices: [
              { label: 'Fast', value: 'fast' },
              { label: 'Slow', value: 'slow' },
            ],
          },
        ],
      },
    ];
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={vi.fn()} />
    );
    expect(screen.queryByText('Select…')).toBeNull();
  });
});

describe('SchemaFormRenderer — multi_choice empty-state label shrink', () => {
  it('floats the label above when nothing is selected (no overlap with placeholder)', () => {
    const sections: FormSection[] = [
      {
        title: 'Upload',
        fields: [
          {
            type: 'multi_choice',
            name: 'upload',
            label: 'Upload providers',
            choices: [
              { label: 'S3', value: 'S3' },
              { label: 'Rsync', value: 'RSYNC' },
            ],
          },
        ],
      },
    ];
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={vi.fn()} />
    );
    // MUI renders the label text in both <label> and the notched <legend>; pick the <label>.
    const labelRoot = screen
      .getAllByText('Upload providers')[0]
      .closest('label');
    expect(labelRoot).toHaveAttribute('data-shrink', 'true');
    expect(labelRoot).toHaveClass('MuiInputLabel-shrink');
  });
});

describe('SchemaFormRenderer — choice (select mode) empty-state label shrink', () => {
  it('floats the label above on empty >3-choice select', () => {
    const sections: FormSection[] = [
      {
        title: 'Main',
        fields: [
          {
            type: 'choice',
            name: 'region',
            label: 'Region',
            choices: [
              { label: 'us-east-1', value: 'us-east-1' },
              { label: 'us-west-2', value: 'us-west-2' },
              { label: 'eu-west-1', value: 'eu-west-1' },
              { label: 'ap-south-1', value: 'ap-south-1' },
            ],
          },
        ],
      },
    ];
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={vi.fn()} />
    );
    const labelRoot = screen.getAllByText('Region')[0].closest('label');
    expect(labelRoot).toHaveAttribute('data-shrink', 'true');
    expect(labelRoot).toHaveClass('MuiInputLabel-shrink');
  });
});

describe('SchemaFormRenderer — file required', () => {
  it('blocks submission when a required file field is empty', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    const sections: FormSection[] = [
      {
        title: 'Main',
        fields: [
          { type: 'file', name: 'upload', label: 'Upload', required: true },
        ],
      },
    ];
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={onSubmit} />
    );

    await user.click(screen.getByRole('button', { name: /Run/ }));
    expect(onSubmit).not.toHaveBeenCalled();
    expect(
      await screen.findByText(/Fix the highlighted fields/)
    ).toBeInTheDocument();
    expect(screen.getByText(/Upload is required/)).toBeInTheDocument();
  });
});

describe('SchemaFormRenderer — cascade behaviour', () => {
  beforeEach(() => {
    mockedApi.get.mockReset();
  });

  it('clears downstream schema value when the upstream service changes', async () => {
    mockedApi.get.mockImplementation((url: string) => {
      if (url === '/sep/services/') {
        return Promise.resolve({
          data: {
            items: [
              { id: 1, name: 'mysql-prod-1', type: 'mysql' },
              { id: 2, name: 'mysql-staging-1', type: 'mysql' },
            ],
            total: 2,
            offset: 0,
            limit: 200,
          },
        });
      }
      if (url === '/sep/services/1/schemas') {
        return Promise.resolve({ data: [{ id: 11, name: 'app_production' }] });
      }
      if (url === '/sep/services/2/schemas') {
        return Promise.resolve({ data: [{ id: 21, name: 'app_staging' }] });
      }
      return Promise.resolve({ data: [] });
    });

    const user = userEvent.setup();
    const onSubmit = vi.fn();
    const sections: FormSection[] = [
      {
        title: 'Target',
        fields: [
          {
            type: 'service',
            name: 'serviceId',
            label: 'Service',
            service_types: ['mysql'],
          },
          {
            type: 'schema',
            name: 'schemaName',
            label: 'Schema',
            depends_on: 'serviceId',
          },
        ],
      },
    ];
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={onSubmit} />
    );

    const serviceCombo = screen.getByLabelText('Service');
    const schemaCombo = screen.getByLabelText('Schema');

    // Wait for services to load.
    await waitFor(() =>
      expect(mockedApi.get).toHaveBeenCalledWith(
        '/sep/services/',
        expect.anything()
      )
    );

    // Pick first service.
    await user.click(serviceCombo);
    const svcOption = await screen.findByRole('option', {
      name: /mysql-prod-1/,
    });
    await user.click(svcOption);

    // Schema fetch fires, dropdown becomes enabled.
    await waitFor(() => expect(schemaCombo).not.toBeDisabled());

    await user.click(schemaCombo);
    const schemaOption = await screen.findByRole('option', {
      name: 'app_production',
    });
    await user.click(schemaOption);
    expect((schemaCombo as HTMLInputElement).value).toBe('app_production');

    // Switch to another service — schema should be cleared by the cascade reset.
    await user.click(serviceCombo);
    const svc2 = await screen.findByRole('option', { name: /mysql-staging-1/ });
    await user.click(svc2);

    await waitFor(() => {
      expect((schemaCombo as HTMLInputElement).value).toBe('');
    });
  });
});

// ── evaluatePredicate ──────────────────────────────────────────────────────

describe('evaluatePredicate', () => {
  const v = (overrides: Record<string, unknown>) => overrides;

  it('truthy / falsy', () => {
    expect(evaluatePredicate({ truthy: 'x' }, v({ x: 'yes' }))).toBe(true);
    expect(evaluatePredicate({ truthy: 'x' }, v({ x: '' }))).toBe(false);
    expect(evaluatePredicate({ falsy: 'x' }, v({ x: '' }))).toBe(true);
    expect(evaluatePredicate({ falsy: 'x' }, v({ x: 1 }))).toBe(false);
    expect(evaluatePredicate({ truthy: 'arr' }, v({ arr: ['a'] }))).toBe(true);
    expect(evaluatePredicate({ truthy: 'arr' }, v({ arr: [] }))).toBe(false);
    // empty plain objects are falsy — mirrors Python bool({}) = False
    expect(evaluatePredicate({ truthy: 'obj' }, v({ obj: {} }))).toBe(false);
    expect(evaluatePredicate({ truthy: 'obj' }, v({ obj: { k: 1 } }))).toBe(
      true
    );
    expect(evaluatePredicate({ falsy: 'obj' }, v({ obj: {} }))).toBe(true);
  });

  it('equals / not_equals', () => {
    expect(
      evaluatePredicate(
        { equals: { mode: 'advanced' } },
        v({ mode: 'advanced' })
      )
    ).toBe(true);
    expect(
      evaluatePredicate({ equals: { mode: 'basic' } }, v({ mode: 'advanced' }))
    ).toBe(false);
    expect(
      evaluatePredicate(
        { not_equals: { mode: 'basic' } },
        v({ mode: 'advanced' })
      )
    ).toBe(true);
  });

  it('equals with $field ref', () => {
    expect(
      evaluatePredicate(
        { equals: { a: { $field: 'b' } } },
        v({ a: 'x', b: 'x' })
      )
    ).toBe(true);
    expect(
      evaluatePredicate(
        { equals: { a: { $field: 'b' } } },
        v({ a: 'x', b: 'y' })
      )
    ).toBe(false);
  });

  it('equals / not_equals coerce numeric literals (RHF stores int inputs as strings)', () => {
    // BE emits integer literals as JSON numbers; RHF stores text inputs as strings
    expect(
      evaluatePredicate({ equals: { swap_drop: 1 } }, v({ swap_drop: '1' }))
    ).toBe(true);
    expect(
      evaluatePredicate({ equals: { swap_drop: 1 } }, v({ swap_drop: '0' }))
    ).toBe(false);
    expect(
      evaluatePredicate({ not_equals: { swap_drop: 1 } }, v({ swap_drop: '0' }))
    ).toBe(true);
    expect(
      evaluatePredicate({ not_equals: { swap_drop: 1 } }, v({ swap_drop: '1' }))
    ).toBe(false);
    // string literals still use strict equality (no coercion)
    expect(
      evaluatePredicate({ equals: { mode: 'dsn' } }, v({ mode: 'dsn' }))
    ).toBe(true);
    expect(
      evaluatePredicate({ equals: { mode: 'dsn' } }, v({ mode: 'none' }))
    ).toBe(false);
  });

  it('gt / gte / lt / lte', () => {
    expect(evaluatePredicate({ gt: { n: 5 } }, v({ n: 6 }))).toBe(true);
    expect(evaluatePredicate({ gt: { n: 5 } }, v({ n: 5 }))).toBe(false);
    expect(evaluatePredicate({ gte: { n: 5 } }, v({ n: 5 }))).toBe(true);
    expect(evaluatePredicate({ lt: { n: 5 } }, v({ n: 4 }))).toBe(true);
    expect(evaluatePredicate({ lte: { n: 5 } }, v({ n: 5 }))).toBe(true);
    // RHF stores text-input values as strings — coerce before comparing
    expect(evaluatePredicate({ gt: { n: 5 } }, v({ n: '6' }))).toBe(true);
    expect(evaluatePredicate({ gt: { n: 5 } }, v({ n: '10' }))).toBe(true); // would be false lexicographically
    expect(evaluatePredicate({ lt: { n: 10 } }, v({ n: '9' }))).toBe(true); // "9" < "10" is false lexicographically
  });

  it('gt / gte / lt / lte with $field ref', () => {
    expect(
      evaluatePredicate({ gt: { a: { $field: 'b' } } }, v({ a: 6, b: 5 }))
    ).toBe(true);
    expect(
      evaluatePredicate({ gt: { a: { $field: 'b' } } }, v({ a: 5, b: 5 }))
    ).toBe(false);
    expect(
      evaluatePredicate({ gte: { a: { $field: 'b' } } }, v({ a: 5, b: 5 }))
    ).toBe(true);
    expect(
      evaluatePredicate({ lt: { a: { $field: 'b' } } }, v({ a: 4, b: 5 }))
    ).toBe(true);
    expect(
      evaluatePredicate({ lte: { a: { $field: 'b' } } }, v({ a: 5, b: 5 }))
    ).toBe(true);
  });

  it('any_present / all_present / none_present', () => {
    expect(
      evaluatePredicate({ any_present: ['a', 'b'] }, v({ a: '', b: 'x' }))
    ).toBe(true);
    expect(
      evaluatePredicate({ any_present: ['a', 'b'] }, v({ a: '', b: '' }))
    ).toBe(false);
    expect(
      evaluatePredicate({ all_present: ['a', 'b'] }, v({ a: '1', b: '2' }))
    ).toBe(true);
    expect(
      evaluatePredicate({ all_present: ['a', 'b'] }, v({ a: '1', b: '' }))
    ).toBe(false);
    expect(
      evaluatePredicate({ none_present: ['a', 'b'] }, v({ a: '', b: '' }))
    ).toBe(true);
    expect(
      evaluatePredicate({ none_present: ['a', 'b'] }, v({ a: 'x', b: '' }))
    ).toBe(false);
    // empty arrays are absent (mirrors BE behaviour for multiselect/list fields)
    expect(evaluatePredicate({ any_present: ['tags'] }, v({ tags: [] }))).toBe(
      false
    );
    expect(evaluatePredicate({ all_present: ['tags'] }, v({ tags: [] }))).toBe(
      false
    );
    expect(evaluatePredicate({ none_present: ['tags'] }, v({ tags: [] }))).toBe(
      true
    );
    expect(
      evaluatePredicate({ any_present: ['tags'] }, v({ tags: ['x'] }))
    ).toBe(true);
    // empty plain objects are absent — mirrors BE _field_is_present({}) = False
    expect(evaluatePredicate({ any_present: ['obj'] }, v({ obj: {} }))).toBe(
      false
    );
    expect(
      evaluatePredicate({ any_present: ['obj'] }, v({ obj: { k: 1 } }))
    ).toBe(true);
    // empty operand list → false (no vacuous truth)
    expect(evaluatePredicate({ all_present: [] }, v({}))).toBe(false);
    expect(evaluatePredicate({ none_present: [] }, v({}))).toBe(false);
  });

  it('all_truthy / any_truthy / all_falsy / any_falsy', () => {
    expect(
      evaluatePredicate({ all_truthy: ['a', 'b'] }, v({ a: 1, b: 2 }))
    ).toBe(true);
    expect(
      evaluatePredicate({ all_truthy: ['a', 'b'] }, v({ a: 1, b: 0 }))
    ).toBe(false);
    expect(
      evaluatePredicate({ any_truthy: ['a', 'b'] }, v({ a: 0, b: 1 }))
    ).toBe(true);
    expect(
      evaluatePredicate({ all_falsy: ['a', 'b'] }, v({ a: 0, b: '' }))
    ).toBe(true);
    expect(
      evaluatePredicate({ any_falsy: ['a', 'b'] }, v({ a: 1, b: 0 }))
    ).toBe(true);
    // empty operand list → false (no vacuous truth)
    expect(evaluatePredicate({ all_truthy: [] }, v({}))).toBe(false);
    expect(evaluatePredicate({ all_falsy: [] }, v({}))).toBe(false);
  });

  it('all_equal', () => {
    expect(
      evaluatePredicate(
        { all_equal: ['a', 'b', 'c'] },
        v({ a: 'x', b: 'x', c: 'x' })
      )
    ).toBe(true);
    expect(
      evaluatePredicate({ all_equal: ['a', 'b'] }, v({ a: 'x', b: 'y' }))
    ).toBe(false);
    // requires at least 2 fields — fewer returns false
    expect(evaluatePredicate({ all_equal: [] }, v({}))).toBe(false);
    expect(evaluatePredicate({ all_equal: ['a'] }, v({ a: 'x' }))).toBe(false);
  });

  it('all / any (logical combinators)', () => {
    expect(
      evaluatePredicate(
        { all: [{ truthy: 'a' }, { truthy: 'b' }] },
        v({ a: 1, b: 1 })
      )
    ).toBe(true);
    expect(
      evaluatePredicate(
        { all: [{ truthy: 'a' }, { truthy: 'b' }] },
        v({ a: 1, b: 0 })
      )
    ).toBe(false);
    expect(
      evaluatePredicate(
        { any: [{ truthy: 'a' }, { truthy: 'b' }] },
        v({ a: 0, b: 1 })
      )
    ).toBe(true);
    // empty all combinator → false (no vacuous truth)
    expect(evaluatePredicate({ all: [] }, v({}))).toBe(false);
  });

  it('xor — binary and n-ary (exactly one truthy)', () => {
    // binary cases
    expect(
      evaluatePredicate(
        { xor: [{ truthy: 'a' }, { truthy: 'b' }] },
        v({ a: 1, b: 0 })
      )
    ).toBe(true);
    expect(
      evaluatePredicate(
        { xor: [{ truthy: 'a' }, { truthy: 'b' }] },
        v({ a: 1, b: 1 })
      )
    ).toBe(false);
    expect(
      evaluatePredicate(
        { xor: [{ truthy: 'a' }, { truthy: 'b' }] },
        v({ a: 0, b: 0 })
      )
    ).toBe(false);
    // n-ary: exactly one of three
    expect(
      evaluatePredicate(
        { xor: [{ truthy: 'a' }, { truthy: 'b' }, { truthy: 'c' }] },
        v({ a: 1, b: 0, c: 0 })
      )
    ).toBe(true);
    expect(
      evaluatePredicate(
        { xor: [{ truthy: 'a' }, { truthy: 'b' }, { truthy: 'c' }] },
        v({ a: 1, b: 1, c: 0 })
      )
    ).toBe(false);
    // empty → false
    expect(evaluatePredicate({ xor: [] }, v({}))).toBe(false);
  });

  it('not', () => {
    expect(evaluatePredicate({ not: { truthy: 'x' } }, v({ x: '' }))).toBe(
      true
    );
    expect(evaluatePredicate({ not: { truthy: 'x' } }, v({ x: 'yes' }))).toBe(
      false
    );
  });

  it('returns false for unknown operator', () => {
    expect(
      evaluatePredicate({ unknown_op: 'x' } as Record<string, unknown>, v({}))
    ).toBe(false);
  });
});

// ── Conditional visibility (forbidden gates) ───────────────────────────────

describe('SchemaFormRenderer — conditional visibility', () => {
  const sections: FormSection[] = [
    {
      title: 'Main',
      fields: [
        { type: 'bool', name: 'advanced', label: 'Advanced mode' },
        {
          type: 'string',
          name: 'advancedOption',
          label: 'Advanced option',
          forbidden: [{ when: { falsy: 'advanced' } }],
        },
      ],
    },
  ];

  it('hides a field when its forbidden gate fires', () => {
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={() => {}} />
    );
    // advanced = false (default) → forbidden gate fires → advancedOption hidden
    expect(screen.queryByLabelText('Advanced option')).toBeNull();
  });

  it('shows a field when its forbidden gate no longer fires', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={() => {}} />
    );

    // Flip the bool — advanced becomes true → falsy('advanced') = false → gate doesn't fire
    await user.click(screen.getByLabelText('Advanced mode'));
    expect(await screen.findByLabelText('Advanced option')).toBeInTheDocument();
  });

  it('excludes hidden field from submission', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={onSubmit} />
    );

    // advanced=false → advancedOption hidden → submit without it
    await user.click(screen.getByRole('button', { name: /Run/ }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(onSubmit).toHaveBeenCalledWith(
      expect.not.objectContaining({ advancedOption: expect.anything() })
    );
  });

  it('includes revealed field in submission', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={onSubmit} />
    );

    await user.click(screen.getByLabelText('Advanced mode'));
    const input = await screen.findByLabelText('Advanced option');
    await user.type(input, 'secret');
    await user.click(screen.getByRole('button', { name: /Run/ }));

    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ advancedOption: 'secret' })
    );
  });

  // ── BoolField with a forbidden gate ──────────────────────────────────────
  // Mirrors the SEP-1061 ``skip_s3_safety_check`` scenario: a bool field is
  // visible only when the ``upload`` MultiChoice contains ``"S3"``. Locks in
  // that ``isPresent(false) === false`` so the default does not trip the
  // forbidden gate, and that the field is unregistered (omitted from the
  // submit payload) when hidden.
  const boolGateSections: FormSection[] = [
    {
      title: 'Upload',
      fields: [
        {
          type: 'multi_choice',
          name: 'upload',
          label: 'Upload providers',
          choices: [
            { label: 'Rsync', value: 'RSYNC' },
            { label: 'S3', value: 'S3' },
          ],
        },
        {
          type: 'bool',
          name: 'skip_s3_safety_check',
          label: 'Skip S3 safety check',
          forbidden: [{ when: { not: { contains: { upload: 'S3' } } } }],
        },
      ],
    },
  ];

  it('hides a BoolField when its upload-membership gate fires', () => {
    renderWithProviders(
      <SchemaFormRenderer sections={boolGateSections} onSubmit={() => {}} />
    );
    // upload defaults to [] → not contains "S3" → gate fires → checkbox hidden
    expect(screen.queryByLabelText('Skip S3 safety check')).toBeNull();
  });

  it('omits a hidden BoolField from the submit payload (RHF unregister)', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    renderWithProviders(
      <SchemaFormRenderer
        sections={boolGateSections}
        onSubmit={onSubmit}
        defaultValues={{ upload: ['RSYNC'], skip_s3_safety_check: true }}
      />
    );

    await user.click(screen.getByRole('button', { name: /Run/ }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(onSubmit).toHaveBeenCalledWith(
      expect.not.objectContaining({ skip_s3_safety_check: expect.anything() })
    );
  });
});

describe('SchemaFormRenderer — storage type visibility', () => {
  const sections: FormSection[] = [
    {
      title: 'Storage',
      fields: [
        {
          type: 'choice',
          name: 'storage_type',
          label: 'Storage Type',
          default: 's3',
          choices: [
            { label: 'S3-compatible', value: 's3' },
            { label: 'Filesystem', value: 'filesystem' },
          ],
        },
        {
          type: 'string',
          name: 'storage_s3_bucket',
          label: 'S3 Bucket',
          forbidden: [{ when: { not_equals: { storage_type: 's3' } } }],
        },
        {
          type: 'string',
          name: 'storage_filesystem_path',
          label: 'Filesystem Path',
          required: true,
          forbidden: [{ when: { not_equals: { storage_type: 'filesystem' } } }],
        },
      ],
    },
  ];

  it('shows S3 fields and hides filesystem when storage_type defaults to s3', () => {
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={() => {}} />
    );
    expect(
      screen.getByTestId('text-input-storage_s3_bucket')
    ).toBeInTheDocument();
    expect(
      screen.queryByTestId('text-input-storage_filesystem_path')
    ).toBeNull();
  });

  it('shows filesystem and hides S3 when storage_type is filesystem', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={() => {}} />
    );

    await user.click(screen.getByTestId('radio-option-filesystem'));
    expect(
      await screen.findByTestId('text-input-storage_filesystem_path')
    ).toBeInTheDocument();
    expect(screen.queryByTestId('text-input-storage_s3_bucket')).toBeNull();
  });
});

// ── isPresent unit tests ───────────────────────────────────────────────────

describe('isPresent', () => {
  it('treats null / undefined / empty string as absent', () => {
    expect(isPresent(null)).toBe(false);
    expect(isPresent(undefined)).toBe(false);
    expect(isPresent('')).toBe(false);
  });

  it('treats empty array as absent', () => {
    expect(isPresent([])).toBe(false);
    expect(isPresent(['x'])).toBe(true);
  });

  it('treats empty plain object as absent', () => {
    expect(isPresent({})).toBe(false);
    expect(isPresent({ k: 1 })).toBe(true);
  });

  it('treats non-empty scalars as present', () => {
    expect(isPresent(0)).toBe(true);
    expect(isPresent('x')).toBe(true);
  });

  it('treats false as absent so BoolField defaults do not trip forbidden gates', () => {
    // Matches the backend convention: only an explicit `true` toggle counts
    // as present. `0` stays present because numeric zero is a real value.
    expect(isPresent(false)).toBe(false);
    expect(isPresent(true)).toBe(true);
  });
});

// ── Cardinality rules ──────────────────────────────────────────────────────

describe('SchemaFormRenderer — cardinality_rules', () => {
  it('shows a violation banner when neither required field is filled', async () => {
    renderWithProviders(
      <SchemaFormRenderer
        sections={[
          {
            title: 'Source',
            cardinality_rules: [
              {
                fields: ['source_db_id', 'source_table_id'],
                min: 1,
                max: 1,
                message:
                  'Specify exactly one source: either DB or Table, not both.',
              },
            ],
            fields: [
              { type: 'string', name: 'source_db_id', label: 'Source DB' },
              {
                type: 'string',
                name: 'source_table_id',
                label: 'Source Table',
              },
            ],
          },
        ]}
        onSubmit={() => {}}
      />
    );

    // Both fields empty at mount → violation fires immediately
    expect(
      await screen.findByText(
        'Specify exactly one source: either DB or Table, not both.'
      )
    ).toBeInTheDocument();
  });

  it('clears the violation when exactly one field is filled', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <SchemaFormRenderer
        sections={[
          {
            title: 'Source',
            cardinality_rules: [
              {
                fields: ['source_db_id', 'source_table_id'],
                min: 1,
                max: 1,
                message: 'Specify exactly one source.',
              },
            ],
            fields: [
              { type: 'string', name: 'source_db_id', label: 'Source DB' },
              {
                type: 'string',
                name: 'source_table_id',
                label: 'Source Table',
              },
            ],
          },
        ]}
        onSubmit={() => {}}
      />
    );

    await user.type(screen.getByLabelText('Source DB'), 'mydb');
    await waitFor(() =>
      expect(
        screen.queryByText('Specify exactly one source.')
      ).not.toBeInTheDocument()
    );
  });

  it('re-shows the violation when both fields are filled (exceeds max)', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <SchemaFormRenderer
        sections={[
          {
            title: 'Source',
            cardinality_rules: [
              {
                fields: ['source_db_id', 'source_table_id'],
                min: 1,
                max: 1,
                message: 'Specify exactly one source.',
              },
            ],
            fields: [
              { type: 'string', name: 'source_db_id', label: 'Source DB' },
              {
                type: 'string',
                name: 'source_table_id',
                label: 'Source Table',
              },
            ],
          },
        ]}
        onSubmit={() => {}}
      />
    );

    await user.type(screen.getByLabelText('Source DB'), 'db');
    await user.type(screen.getByLabelText('Source Table'), 'tbl');
    expect(
      await screen.findByText('Specify exactly one source.')
    ).toBeInTheDocument();
  });

  it('blocks submission when a cardinality violation is active', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    renderWithProviders(
      <SchemaFormRenderer
        sections={[
          {
            title: 'Source',
            cardinality_rules: [
              { fields: ['a', 'b'], min: 1, max: 1, message: 'Exactly one.' },
            ],
            fields: [
              { type: 'string', name: 'a', label: 'A' },
              { type: 'string', name: 'b', label: 'B' },
            ],
          },
        ]}
        onSubmit={onSubmit}
      />
    );

    // Both empty → violation active → submit blocked
    await user.click(screen.getByRole('button', { name: /Run/ }));
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it('allows submission when cardinality is satisfied', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    renderWithProviders(
      <SchemaFormRenderer
        sections={[
          {
            title: 'Source',
            cardinality_rules: [
              { fields: ['a', 'b'], min: 1, max: 1, message: 'Exactly one.' },
            ],
            fields: [
              { type: 'string', name: 'a', label: 'A' },
              { type: 'string', name: 'b', label: 'B' },
            ],
          },
        ]}
        onSubmit={onSubmit}
      />
    );

    await user.type(screen.getByLabelText('A'), 'hello');
    await user.click(screen.getByRole('button', { name: /Run/ }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
  });

  it('respects the when predicate — skips rule when guard is inactive', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    renderWithProviders(
      <SchemaFormRenderer
        sections={[
          {
            title: 'Conditional',
            cardinality_rules: [
              {
                when: { truthy: 'enable' },
                fields: ['a', 'b'],
                min: 1,
                max: 1,
                message: 'Exactly one when enabled.',
              },
            ],
            fields: [
              { type: 'bool', name: 'enable', label: 'Enable' },
              { type: 'string', name: 'a', label: 'A' },
              { type: 'string', name: 'b', label: 'B' },
            ],
          },
        ]}
        onSubmit={onSubmit}
      />
    );

    // enable=false → guard inactive → no violation → submit succeeds
    await user.click(screen.getByRole('button', { name: /Run/ }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
  });

  it('uses a default message when none is provided', async () => {
    renderWithProviders(
      <SchemaFormRenderer
        sections={[
          {
            title: 'S',
            cardinality_rules: [{ fields: ['a', 'b'], min: 1 }],
            fields: [
              { type: 'string', name: 'a', label: 'A' },
              { type: 'string', name: 'b', label: 'B' },
            ],
          },
        ]}
        onSubmit={() => {}}
      />
    );

    // Both empty → min=1 violation → default message rendered
    expect(
      await screen.findByText(/At least 1 of these 2 fields/)
    ).toBeInTheDocument();
  });
});

// ── fail_when rules ───────────────────────────────────────────────────────

describe('SchemaFormRenderer — fail_when', () => {
  it('shows a violation banner when the fail_when predicate fires', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <SchemaFormRenderer
        sections={[
          {
            title: 'S',
            fail_when: [
              {
                fail_when: { truthy: 'flag' },
                error_fields: ['flag'],
                message: 'Flag must not be set.',
              },
            ],
            fields: [{ type: 'bool', name: 'flag', label: 'Flag' }],
          },
        ]}
        onSubmit={() => {}}
      />
    );

    // flag=false at mount → predicate inactive → no banner
    expect(screen.queryByText('Flag must not be set.')).toBeNull();

    // flip flag → predicate fires → banner appears
    await user.click(screen.getByLabelText('Flag'));
    expect(
      await screen.findByText('Flag must not be set.')
    ).toBeInTheDocument();
  });

  it('clears the banner when the predicate stops firing', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <SchemaFormRenderer
        sections={[
          {
            title: 'S',
            fail_when: [
              {
                fail_when: { truthy: 'flag' },
                error_fields: ['flag'],
                message: 'Flag must not be set.',
              },
            ],
            fields: [{ type: 'bool', name: 'flag', label: 'Flag' }],
          },
        ]}
        onSubmit={() => {}}
      />
    );

    await user.click(screen.getByLabelText('Flag'));
    expect(
      await screen.findByText('Flag must not be set.')
    ).toBeInTheDocument();

    // flip back → predicate inactive → banner gone
    await user.click(screen.getByLabelText('Flag'));
    await waitFor(() =>
      expect(
        screen.queryByText('Flag must not be set.')
      ).not.toBeInTheDocument()
    );
  });

  it('blocks submission when a fail_when rule is active', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    renderWithProviders(
      <SchemaFormRenderer
        sections={[
          {
            title: 'S',
            fail_when: [
              {
                fail_when: { truthy: 'flag' },
                error_fields: ['flag'],
                message: 'Blocked.',
              },
            ],
            fields: [{ type: 'bool', name: 'flag', label: 'Flag' }],
          },
        ]}
        onSubmit={onSubmit}
      />
    );

    await user.click(screen.getByLabelText('Flag'));
    await user.click(screen.getByRole('button', { name: /Run/ }));
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it('uses a default message when none is provided', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <SchemaFormRenderer
        sections={[
          {
            title: 'S',
            fail_when: [{ fail_when: { truthy: 'x' }, error_fields: [] }],
            fields: [{ type: 'bool', name: 'x', label: 'X' }],
          },
        ]}
        onSubmit={() => {}}
      />
    );

    await user.click(screen.getByLabelText('X'));
    expect(
      await screen.findByText('This combination of values is not allowed.')
    ).toBeInTheDocument();
  });

  it('shows both cardinality and fail_when violations together', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <SchemaFormRenderer
        sections={[
          {
            title: 'S',
            cardinality_rules: [
              { fields: ['a', 'b'], min: 1, message: 'Need at least one.' },
            ],
            fail_when: [
              {
                fail_when: { all_present: ['a', 'b'] },
                error_fields: ['a', 'b'],
                message: 'Cannot have both.',
              },
            ],
            fields: [
              { type: 'string', name: 'a', label: 'A' },
              { type: 'string', name: 'b', label: 'B' },
            ],
          },
        ]}
        onSubmit={() => {}}
      />
    );

    // Both empty → cardinality fires, fail_when inactive
    expect(await screen.findByText('Need at least one.')).toBeInTheDocument();
    expect(screen.queryByText('Cannot have both.')).toBeNull();

    // Fill both → cardinality satisfied, fail_when fires
    await user.type(screen.getByLabelText('A'), 'x');
    await user.type(screen.getByLabelText('B'), 'y');
    await waitFor(() =>
      expect(screen.queryByText('Need at least one.')).not.toBeInTheDocument()
    );
    expect(await screen.findByText('Cannot have both.')).toBeInTheDocument();
  });
});

// ── Conditional required (requires gates) ─────────────────────────────────

describe('SchemaFormRenderer — conditional required', () => {
  const sections: FormSection[] = [
    {
      title: 'Main',
      fields: [
        { type: 'bool', name: 'needsReason', label: 'Needs reason' },
        {
          type: 'string',
          name: 'reason',
          label: 'Reason',
          requires: [{ when: { truthy: 'needsReason' } }],
        },
      ],
    },
  ];

  it('does not block submission when gate is inactive', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={onSubmit} />
    );

    // needsReason=false → requires gate inactive → reason not required
    await user.click(screen.getByRole('button', { name: /Run/ }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
  });

  it('blocks submission and shows required error when gate fires and field is empty', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={onSubmit} />
    );

    await user.click(screen.getByLabelText('Needs reason'));
    await user.click(screen.getByRole('button', { name: /Run/ }));

    expect(onSubmit).not.toHaveBeenCalled();
    expect(await screen.findByText(/Reason is required/)).toBeInTheDocument();
  });

  it('submits when gate fires and field is filled', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={onSubmit} />
    );

    await user.click(screen.getByLabelText('Needs reason'));
    // Use data-testid (set by percona-ui's TextInput) to find the input directly,
    // since the label-for association is briefly stale while isRequired flips.
    await user.type(screen.getByTestId('text-input-reason'), 'because');
    await user.click(screen.getByRole('button', { name: /Run/ }));

    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ reason: 'because', needsReason: true })
    );
  });
});

// ── capabilities.alert_on_fail ────────────────────────────────────────────

describe('SchemaFormRenderer — capabilities.alert_on_fail', () => {
  const sections: FormSection[] = [
    {
      title: 'Main',
      fields: [{ type: 'string', name: 'title', label: 'Title' }],
    },
  ];

  beforeEach(() => {
    useAlertConfigMock.mockReset();
  });

  it('renders AlertOnFailField when capabilities.alert_on_fail is true', () => {
    useAlertConfigMock.mockReturnValue({
      data: { available: true },
      isLoading: false,
      isError: false,
    });
    renderWithProviders(
      <SchemaFormRenderer
        sections={sections}
        onSubmit={() => {}}
        capabilities={{ alert_on_fail: true }}
      />
    );
    expect(
      screen.getByRole('checkbox', { name: /Alert on failure/i })
    ).toBeInTheDocument();
  });

  it('does not render AlertOnFailField when capabilities.alert_on_fail is false', () => {
    renderWithProviders(
      <SchemaFormRenderer
        sections={sections}
        onSubmit={() => {}}
        capabilities={{ alert_on_fail: false }}
      />
    );
    expect(
      screen.queryByRole('checkbox', { name: /Alert on failure/i })
    ).toBeNull();
  });

  it('does not render AlertOnFailField when capabilities is undefined', () => {
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={() => {}} />
    );
    expect(
      screen.queryByRole('checkbox', { name: /Alert on failure/i })
    ).toBeNull();
  });

  it('renders the field disabled when no alert provider is configured', () => {
    useAlertConfigMock.mockReturnValue({
      data: { available: false },
      isLoading: false,
      isError: false,
    });
    renderWithProviders(
      <SchemaFormRenderer
        sections={sections}
        onSubmit={() => {}}
        capabilities={{ alert_on_fail: true }}
      />
    );
    expect(
      screen.getByRole('checkbox', { name: /Alert on failure/i })
    ).toBeDisabled();
  });

  it('submits alert_on_fail: false by default', async () => {
    useAlertConfigMock.mockReturnValue({
      data: { available: true },
      isLoading: false,
      isError: false,
    });
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    renderWithProviders(
      <SchemaFormRenderer
        sections={sections}
        onSubmit={onSubmit}
        capabilities={{ alert_on_fail: true }}
      />
    );

    await user.click(screen.getByRole('button', { name: /Run/ }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ alert_on_fail: false })
    );
  });

  it('submits alert_on_fail: true when user checks the field', async () => {
    useAlertConfigMock.mockReturnValue({
      data: { available: true },
      isLoading: false,
      isError: false,
    });
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    renderWithProviders(
      <SchemaFormRenderer
        sections={sections}
        onSubmit={onSubmit}
        capabilities={{ alert_on_fail: true }}
      />
    );

    await user.click(
      screen.getByRole('checkbox', { name: /Alert on failure/i })
    );
    await user.click(screen.getByRole('button', { name: /Run/ }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ alert_on_fail: true })
    );
  });
});

// ── Unsaved changes guard (integration) ──────────────────────────────────────

const GUARD_SECTIONS: FormSection[] = [
  {
    title: 'Main',
    fields: [{ type: 'string', name: 'title', label: 'Title' }],
  },
];

/**
 * Renders SchemaFormRenderer inside a createMemoryRouter so that useBlocker
 * (and therefore the in-app navigation guard) is active.
 */
function renderWithRouter(
  ui: ReactNode,
  { initialPath = '/form' }: { initialPath?: string } = {}
) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  });
  const router = createMemoryRouter(
    [
      { path: '/form', element: ui },
      { path: '/other', element: <div>Other page</div> },
    ],
    { initialEntries: [initialPath] }
  );
  return {
    router,
    ...render(
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
      </QueryClientProvider>
    ),
  };
}

describe('SchemaFormRenderer — unsaved changes guard', () => {
  beforeEach(() => {
    useAlertConfigMock.mockReset();
    useAlertConfigMock.mockReturnValue({
      data: { available: false },
      isLoading: false,
    });
  });

  it('AC#6 — renders without crash outside a Data Router', () => {
    // When SchemaFormRenderer is mounted without a router context (e.g. inside
    // a unit test that uses renderWithProviders) the in-app blocker is skipped
    // but the component should still mount and render normally.
    renderWithProviders(
      <SchemaFormRenderer sections={GUARD_SECTIONS} onSubmit={() => {}} />
    );
    expect(screen.getByLabelText('Title')).toBeInTheDocument();
    expect(screen.queryByRole('dialog')).toBeNull();
  });

  it('AC#3 — does not block navigation when form is clean', async () => {
    const { router } = renderWithRouter(
      <SchemaFormRenderer sections={GUARD_SECTIONS} onSubmit={() => {}} />
    );

    act(() => {
      router.navigate('/other');
    });

    expect(await screen.findByText('Other page')).toBeInTheDocument();
  });

  it('AC#1 — shows confirmation dialog when dirty form navigates away', async () => {
    const user = userEvent.setup();
    const { router } = renderWithRouter(
      <SchemaFormRenderer sections={GUARD_SECTIONS} onSubmit={() => {}} />
    );

    await user.type(screen.getByLabelText('Title'), 'hello');
    act(() => {
      router.navigate('/other');
    });

    expect(await screen.findByRole('dialog')).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: /unsaved changes/i })
    ).toBeInTheDocument();
    expect(screen.queryByText('Other page')).toBeNull();
  });

  it('AC#1 — Stay button cancels navigation and keeps form state intact', async () => {
    const user = userEvent.setup();
    const { router } = renderWithRouter(
      <SchemaFormRenderer sections={GUARD_SECTIONS} onSubmit={() => {}} />
    );

    await user.type(screen.getByLabelText('Title'), 'hello');
    act(() => {
      router.navigate('/other');
    });

    await screen.findByRole('dialog');
    await user.click(screen.getByRole('button', { name: /stay/i }));

    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
    expect(screen.queryByText('Other page')).toBeNull();
    expect((screen.getByLabelText('Title') as HTMLInputElement).value).toBe(
      'hello'
    );
  });

  it('AC#1 — Discard changes button proceeds with navigation', async () => {
    const user = userEvent.setup();
    const { router } = renderWithRouter(
      <SchemaFormRenderer sections={GUARD_SECTIONS} onSubmit={() => {}} />
    );

    await user.type(screen.getByLabelText('Title'), 'hello');
    act(() => {
      router.navigate('/other');
    });

    await screen.findByRole('dialog');
    await user.click(screen.getByRole('button', { name: /discard/i }));

    expect(await screen.findByText('Other page')).toBeInTheDocument();
  });

  it('AC#4 — does not block navigation after successful submit', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    const { router } = renderWithRouter(
      <SchemaFormRenderer
        sections={GUARD_SECTIONS}
        onSubmit={onSubmit}
        submitLabel="Save"
      />
    );

    await user.type(screen.getByLabelText('Title'), 'hello');
    await user.click(screen.getByRole('button', { name: /save/i }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));

    act(() => {
      router.navigate('/other');
    });

    expect(await screen.findByText('Other page')).toBeInTheDocument();
  });

  it('AC#5 — re-arms guard after failed async mutation (submitError set after submit)', async () => {
    const user = userEvent.setup();

    // FormWithMutationError simulates a parent that sets submitError after its
    // async mutation rejects. onSubmit returns synchronously (no throw), so
    // RHF marks isSubmitSuccessful=true — then submitError is set to re-arm.
    function FormWithMutationError() {
      const [submitError, setSubmitError] = useState<string | null>(null);
      return (
        <SchemaFormRenderer
          sections={GUARD_SECTIONS}
          onSubmit={() => {
            setSubmitError('Mutation failed');
          }}
          submitError={submitError}
          submitLabel="Save"
        />
      );
    }

    const { router } = renderWithRouter(<FormWithMutationError />);

    await user.type(screen.getByLabelText('Title'), 'hello');
    await user.click(screen.getByRole('button', { name: /save/i }));
    await waitFor(() =>
      expect(screen.getByText('Mutation failed')).toBeInTheDocument()
    );
    // Flush effects from reset() so the blocker re-registers on the router
    await act(async () => {});

    // Guard must be re-armed: navigation should now prompt.
    act(() => {
      router.navigate('/other');
    });
    expect(await screen.findByRole('dialog')).toBeInTheDocument();
  });
});

// ── Section-level visibility gates (SEP-1276) ────────────────────────────
//
// Mirrors the MySQL Backups Mydumper/XtraBackup/Binlog regression: a
// FormSection declares a ``forbidden`` gate keyed off a controller field
// elsewhere in the form (in production, ``backup_type``). The section,
// its header, and every child field must disappear from the DOM when the
// gate fires, and the children must be unregistered from RHF so stale
// defaults do not ship in the submission payload.
//
// Mode-owned bool fields are intentionally not field-gated in the
// MySQL Backups schema (module docstring): ``_field_is_present(False)``
// is ``True``, so a per-field ``forbidden=[when != mode]`` would reject
// the legitimate default-False. The whole point of section-level gating
// is to cover those bool fields too — the tests below pin that behaviour
// by hiding a section whose child is a bool with default ``false``.

describe('SchemaFormRenderer — section-level visibility', () => {
  const sections: FormSection[] = [
    {
      title: 'Mode',
      fields: [
        {
          type: 'choice',
          name: 'backup_type',
          label: 'Backup Type',
          choices: [
            { label: 'Mydumper backup', value: 'M' },
            { label: 'XtraBackup backup', value: 'X' },
          ],
        },
      ],
    },
    {
      title: 'Mydumper',
      forbidden: [{ when: { not_equals: { backup_type: 'M' } } }],
      fields: [
        {
          type: 'string',
          name: 'mydumper_extra_args',
          label: 'Mydumper extra args',
        },
        // Mode-owned bool — has no field-level forbidden gate by design.
        {
          type: 'bool',
          name: 'mydumper_dump_triggers',
          label: 'Dump triggers',
          default: false,
        },
      ],
    },
    {
      title: 'XtraBackup',
      forbidden: [{ when: { not_equals: { backup_type: 'X' } } }],
      fields: [
        {
          type: 'string',
          name: 'xtrabackup_extra_args',
          label: 'XtraBackup extra args',
        },
        {
          type: 'bool',
          name: 'xtrabackup_kill_queries',
          label: 'Kill blocking queries',
          default: false,
        },
      ],
    },
  ];

  it('hides every mode section when the controller is unset (initial form load)', () => {
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={() => {}} />
    );
    expect(screen.queryByText('Mydumper')).toBeNull();
    expect(screen.queryByText('XtraBackup')).toBeNull();
    expect(screen.queryByLabelText('Mydumper extra args')).toBeNull();
    expect(screen.queryByLabelText('Dump triggers')).toBeNull();
    expect(screen.queryByLabelText('XtraBackup extra args')).toBeNull();
    expect(screen.queryByLabelText('Kill blocking queries')).toBeNull();
  });

  it('shows only the matching mode section after picking a backup type', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={() => {}} />
    );

    await user.click(screen.getByTestId('radio-option-M'));

    expect(await screen.findByText('Mydumper')).toBeInTheDocument();
    expect(
      await screen.findByLabelText('Mydumper extra args')
    ).toBeInTheDocument();
    expect(await screen.findByLabelText('Dump triggers')).toBeInTheDocument();
    expect(screen.queryByText('XtraBackup')).toBeNull();
    expect(screen.queryByLabelText('XtraBackup extra args')).toBeNull();
    expect(screen.queryByLabelText('Kill blocking queries')).toBeNull();
  });

  it('excludes hidden-section children (incl. mode-owned bools) from the submit payload', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    renderWithProviders(
      <SchemaFormRenderer
        sections={sections}
        onSubmit={onSubmit}
        defaultValues={{
          backup_type: 'M',
          mydumper_extra_args: 'preserved',
          xtrabackup_extra_args: 'should-not-ship',
          xtrabackup_kill_queries: true,
        }}
      />
    );

    await user.click(screen.getByRole('button', { name: /Run/ }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));

    const payload = onSubmit.mock.calls[0]?.[0];
    expect(payload).toMatchObject({
      backup_type: 'M',
      mydumper_extra_args: 'preserved',
    });
    expect(payload).not.toHaveProperty('xtrabackup_extra_args');
    expect(payload).not.toHaveProperty('xtrabackup_kill_queries');
  });

  it('drops stale values when the user toggles mode then submits immediately', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={onSubmit} />
    );

    // Pick XtraBackup, type a value, then flip to Mydumper before submitting.
    await user.click(screen.getByTestId('radio-option-X'));
    const xtraInput = await screen.findByLabelText('XtraBackup extra args');
    await user.type(xtraInput, 'leaked');
    await user.click(screen.getByTestId('radio-option-M'));

    await user.click(screen.getByRole('button', { name: /Run/ }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));

    const payload = onSubmit.mock.calls[0]?.[0];
    expect(payload).toMatchObject({ backup_type: 'M' });
    expect(payload).not.toHaveProperty('xtrabackup_extra_args');
    expect(payload).not.toHaveProperty('xtrabackup_kill_queries');
  });

  it('re-mounts a section with schema defaults after toggling away and back', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={() => {}} />
    );

    // Show Mydumper, type a value, hide via XtraBackup, then re-show Mydumper.
    await user.click(screen.getByTestId('radio-option-M'));
    const mydumperInput = await screen.findByLabelText('Mydumper extra args');
    await user.type(mydumperInput, 'old-value');
    await user.click(screen.getByTestId('radio-option-X'));
    expect(screen.queryByLabelText('Mydumper extra args')).toBeNull();
    await user.click(screen.getByTestId('radio-option-M'));

    const reshown = await screen.findByLabelText('Mydumper extra args');
    expect((reshown as HTMLInputElement).value).toBe('');
  });
});

// ── multi_choice discriminator regression (SEP-1293) ─────────────────────────
//
// Guards against future drift between the hand-maintained `MultiChoiceField.type`
// literal in plugin-schema.ts and the switch branches that consume it. A wrong
// literal silently falls through to `default: return null` — these tests make
// that failure loud.

describe('SchemaFormRenderer — multi_choice discriminator regression (SEP-1293)', () => {
  it('renders the multi-select control for type: multi_choice', () => {
    const sections: FormSection[] = [
      {
        title: 'Upload',
        fields: [
          {
            type: 'multi_choice',
            name: 'upload',
            label: 'Upload providers',
            choices: [{ label: 'S3', value: 'S3' }],
          },
        ],
      },
    ];
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={vi.fn()} />
    );
    expect(
      document.getElementById('mui-component-select-upload')
    ).not.toBeNull();
  });

  it('renders nothing for the stale "multichoice" literal (old wrong discriminator)', () => {
    // Cast bypasses TypeScript: simulates an outdated API response before clients update.
    const sections = [
      {
        title: 'Upload',
        fields: [
          {
            type: 'multichoice',
            name: 'upload',
            label: 'Upload providers',
            choices: [],
          },
        ],
      },
    ] as unknown as FormSection[];
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={vi.fn()} />
    );
    expect(document.getElementById('mui-component-select-upload')).toBeNull();
  });
});

describe('SchemaFormRenderer — multi_choice maxItems', () => {
  it('blocks submission when selection count exceeds maxItems', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    const sections: FormSection[] = [
      {
        title: 'Upload',
        fields: [
          {
            type: 'multi_choice',
            name: 'upload',
            label: 'Upload providers',
            choices: [
              { label: 'S3', value: 'S3' },
              { label: 'Rsync', value: 'RSYNC' },
              { label: 'GCS', value: 'GSUTIL' },
            ],
            max_items: 2,
          },
        ],
      },
    ];
    renderWithProviders(
      <SchemaFormRenderer
        sections={sections}
        onSubmit={onSubmit}
        defaultValues={{ upload: ['S3', 'RSYNC', 'GSUTIL'] }}
      />
    );

    await user.click(screen.getByRole('button', { name: /Run/ }));
    expect(onSubmit).not.toHaveBeenCalled();
    expect(
      await screen.findByText(/Fix the highlighted fields/)
    ).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.getByTestId('select-input-upload')).toHaveAttribute(
        'aria-invalid',
        'true'
      )
    );
  });
});

describe('SchemaFormRenderer — multi_choice default value', () => {
  it('pre-selects a value from the field default', () => {
    const sections: FormSection[] = [
      {
        title: 'Upload',
        fields: [
          {
            type: 'multi_choice',
            name: 'upload',
            label: 'Upload providers',
            default: ['S3'],
            choices: [
              { label: 'S3', value: 'S3' },
              { label: 'Rsync', value: 'RSYNC' },
            ],
          },
        ],
      },
    ];
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={vi.fn()} />
    );
    expect(screen.queryByText('Select…')).toBeNull();
    expect(screen.getByText('S3')).toBeVisible();
  });
});

// ── renderField override hook (SEP-1355) ──────────────────────────────────
//
// A renderField override that writes its value through react-hook-form so the
// value participates in the rule engines and submission. This is the documented
// write-through contract.
function CustomTextWidget({ name }: { name: string }) {
  const { register } = useFormContext();
  return <input aria-label={`custom-${name}`} {...register(name)} />;
}

describe('SchemaFormRenderer — renderField override', () => {
  const sections: FormSection[] = [
    {
      title: 'Main',
      fields: [
        { type: 'string', name: 'mode', label: 'Mode' },
        { type: 'string', name: 'title', label: 'Title' },
      ],
    },
  ];

  // Only customizes `mode`; every other field falls back to renderDefault().
  const renderField: RenderFieldOverride = ({ field, renderDefault }) =>
    field.name === 'mode' ? (
      <CustomTextWidget name={field.name} />
    ) : (
      renderDefault()
    );

  it('renders the override widget for a matching field', () => {
    renderWithProviders(
      <SchemaFormRenderer
        sections={sections}
        onSubmit={vi.fn()}
        renderField={renderField}
      />
    );
    // Override widget present; the default Mode widget is replaced.
    expect(screen.getByLabelText('custom-mode')).toBeInTheDocument();
    expect(screen.queryByLabelText('Mode')).toBeNull();
  });

  it('falls back to the default widget for a non-matching field', () => {
    renderWithProviders(
      <SchemaFormRenderer
        sections={sections}
        onSubmit={vi.fn()}
        renderField={renderField}
      />
    );
    // `title` is not customized → framework default widget renders.
    expect(screen.getByLabelText('Title')).toBeInTheDocument();
  });

  it('does not call the override for a field hidden by its gate', () => {
    const gatedSections: FormSection[] = [
      {
        title: 'Main',
        fields: [
          { type: 'bool', name: 'advanced', label: 'Advanced mode' },
          {
            type: 'string',
            name: 'mode',
            label: 'Mode',
            forbidden: [{ when: { falsy: 'advanced' } }],
          },
        ],
      },
    ];
    const override = vi.fn(renderField);
    renderWithProviders(
      <SchemaFormRenderer
        sections={gatedSections}
        onSubmit={vi.fn()}
        renderField={override}
      />
    );
    // advanced=false (default) → forbidden gate fires → mode hidden → override
    // never invoked for it, and the override widget is absent.
    expect(screen.queryByLabelText('custom-mode')).toBeNull();
    expect(override).not.toHaveBeenCalledWith(
      expect.objectContaining({
        field: expect.objectContaining({ name: 'mode' }),
      })
    );
  });

  it('passes the gate-resolved required flag to the override', async () => {
    const user = userEvent.setup();
    const seenRequired: boolean[] = [];
    const captureOverride: RenderFieldOverride = ({ field, renderDefault }) => {
      if (field.name === 'mode') {
        seenRequired.push(Boolean(field.required));
        return <CustomTextWidget name={field.name} />;
      }
      return renderDefault();
    };
    const gatedSections: FormSection[] = [
      {
        title: 'Main',
        fields: [
          { type: 'bool', name: 'advanced', label: 'Advanced mode' },
          {
            type: 'string',
            name: 'mode',
            label: 'Mode',
            // becomes required only when `advanced` is truthy
            requires: [{ when: { truthy: 'advanced' } }],
          },
        ],
      },
    ];
    renderWithProviders(
      <SchemaFormRenderer
        sections={gatedSections}
        onSubmit={vi.fn()}
        renderField={captureOverride}
      />
    );
    // advanced=false at mount → resolved required=false
    expect(seenRequired.at(-1)).toBe(false);
    // flip advanced → requires gate fires → resolved required=true reaches override
    await user.click(screen.getByLabelText('Advanced mode'));
    await waitFor(() => expect(seenRequired.at(-1)).toBe(true));
  });

  it('write-through: overridden value participates in the fail_when engine', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <SchemaFormRenderer
        sections={[
          {
            title: 'Main',
            fail_when: [
              {
                fail_when: { truthy: 'mode' },
                error_fields: ['mode'],
                message: 'Mode must be empty.',
              },
            ],
            fields: [{ type: 'string', name: 'mode', label: 'Mode' }],
          },
        ]}
        onSubmit={vi.fn()}
        renderField={renderField}
      />
    );
    // mode empty at mount → predicate inactive
    expect(screen.queryByText('Mode must be empty.')).toBeNull();
    // type into the OVERRIDDEN widget → value reaches RHF watch state → fail_when fires
    await user.type(screen.getByLabelText('custom-mode'), 'danger');
    expect(await screen.findByText('Mode must be empty.')).toBeInTheDocument();
  });

  it('write-through: overridden value is included in the submit payload', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    renderWithProviders(
      <SchemaFormRenderer
        sections={sections}
        onSubmit={onSubmit}
        renderField={renderField}
      />
    );
    await user.type(screen.getByLabelText('custom-mode'), 'fast');
    await user.click(screen.getByRole('button', { name: /Run/ }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ mode: 'fast' })
    );
  });

  it('falls back to the default widget when the override returns a nullish value', () => {
    // Override opts out of every field by returning undefined → defaults render.
    const optOut: RenderFieldOverride = () => undefined;
    renderWithProviders(
      <SchemaFormRenderer
        sections={sections}
        onSubmit={vi.fn()}
        renderField={optOut}
      />
    );
    expect(screen.getByLabelText('Mode')).toBeInTheDocument();
    expect(screen.getByLabelText('Title')).toBeInTheDocument();
    expect(screen.queryByLabelText('custom-mode')).toBeNull();
  });

  it('renders identically when no override is supplied (no regression)', () => {
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={vi.fn()} />
    );
    // Default widgets for both fields; no custom widget.
    expect(screen.getByLabelText('Mode')).toBeInTheDocument();
    expect(screen.getByLabelText('Title')).toBeInTheDocument();
    expect(screen.queryByLabelText('custom-mode')).toBeNull();
  });
});

describe('SchemaFormRenderer — one_of groups', () => {
  const sections: FormSection[] = [
    {
      title: 'Source',
      fields: [
        {
          type: 'one_of',
          name: 'source',
          label: 'Source',
          description: 'Choose how to specify source rows.',
          discriminator: 'source.mode',
          default: 'schema',
          branches: [
            {
              value: 'schema',
              label: 'Schema & Table',
              fields: [
                {
                  type: 'string',
                  name: 'source.source_db_id',
                  label: 'Source Schema',
                },
              ],
            },
            {
              value: 'query',
              label: 'Custom Query',
              fields: [
                {
                  type: 'string',
                  name: 'source.source_query',
                  label: 'Source Query',
                },
              ],
            },
          ],
        },
      ],
    },
  ];

  it('renders segmented control, helper text, and the default branch fields', () => {
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={() => {}} />
    );
    expect(screen.getByTestId('one-of-source')).toBeInTheDocument();
    expect(
      screen.getByText('Choose how to specify source rows.')
    ).toBeInTheDocument();
    expect(
      screen.getByTestId('text-input-source.source_db_id')
    ).toBeInTheDocument();
    expect(screen.queryByTestId('text-input-source.source_query')).toBeNull();
  });

  it('swaps visible branch fields and omits inactive values on submit', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={onSubmit} />
    );

    await user.click(screen.getByTestId('one-of-option-query'));
    expect(
      await screen.findByTestId('text-input-source.source_query')
    ).toBeInTheDocument();
    expect(screen.queryByTestId('text-input-source.source_db_id')).toBeNull();

    await user.type(
      screen.getByTestId('text-input-source.source_query'),
      'SELECT 1'
    );
    await user.click(screen.getByRole('button', { name: 'Run' }));

    await waitFor(() => expect(onSubmit).toHaveBeenCalled());
    expect(onSubmit).toHaveBeenCalledWith({
      source: { mode: 'query', source_query: 'SELECT 1' },
    });
  });

  it('unregisters one_of discriminator and leaf values when a section becomes hidden', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    const gatedSections: FormSection[] = [
      {
        title: 'Options',
        fields: [
          {
            type: 'bool',
            name: 'include_target',
            label: 'Include Target',
            default: true,
          },
        ],
      },
      {
        title: 'Target',
        forbidden: [{ when: { not_equals: { include_target: true } } }],
        fields: [
          {
            type: 'one_of',
            name: 'target',
            label: 'Target',
            discriminator: 'target_mode',
            default: 'service',
            branches: [
              {
                value: 'service',
                label: 'Service',
                fields: [
                  { type: 'string', name: 'target', label: 'Service Target' },
                ],
              },
              {
                value: 'schema',
                label: 'Schema',
                fields: [
                  { type: 'string', name: 'target', label: 'Schema Target' },
                ],
              },
            ],
          },
        ],
      },
    ];
    renderWithProviders(
      <SchemaFormRenderer
        sections={gatedSections}
        onSubmit={onSubmit}
        defaultValues={{
          include_target: true,
          target_mode: 'schema',
          target: 'inventory',
        }}
      />
    );

    await user.click(screen.getByLabelText('Include Target'));
    await waitFor(() => expect(screen.queryByText('Target')).toBeNull());

    await user.click(screen.getByRole('button', { name: 'Run' }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    const payload = onSubmit.mock.calls[0]?.[0];
    expect(payload).toMatchObject({ include_target: false });
    expect(payload).not.toHaveProperty('target');
    expect(payload).not.toHaveProperty('target_mode');
  });
});
