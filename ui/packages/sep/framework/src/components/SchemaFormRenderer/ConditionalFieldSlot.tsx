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

import { memo, useMemo } from 'react';
import Box from '@mui/material/Box';
import { FieldRenderer } from './fields';
import { collectParentNames, isFullRowField } from './fieldLayout';
import { useFormFields } from './formFieldsContext';
import { useConditionalField } from './hooks/useConditionalField';
import type { PluginField, RenderFieldOverride } from './types';

export const ConditionalFieldSlot = memo(function ConditionalFieldSlot({
  field,
  renderField,
}: {
  field: PluginField;
  renderField?: RenderFieldOverride;
}) {
  const { isHidden, isRequired, isDisabled } = useConditionalField(field);
  const formFields = useFormFields();
  const parentNames = useMemo(
    () => collectParentNames(formFields),
    [formFields]
  );
  const parentLabel = field.parent
    ? (formFields.find((f) => f.name === field.parent)?.label ?? field.parent)
    : undefined;

  if (isHidden) {
    return null;
  }

  const resolvedField =
    Boolean(field.required) !== isRequired
      ? { ...field, required: isRequired }
      : field;
  const renderDefault = () => <FieldRenderer field={resolvedField} />;
  const content =
    renderField?.({ field: resolvedField, renderDefault }) ?? renderDefault();

  const gridColumn = isFullRowField(field, parentNames) ? '1 / -1' : 'auto';

  if (!field.parent) {
    return (
      <Box sx={{ mb: 2, gridColumn }} data-field-name={field.name}>
        {content}
      </Box>
    );
  }

  // A parented field renders as a `fieldset`, so one `disabled` attribute
  // reaches every control inside it — no field renderer has to know it is
  // being disabled, and a `renderField` override inherits the behaviour for
  // free. The rule down the left edge is what makes the nesting readable.
  return (
    <Box
      component="fieldset"
      disabled={isDisabled}
      aria-disabled={isDisabled || undefined}
      data-field-name={field.name}
      data-parent-field={field.parent}
      sx={{
        gridColumn,
        m: 0,
        mb: 2,
        ml: 1,
        p: 0,
        pl: 2,
        border: 0,
        borderLeft: '2px solid',
        borderLeftColor: 'divider',
        // A fieldset sizes to `min-content` by default and would refuse to
        // shrink inside the form's flow.
        minWidth: 0,
        minInlineSize: 0,
        opacity: isDisabled ? 0.6 : 1,
      }}
    >
      {/*
        Names the group for a screen reader, which otherwise reaches a set of
        controls it cannot focus with nothing said about why. Visually the
        indent rule and the greying already carry it.
      */}
      <Box
        component="legend"
        sx={{
          position: 'absolute',
          width: 1,
          height: 1,
          overflow: 'hidden',
          clip: 'rect(0 0 0 0)',
          whiteSpace: 'nowrap',
        }}
      >
        {isDisabled
          ? `Requires ${parentLabel}, which is off`
          : `Part of ${parentLabel}`}
      </Box>
      {content}
    </Box>
  );
});
