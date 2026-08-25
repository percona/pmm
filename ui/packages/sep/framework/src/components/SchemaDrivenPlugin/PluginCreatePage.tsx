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
import { useNavigate, useParams } from 'react-router-dom';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import IconButton from '@mui/material/IconButton';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import { useSnackbar } from 'notistack';
import {
  useAuth,
  useCreatePluginEntity,
  useCreatePluginTask,
  type PluginSchema,
} from '@sep/api';
import { ReadOnlyNotice } from '../ReadOnlyNotice';
import {
  SchemaFormRenderer,
  coerceFormValues,
  flattenSectionFields,
} from '../SchemaFormRenderer';
import type { RenderFieldOverride } from '../SchemaFormRenderer/types';
import type { RenderFormSlot } from './types';
import {
  EMPTY_SUBMIT_ERROR,
  mapSubmitError,
  type SubmitErrorState,
} from './submitErrorMapping';

interface PluginCreatePageProps {
  schema: PluginSchema;
  pluginName: string;
  mockTasks?: Record<string, unknown>[];
  mockEntityItems?: Record<string, Record<string, unknown>[]>;
  /** Per-field widget override threaded to the form renderer. */
  renderField?: RenderFieldOverride;
  /** Whole-form slot replacing the form body while keeping chrome / mutation / snackbars. */
  renderCreateForm?: RenderFormSlot;
}

export function PluginCreatePage({
  schema,
  pluginName,
  mockTasks,
  mockEntityItems,
  renderField,
  renderCreateForm,
}: PluginCreatePageProps) {
  const navigate = useNavigate();
  const { canMutate } = useAuth();
  const { entityName } = useParams<{ entityName?: string }>();
  const entitySchema = useMemo(
    () => schema.entities?.find((e) => e.name === entityName),
    [schema.entities, entityName]
  );
  const multi = Boolean(schema.entities?.length && entityName && entitySchema);
  const { enqueueSnackbar } = useSnackbar();
  const createTask = useCreatePluginTask(pluginName, mockTasks);
  const createEntity = useCreatePluginEntity(
    pluginName,
    entityName ?? '',
    multi ? mockEntityItems?.[entityName!] : undefined
  );

  const create = multi ? createEntity : createTask;
  const title = multi ? entitySchema!.display_name : schema.display_name;
  const sections = multi ? entitySchema!.forms : schema.forms!;
  const capabilities = multi ? undefined : schema.capabilities;

  const [{ submitError, fieldErrors }, setSubmitErrorState] =
    useState<SubmitErrorState>(EMPTY_SUBMIT_ERROR);

  const handleSubmit = (data: Record<string, unknown>) => {
    // Clear any prior failure so the banner / inline errors reset before the
    // new attempt resolves.
    setSubmitErrorState(EMPTY_SUBMIT_ERROR);
    // Coerce at the submit boundary so whole-form slots that build their own
    // form (bypassing SchemaFormRenderer's internal coercion) still send a
    // backend-ready payload. Idempotent for the default SchemaFormRenderer
    // path, which has already coerced.
    create.mutate(coerceFormValues(data, flattenSectionFields(sections)), {
      onSuccess: (created) => {
        enqueueSnackbar(`${title} created`, { variant: 'success' });
        // A post-create connectivity check only rides the create response; the
        // detail/list model omits it. When present, land on the new task's
        // detail page and carry the warning via navigation state so it surfaces
        // there. Entities and warning-free creates go to the list.
        const response = (created ?? {}) as Record<string, unknown>;
        const warning = response.connectivity_warning;
        const name =
          typeof response.name === 'string' ? response.name : undefined;
        if (!multi && warning && name) {
          navigate(`../task/${encodeURIComponent(name)}`, {
            relative: 'path',
            state: { connectivityWarning: warning },
          });
          return;
        }
        navigate('..', { relative: 'path' });
      },
      onError: (error: unknown) => {
        // Reported by the form's own persistent banner (plus inline per-field
        // errors for a 422) and by nothing else: one signal per failure, and one
        // that does not depend on the host mounting a snackbar provider.
        setSubmitErrorState(
          mapSubmitError(error, sections, 'Failed to create')
        );
      },
    });
  };

  // Creating is a mutation, so the whole page is the control: a read-only
  // session gets the guard state instead of a form that would answer 403. The
  // back chrome stays — nothing links here for such a session, so anyone who
  // arrives did so by URL and needs a way out.
  if (!canMutate) {
    return (
      <Box>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 3 }}>
          <IconButton
            onClick={() => navigate('..', { relative: 'path' })}
            aria-label={`Back to ${title}`}
          >
            <ArrowBackIcon />
          </IconButton>
          <Typography variant="h4">New {title}</Typography>
        </Box>
        <ReadOnlyNotice
          action={`create ${title}`}
          testId="plugin-create-read-only"
        />
      </Box>
    );
  }

  return (
    <Box>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 3 }}>
        <IconButton onClick={() => navigate('..', { relative: 'path' })}>
          <ArrowBackIcon />
        </IconButton>
        <Typography variant="h4">New {title}</Typography>
      </Box>

      {renderCreateForm?.({
        sections,
        onSubmit: handleSubmit,
        loading: create.isPending,
        capabilities,
        renderField,
        submitError,
        fieldErrors,
      }) ?? (
        <SchemaFormRenderer
          sections={sections}
          onSubmit={handleSubmit}
          loading={create.isPending}
          submitLabel={`Create ${title}`}
          submitError={submitError}
          fieldErrors={fieldErrors}
          capabilities={capabilities}
          renderField={renderField}
        />
      )}
    </Box>
  );
}
