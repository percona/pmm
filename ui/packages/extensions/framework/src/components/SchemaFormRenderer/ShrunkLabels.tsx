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

import { useMemo, type ReactNode } from 'react';
import { ThemeProvider, useTheme, type Theme } from '@mui/material/styles';

// Peak UI's `TextInput` always shrinks its label onto the outline, while its
// `SelectInput` and `AutoCompleteInput` leave an empty field's label inside and
// expose no prop to change it, so the form sets the default for all of them.
const withShrunkLabels = (outer: Theme): Theme => ({
  ...outer,
  components: {
    ...outer.components,
    MuiInputLabel: {
      ...outer.components?.MuiInputLabel,
      defaultProps: {
        ...outer.components?.MuiInputLabel?.defaultProps,
        shrink: true,
      },
    },
    MuiOutlinedInput: {
      ...outer.components?.MuiOutlinedInput,
      defaultProps: {
        ...outer.components?.MuiOutlinedInput?.defaultProps,
        notched: true,
      },
    },
  },
});

/** Gives every outlined control in a schema form one label position. */
export function ShrunkLabels({ children }: { children: ReactNode }) {
  const outer = useTheme();
  const theme = useMemo(() => withShrunkLabels(outer), [outer]);
  return <ThemeProvider theme={theme}>{children}</ThemeProvider>;
}
