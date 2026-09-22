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

import { useEffect, useMemo, useRef, useState } from 'react';
import { Alert, Box, CircularProgress, Typography } from '@mui/material';
import { useTheme } from '@mui/material/styles';
import { useFormContext, useWatch } from 'react-hook-form';
import { PrismLight as SyntaxHighlighter } from 'react-syntax-highlighter';
import bash from 'react-syntax-highlighter/dist/esm/languages/prism/bash';
import python from 'react-syntax-highlighter/dist/esm/languages/prism/python';
import {
  vs,
  vscDarkPlus,
} from 'react-syntax-highlighter/dist/esm/styles/prism';
import { apiClient } from '@sep/api';
import type { ScriptPreviewField as ScriptPreviewFieldType } from '../types';

SyntaxHighlighter.registerLanguage('bash', bash);
SyntaxHighlighter.registerLanguage('python', python);

interface ScriptPreviewFieldProps {
  field: ScriptPreviewFieldType;
}

interface ScriptPreviewResponse {
  content: string;
  language: string;
  is_truncated: boolean;
}

interface PreviewState {
  status: 'idle' | 'loading' | 'success' | 'error';
  data: ScriptPreviewResponse | null;
  error: string | null;
}

const DEBOUNCE_MS = 300;

/**
 * Read-only field that renders a backend-fetched script preview.
 *
 * Fetches `endpointUrl` on mount and re-fetches whenever any sibling field
 * listed in `dependsOn` changes value — debounced and cancellation-safe via
 * an AbortController. Renders the response's `content` with Prism-based
 * syntax highlighting (bash and python registered; unknown languages fall
 * back to no-color rendering). The rendered `<pre>` keeps `tabIndex={0}`
 * and a `data-language` attribute so the scrollable region remains
 * keyboard-focusable.
 */
export function ScriptPreviewField({ field }: ScriptPreviewFieldProps) {
  const { control } = useFormContext();
  const theme = useTheme();
  const dependsOnValues = useWatch({
    control,
    name: field.depends_on,
  });
  const dependsOnKey = useMemo(
    () => JSON.stringify(dependsOnValues ?? []),
    [dependsOnValues]
  );
  const dependsOnSchema = useMemo(
    () => JSON.stringify(field.depends_on),
    [field.depends_on]
  );

  const [state, setState] = useState<PreviewState>({
    status: 'idle',
    data: null,
    error: null,
  });
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;

    const handle = setTimeout(
      () => {
        setState((prev) => ({ ...prev, status: 'loading', error: null }));
        apiClient
          .get<ScriptPreviewResponse>(field.endpoint_url, {
            signal: controller.signal,
          })
          .then((response) => {
            setState({ status: 'success', data: response.data, error: null });
          })
          .catch((error: unknown) => {
            if (controller.signal.aborted) {
              return;
            }
            const message =
              error instanceof Error ? error.message : 'Preview unavailable';
            setState({ status: 'error', data: null, error: message });
          });
      },
      field.depends_on.length > 0 ? DEBOUNCE_MS : 0
    );

    return () => {
      clearTimeout(handle);
      controller.abort();
    };
  }, [field.endpoint_url, dependsOnSchema, dependsOnKey]);

  return (
    <Box
      sx={{
        border: 1,
        borderColor: 'divider',
        borderRadius: 1,
        p: 1,
        backgroundColor: (theme) =>
          theme.palette.mode === 'light'
            ? theme.palette.grey[50]
            : theme.palette.grey[900],
      }}
    >
      <Typography variant="subtitle2" sx={{ mb: 1 }}>
        {field.label}
      </Typography>
      {state.status === 'loading' && (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 2 }}>
          <CircularProgress size={20} aria-label="Loading preview" />
        </Box>
      )}
      {state.status === 'error' && (
        <Alert severity="error" variant="outlined">
          {state.error ?? 'Preview unavailable'}
        </Alert>
      )}
      {state.status === 'success' &&
        state.data &&
        (() => {
          const language = state.data.language || field.language || 'plaintext';
          const highlighterStyle =
            theme.palette.mode === 'dark' ? vscDarkPlus : vs;
          return (
            <>
              {state.data.is_truncated && (
                <Typography variant="caption" color="text.secondary">
                  Preview truncated.
                </Typography>
              )}
              <SyntaxHighlighter
                language={language}
                style={highlighterStyle}
                customStyle={{
                  margin: 0,
                  maxHeight: 400,
                  overflow: 'auto',
                  fontSize: '0.85rem',
                  borderRadius: 4,
                }}
                PreTag={(props) => (
                  <pre tabIndex={0} data-language={language} {...props} />
                )}
              >
                {state.data.content}
              </SyntaxHighlighter>
            </>
          );
        })()}
    </Box>
  );
}
