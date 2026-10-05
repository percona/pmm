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

/** Chunk lazy-loaded from ``PluginDetailPage`` — keeps ``sql-formatter`` and ``prism-react-renderer`` out of the main framework bundle. */

import { format as formatSql } from 'sql-formatter';
import { Highlight, themes } from 'prism-react-renderer';
import Box from '@mui/material/Box';
import { useTheme } from '@mui/material/styles';
import { detailPrism } from './detailPrism';
import {
  detailSyntaxBlockSx,
  type DetailSyntaxLanguage,
} from './detailSyntaxStyles';

function formatCreateSql(raw: string): string {
  const trimmed = raw.trim();
  if (!trimmed) {
    return raw;
  }
  try {
    return formatSql(trimmed, { language: 'mysql', tabWidth: 2 });
  } catch {
    return raw;
  }
}

function formatKeysForDisplay(value: unknown): string {
  if (typeof value === 'string') {
    const t = value.trim();
    if (!t) {
      return value;
    }
    try {
      return JSON.stringify(JSON.parse(t), null, 2);
    } catch {
      return value;
    }
  }
  if (value !== null && typeof value === 'object') {
    return JSON.stringify(value, null, 2);
  }
  return String(value);
}

function formatDetailSyntaxCode(
  value: unknown,
  language: DetailSyntaxLanguage
): string {
  if (language === 'sql') {
    return typeof value === 'string' ? formatCreateSql(value) : String(value);
  }
  if (language === 'yaml') {
    // A non-string value can't be raw YAML; pretty-print it rather than
    // rendering "[object Object]".
    return typeof value === 'string' ? value : formatKeysForDisplay(value);
  }
  return formatKeysForDisplay(value);
}

interface DetailSyntaxHighlighterProps {
  value: unknown;
  language: DetailSyntaxLanguage;
}

export default function DetailSyntaxHighlighter({
  value,
  language,
}: DetailSyntaxHighlighterProps) {
  const muiTheme = useTheme();
  const prismTheme =
    muiTheme.palette.mode === 'dark' ? themes.vsDark : themes.vsLight;
  const code = formatDetailSyntaxCode(value, language);

  return (
    <Highlight
      prism={detailPrism}
      theme={prismTheme}
      code={code}
      language={language}
    >
      {({ className, style, tokens, getLineProps, getTokenProps }) => (
        <Box
          component="pre"
          className={className}
          style={style}
          sx={{
            ...detailSyntaxBlockSx,
            margin: 0,
          }}
        >
          {tokens.map((line, i) => (
            <div key={i} {...getLineProps({ line })}>
              {line.map((token, key) => (
                <span key={key} {...getTokenProps({ token })} />
              ))}
            </div>
          ))}
        </Box>
      )}
    </Highlight>
  );
}
