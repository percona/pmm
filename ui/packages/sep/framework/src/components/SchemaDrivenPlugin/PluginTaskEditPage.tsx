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

import { useMemo, useState } from 'react';
import { Navigate, useNavigate, useParams } from 'react-router-dom';
import Box from '@mui/material/Box';
import IconButton from '@mui/material/IconButton';
import Skeleton from '@mui/material/Skeleton';
import Typography from '@mui/material/Typography';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import { useSnackbar } from 'notistack';
import {
  usePluginTask,
  useUpdatePluginTask,
  type PluginSchema,
} from '@sep/api';
import {
  SchemaFormRenderer,
  coerceFormValues,
  flattenSectionFields,
  getAtPath,
  setAtPath,
} from '../SchemaFormRenderer';
import type {
  FormSection,
  RenderFieldOverride,
} from '../SchemaFormRenderer/types';
import type { RenderFormSlot } from './types';
import { getStoredForm } from './storedForm';
import {
  EMPTY_SUBMIT_ERROR,
  mapSubmitError,
  type SubmitErrorState,
} from './submitErrorMapping';

/**
 * Normalize choice and multi-choice default values so that case differences
 * between stored enum values (e.g. ``"rsync"`` from Python's StrEnum auto())
 * and schema choice values (e.g. ``"RSYNC"``) do not produce empty selections.
 *
 * For each ``choice`` / ``multi_choice`` field in ``sections``, look up the
 * stored value(s) case-insensitively against the field's canonical choice
 * values and replace them with the canonical form. Unrecognised values are
 * left as-is so the form's own validation surfaces them rather than silently
 * discarding them.
 */
export function normalizeChoiceDefaults(
  form: Record<string, unknown>,
  sections: FormSection[]
): Record<string, unknown> {
  const out = { ...form };
  for (const field of flattenSectionFields(sections)) {
    if (field.type !== 'choice' && field.type !== 'multi_choice') {
      continue;
    }
    const choiceMap = new Map(
      field.choices.map((c) => [c.value.toLowerCase(), c.value])
    );
    // Path-aware: `flattenSectionFields` also returns one-of branch fields,
    // whose names are dotted paths (`source.mode`) stored nested in the form.
    // `setAtPath` shallow-clones the intermediates it walks, so the shallow
    // copy above is enough to leave `form` untouched.
    const raw = getAtPath(out, field.name);
    if (field.type === 'multi_choice' && Array.isArray(raw)) {
      setAtPath(
        out,
        field.name,
        raw.map((v) => {
          const canonical =
            typeof v === 'string' ? choiceMap.get(v.toLowerCase()) : undefined;
          return canonical ?? v;
        })
      );
    } else if (field.type === 'choice' && typeof raw === 'string') {
      const canonical = choiceMap.get(raw.toLowerCase());
      if (canonical !== undefined) {
        setAtPath(out, field.name, canonical);
      }
    }
  }
  return out;
}

/**
 * The create-form field that names the task. It is load-bearing identity — the
 * ``PUT /{task_name}`` route key and the join key for schedules / periodic
 * tasks — so generic edit renders it read-only and never submits a different
 * value; renaming is a separate concern.
 */
const TASK_NAME_FIELD = 'task_name';

/**
 * Generic in-place edit page for a schema-driven task, prefilled from the task's
 * stored create-form body (``data['_form']``) and submitting the already-derived
 * ``PUT``. Serves every ``TaskExecutionApp`` with no per-app code; ``task_name``
 * is immutable here (see {@link TASK_NAME_FIELD}).
 */
export function PluginTaskEditPage({
  schema,
  pluginName,
  mockTasks,
  renderField,
  renderEditForm,
}: {
  schema: PluginSchema;
  pluginName: string;
  mockTasks?: Record<string, unknown>[];
  renderField?: RenderFieldOverride;
  renderEditForm?: RenderFormSlot;
}) {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const { enqueueSnackbar } = useSnackbar();
  const updateTask = useUpdatePluginTask(pluginName, mockTasks);
  const { data: task, isLoading } = usePluginTask(pluginName, id, mockTasks);

  const storedForm = getStoredForm(task);

  const editableSections = useMemo(
    () =>
      (schema.forms ?? [])
        .map((section) => ({
          ...section,
          fields: section.fields.filter(
            (field) => field.name !== TASK_NAME_FIELD
          ),
        }))
        .filter((section) => section.fields.length > 0),
    [schema.forms]
  );

  const normalizedDefaults = useMemo(
    () =>
      storedForm
        ? normalizeChoiceDefaults(storedForm, editableSections)
        : storedForm,
    [storedForm, editableSections]
  );

  const [{ submitError, fieldErrors }, setSubmitErrorState] =
    useState<SubmitErrorState>(EMPTY_SUBMIT_ERROR);

  const handleSubmit = (data: Record<string, unknown>) => {
    if (!id) {
      return;
    }
    // Clear any prior failure so the banner / inline errors reset before the
    // new attempt resolves.
    setSubmitErrorState(EMPTY_SUBMIT_ERROR);
    updateTask.mutate(
      {
        taskId: id,
        // Coerce at the submit boundary so whole-form slots that bypass
        // SchemaFormRenderer's internal coercion still send a backend-ready
        // payload, then pin ``task_name`` to the route id so it cannot change.
        values: {
          ...coerceFormValues(data, flattenSectionFields(editableSections)),
          [TASK_NAME_FIELD]: id,
        },
      },
      {
        onSuccess: () => {
          enqueueSnackbar(`${schema.display_name} task updated`, {
            variant: 'success',
          });
          navigate('..', { relative: 'path' });
        },
        onError: (error: unknown) => {
          const message =
            error instanceof Error ? error.message : 'Failed to update';
          // Transient toast is unchanged; 422s additionally map to a persistent
          // banner plus inline per-field errors.
          enqueueSnackbar(message, { variant: 'error' });
          setSubmitErrorState(mapSubmitError(error, editableSections, message));
        },
      }
    );
  };

  if (isLoading) {
    return (
      <Box>
        <Skeleton variant="text" width={300} height={40} />
        <Skeleton variant="rectangular" height={200} sx={{ mt: 2 }} />
      </Box>
    );
  }

  if (!task) {
    return (
      <Box sx={{ py: 2 }}>
        <Typography variant="h5">Not found</Typography>
      </Box>
    );
  }

  // A task with no stored form input cannot be edited generically; the Edit
  // entry point is only linked when the form is present, so this is defensive
  // against direct navigation.
  if (!storedForm) {
    return <Navigate to=".." relative="path" replace />;
  }

  return (
    <Box>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 3 }}>
        <IconButton onClick={() => navigate('..', { relative: 'path' })}>
          <ArrowBackIcon />
        </IconButton>
        <Typography variant="h4">
          Edit {schema.display_name}: {id}
        </Typography>
      </Box>

      {renderEditForm?.({
        sections: editableSections,
        onSubmit: handleSubmit,
        loading: updateTask.isPending,
        defaultValues: normalizedDefaults,
        capabilities: schema.capabilities,
        renderField,
        submitError,
        fieldErrors,
      }) ?? (
        <SchemaFormRenderer
          sections={editableSections}
          onSubmit={handleSubmit}
          loading={updateTask.isPending}
          submitLabel="Save"
          submitError={submitError}
          fieldErrors={fieldErrors}
          defaultValues={normalizedDefaults}
          capabilities={schema.capabilities}
          renderField={renderField}
        />
      )}
    </Box>
  );
}
