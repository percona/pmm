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

import { lazy, Suspense } from 'react';
import { Box, type SxProps, type Theme } from '@mui/material';

const DetailSyntaxHighlighter = lazy(() =>
  import('@sep/framework').then((mod) => ({
    default: mod.DetailSyntaxHighlighter,
  }))
);

/*
 * Detail-page styling, as theme-aware `sx` rather than the `CSSProperties`
 * literals this UI arrived with. The literals hardcoded light-mode values
 * (`#fafafa`, `rgba(0, 0, 0, 0.12)`), which render as near-black-on-black once
 * PMM's theme switches to dark -- the palette keys below resolve per mode, and
 * the numeric spacing/radius values come off the theme scale instead of
 * restating its 8px unit in rem.
 */

export const sectionSx: SxProps<Theme> = {
  border: 1,
  borderColor: 'divider',
  borderRadius: 1,
  p: 3,
  mb: 3,
};

export const sectionHeadingSx: SxProps<Theme> = {
  mt: 0,
  mb: 2,
  fontSize: '1.125rem',
};

export const tableSx: SxProps<Theme> = {
  width: '100%',
  borderCollapse: 'collapse',
};

export const cellSx: SxProps<Theme> = {
  px: 1.5,
  py: 1,
  borderBottom: 1,
  borderColor: 'divider',
  textAlign: 'left',
};

export const preSx: SxProps<Theme> = {
  m: 0,
  p: 2,
  bgcolor: 'action.hover',
  borderRadius: 1,
  overflow: 'auto',
  fontFamily: "'Roboto Mono', monospace",
  fontSize: '0.875rem',
  whiteSpace: 'pre-wrap',
};

/** Read the parent task's PBM YAML from ``task.data.meta.config``. */
export function readPbmConfigYaml(
  task: Record<string, unknown>
): string | null {
  const data = task.data;
  if (!data || typeof data !== 'object' || Array.isArray(data)) {
    return null;
  }
  const meta = (data as Record<string, unknown>).meta;
  if (!meta || typeof meta !== 'object' || Array.isArray(meta)) {
    return null;
  }
  const config = (meta as Record<string, unknown>).config;
  if (typeof config !== 'string') {
    return null;
  }
  const trimmed = config.trim();
  return trimmed || null;
}

const configSyntaxFallback = (
  <Box component="pre" sx={[preSx, { minHeight: '8rem' }]} aria-hidden />
);

export function PbmConfigSection({ task }: { task: Record<string, unknown> }) {
  const configYaml = readPbmConfigYaml(task);
  if (!configYaml) {
    return null;
  }

  return (
    <Box component="section" sx={sectionSx}>
      <Box component="h2" sx={sectionHeadingSx}>
        Configuration
      </Box>
      <Suspense fallback={configSyntaxFallback}>
        <DetailSyntaxHighlighter value={configYaml} language="yaml" />
      </Suspense>
    </Box>
  );
}
