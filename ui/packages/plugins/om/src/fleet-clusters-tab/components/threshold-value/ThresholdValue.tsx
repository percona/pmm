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

import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import { Duration } from '../../../components/Metric';
import { THRESHOLDS_VALUE_CUE } from './ThresholdValue.constants';

/**
 * A duration, marked when it crosses its threshold: colour, an icon and a tooltip
 * saying which threshold, so the cue never rests on colour alone.
 */
export const ThresholdValue = ({
  seconds,
  crossed,
  level,
  reason,
}: {
  seconds: number | null | undefined;
  crossed: boolean;
  level: keyof typeof THRESHOLDS_VALUE_CUE;
  reason: string;
}) => {
  if (!crossed) {
    return <Duration value={seconds} />;
  }
  const { color, icon } = THRESHOLDS_VALUE_CUE[level];
  return (
    <Tooltip title={reason}>
      <Stack
        direction="row"
        alignItems="center"
        gap={0.5}
        tabIndex={0}
        sx={{ color, width: 'fit-content' }}
        data-testid={`om-threshold-${level}`}
      >
        {icon}
        <Duration value={seconds} />
      </Stack>
    </Tooltip>
  );
};
