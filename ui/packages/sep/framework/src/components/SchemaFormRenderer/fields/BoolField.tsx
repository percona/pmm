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

import { useFormContext } from 'react-hook-form';
import Box from '@mui/material/Box';
import { SwitchInput } from '@percona/percona-ui';
import { FieldHelpIcon } from '../FieldLabelWithHelp';
import type { BoolField as BoolFieldType } from '../types';

interface BoolFieldProps {
  field: BoolFieldType;
}

export function BoolField({ field }: BoolFieldProps) {
  const { control } = useFormContext();
  return (
    <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 0.5 }}>
      <Box sx={{ minWidth: 0 }}>
        <SwitchInput
          name={field.name}
          label={field.label}
          labelCaption={field.description}
          control={control}
        />
      </Box>
      {field.description ? (
        <FieldHelpIcon description={field.description} label={field.label} />
      ) : null}
    </Box>
  );
}
