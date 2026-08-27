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

import { memo } from 'react';
import Box from '@mui/material/Box';
import { FieldRenderer } from './fields';
import { useConditionalField } from './hooks/useConditionalField';
import type { PluginField, RenderFieldOverride } from './types';

export const ConditionalFieldSlot = memo(function ConditionalFieldSlot({
  field,
  renderField,
}: {
  field: PluginField;
  renderField?: RenderFieldOverride;
}) {
  const { isHidden, isRequired } = useConditionalField(field);

  if (isHidden) {
    return null;
  }

  const resolvedField =
    Boolean(field.required) !== isRequired
      ? { ...field, required: isRequired }
      : field;
  const renderDefault = () => <FieldRenderer field={resolvedField} />;

  return (
    <Box sx={{ mb: 2 }} data-field-name={field.name}>
      {renderField?.({ field: resolvedField, renderDefault }) ??
        renderDefault()}
    </Box>
  );
});
