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

import { useLayoutEffect, useRef, type ReactNode } from 'react';
import { Box } from '@mui/material';
import { useFormContext, useWatch } from 'react-hook-form';
import {
  SchemaFormRenderer,
  type RenderFieldOverride,
  type RenderFormSlot,
} from '@sep/framework';
import {
  isAutoMongoRestoreTaskName,
  suggestMongoRestoreTaskName,
} from './suggestMongoRestoreTaskName';

function serviceDisplayName(value: unknown): string | undefined {
  if (value && typeof value === 'object' && 'name' in value) {
    const name = (value as { name: unknown }).name;
    return typeof name === 'string' ? name : undefined;
  }
  return undefined;
}

function serviceResetKey(value: unknown): string {
  if (value && typeof value === 'object' && 'id' in value) {
    return `id:${String((value as { id: unknown }).id ?? 'none')}`;
  }
  if (typeof value === 'number' && Number.isFinite(value)) {
    return `id:${value}`;
  }
  if (typeof value === 'string' && value.trim() !== '') {
    return `raw:${value.trim()}`;
  }
  return 'id:none';
}

/**
 * Keep ``task_name`` filled with a service+timestamp suggestion while the user
 * has not edited it. Mounted via the create-form ``renderField`` override so it
 * runs inside SchemaFormRenderer's FormProvider.
 */
function SuggestedTaskNameEffect({
  schemaDefault,
}: {
  schemaDefault?: string;
}) {
  const { setValue, getValues } = useFormContext();
  const service = useWatch({ name: 'service_id' });
  const resetKey = serviceResetKey(service);
  const autoRef = useRef<string | undefined>(undefined);
  const prevServiceKeyRef = useRef<string | undefined>(undefined);

  // useLayoutEffect so the stamped name is visible on first paint; useEffect
  // alone left the static schema default (`mongodb-restore`) on screen when the
  // Controller committed before the post-paint setValue.
  useLayoutEffect(() => {
    const isFirst = prevServiceKeyRef.current === undefined;
    const serviceChanged =
      prevServiceKeyRef.current !== undefined &&
      prevServiceKeyRef.current !== resetKey;
    prevServiceKeyRef.current = resetKey;

    // Only act on mount or when the upstream service changes — avoid minting a
    // new timestamp on every incidental re-render.
    if (!isFirst && !serviceChanged) {
      return;
    }

    const current = String(getValues('task_name') ?? '');
    if (!isAutoMongoRestoreTaskName(current, autoRef.current, schemaDefault)) {
      return;
    }

    const next = suggestMongoRestoreTaskName(serviceDisplayName(service));
    setValue('task_name', next, { shouldDirty: false, shouldValidate: false });
    autoRef.current = next;
  }, [resetKey, schemaDefault, service, setValue, getValues]);

  return null;
}

/**
 * Create-form field override: suggest a task name for ``task_name``, leave
 * every other field to the framework default widget.
 */
export const restoreMongoCreateRenderField: RenderFieldOverride = ({
  field,
  renderDefault,
}): ReactNode => {
  if (field.name !== 'task_name') {
    return renderDefault();
  }
  const schemaDefault =
    typeof field.default === 'string' ? field.default : undefined;

  return (
    <>
      <SuggestedTaskNameEffect schemaDefault={schemaDefault} />
      {renderDefault()}
    </>
  );
};

/**
 * Render the shared MongoDB restore form layout for a create or edit slot.
 *
 * Accordion padding is scoped via the wrapper class (not in SchemaFormRenderer)
 * so other apps' collapsible sections keep their existing spacing.
 */
function renderRestoreMongoForm(
  {
    sections,
    onSubmit,
    loading,
    capabilities,
    renderField,
    submitError,
    fieldErrors,
  }: Parameters<RenderFormSlot>[0],
  {
    defaultValues,
    submitLabel,
  }: { defaultValues?: Record<string, unknown>; submitLabel: string }
): ReactNode {
  return (
    /*
     * The accordion is rendered inside SchemaFormRenderer, so its padding is
     * reached by descendant selector rather than a prop. This was a stylesheet
     * -- the only first-party .css in pmm/ui -- for exactly that reason; `sx`
     * does the same job without a global class name, and takes the values off
     * the theme's spacing scale.
     */
    <Box sx={{ '& .MuiAccordionDetails-root': { pt: 1, px: 2, pb: 2 } }}>
      <SchemaFormRenderer
        sections={sections}
        onSubmit={onSubmit}
        loading={loading}
        capabilities={capabilities}
        renderField={renderField}
        defaultValues={defaultValues}
        submitLabel={submitLabel}
        submitError={submitError}
        fieldErrors={fieldErrors}
      />
    </Box>
  );
}

/**
 * Create-form slot: seed ``task_name`` with a stamped suggestion so the field
 * does not flash (or stick on) the static schema default ``mongodb-restore``.
 */
export const restoreMongoCreateForm: RenderFormSlot = (props) =>
  renderRestoreMongoForm(props, {
    defaultValues: { task_name: suggestMongoRestoreTaskName() },
    submitLabel: 'Create MongoDB Restores',
  });

/**
 * Edit-form slot: reuse the custom restore layout while deferring to the edit
 * page's stored-form ``defaultValues`` and a neutral ``Save`` label. The edit
 * page strips ``task_name`` from the sections, so the create-only task-name
 * suggestion never mounts here.
 */
export const restoreMongoEditForm: RenderFormSlot = (props) =>
  renderRestoreMongoForm(props, {
    defaultValues: props.defaultValues,
    submitLabel: 'Save',
  });
