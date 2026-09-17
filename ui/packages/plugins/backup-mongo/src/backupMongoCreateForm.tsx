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

import { useEffect, useRef, type ReactNode } from 'react';
import { useFormContext } from 'react-hook-form';
import {
  useResolvedServiceField,
  type RenderFieldOverride,
} from '@sep/framework';
import {
  isAutoMongoBackupTaskName,
  suggestMongoBackupTaskName,
} from './suggestMongoBackupTaskName';

/**
 * Keep ``task_name`` filled with a service+timestamp suggestion while the user
 * has not edited it. Mounted via the create-form ``renderField`` override so it
 * runs inside SchemaFormRenderer's FormProvider.
 *
 * Resolves scalar ``service_id`` values through {@link useResolvedServiceField}
 * (same path as host cascade) so the suggestion uses the service name rather
 * than falling back to the generic ``mongodb`` slug.
 *
 * Exported for component tests that assert override-safe suggestion behaviour.
 */
export function SuggestedTaskNameEffect({
  schemaDefault,
}: {
  schemaDefault?: string;
}) {
  const { setValue, getValues } = useFormContext();
  const { service, resetKey, isResolving } =
    useResolvedServiceField('service_id');
  const autoRef = useRef<string | undefined>(undefined);
  const prevServiceKeyRef = useRef<string | undefined>(undefined);

  useEffect(() => {
    // Wait for scalar rehydration before advancing the reset-key cursor —
    // otherwise a first paint with only an id would lock in the generic slug
    // and skip the name-based suggestion once services arrive.
    if (isResolving) {
      return;
    }

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
    if (!isAutoMongoBackupTaskName(current, autoRef.current, schemaDefault)) {
      return;
    }

    const next = suggestMongoBackupTaskName(service?.name);
    setValue('task_name', next, { shouldDirty: false, shouldValidate: false });
    autoRef.current = next;
  }, [resetKey, schemaDefault, service, isResolving, setValue, getValues]);

  return null;
}

/**
 * Create-form field override: suggest a task name for ``task_name``, leave
 * every other field to the framework default widget.
 */
export const backupMongoCreateRenderField: RenderFieldOverride = ({
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
