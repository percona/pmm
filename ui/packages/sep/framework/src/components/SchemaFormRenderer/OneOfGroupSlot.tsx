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

import { memo, useEffect, useMemo, type MouseEvent } from 'react';
import { useFormContext, useWatch } from 'react-hook-form';
import Box from '@mui/material/Box';
import ToggleButton from '@mui/material/ToggleButton';
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup';
import Typography from '@mui/material/Typography';
import type { OneOfGroup } from '@sep/api';
import { ConditionalFieldSlot } from './ConditionalFieldSlot';
import type { RenderFieldOverride } from './types';

export interface OneOfGroupSlotProps {
  group: OneOfGroup;
  renderField?: RenderFieldOverride;
}

export const OneOfGroupSlot = memo(function OneOfGroupSlot({
  group,
  renderField,
}: OneOfGroupSlotProps) {
  const { control, setValue, unregister } =
    useFormContext<Record<string, unknown>>();

  const watchedMode = useWatch({
    control,
    name: group.discriminator,
  });

  const activeValue = String(
    watchedMode ?? group.default ?? group.branches[0]?.value ?? ''
  );

  const activeBranch = useMemo(
    () =>
      group.branches.find((branch) => branch.value === activeValue) ??
      group.branches[0],
    [group.branches, activeValue]
  );

  useEffect(() => {
    for (const branch of group.branches) {
      if (branch.value === activeValue) {
        continue;
      }
      for (const leaf of branch.fields) {
        unregister(leaf.name);
      }
    }
  }, [activeValue, group.branches, unregister]);

  const handleModeChange = (
    _event: MouseEvent<HTMLElement>,
    next: string | null
  ) => {
    if (!next || next === activeValue) {
      return;
    }
    setValue(group.discriminator, next, {
      shouldDirty: true,
      shouldTouch: true,
    });
  };

  return (
    <Box sx={{ mb: 2 }} data-testid={`one-of-${group.name}`}>
      <Typography
        component="div"
        variant="subtitle2"
        sx={{ fontWeight: 600, mb: 0.5 }}
      >
        {group.label}
      </Typography>
      {group.description ? (
        <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
          {group.description}
        </Typography>
      ) : null}
      <ToggleButtonGroup
        exclusive
        value={activeValue}
        onChange={handleModeChange}
        aria-label={group.label}
        sx={{ mb: 2, display: 'flex', flexWrap: 'wrap' }}
      >
        {group.branches.map((branch) => (
          <ToggleButton
            key={branch.value}
            value={branch.value}
            data-testid={`one-of-option-${branch.value}`}
            sx={{ flex: '1 1 auto' }}
          >
            {branch.label}
          </ToggleButton>
        ))}
      </ToggleButtonGroup>
      {activeBranch.fields.map((field) => (
        <ConditionalFieldSlot
          key={field.name}
          field={field}
          renderField={renderField}
        />
      ))}
    </Box>
  );
});
