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

import { lazy, Suspense, type CSSProperties } from 'react';

const DetailSyntaxHighlighter = lazy(() =>
  import('@sep/framework').then((mod) => ({
    default: mod.DetailSyntaxHighlighter,
  }))
);

export const sectionStyle: CSSProperties = {
  border: '1px solid rgba(0, 0, 0, 0.12)',
  borderRadius: 4,
  padding: '1.5rem',
  marginBottom: '1.5rem',
};

export const sectionHeadingStyle: CSSProperties = {
  marginTop: 0,
  marginBottom: '1rem',
  fontSize: '1.125rem',
};

export const tableStyle: CSSProperties = {
  width: '100%',
  borderCollapse: 'collapse',
};

export const cellStyle: CSSProperties = {
  padding: '0.5rem 0.75rem',
  borderBottom: '1px solid rgba(0, 0, 0, 0.08)',
  textAlign: 'left',
};

export const preStyle: CSSProperties = {
  margin: 0,
  padding: '1rem',
  background: '#fafafa',
  borderRadius: 4,
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
  <pre
    style={{
      ...preStyle,
      minHeight: '8rem',
    }}
    aria-hidden
  />
);

export function PbmConfigSection({ task }: { task: Record<string, unknown> }) {
  const configYaml = readPbmConfigYaml(task);
  if (!configYaml) {
    return null;
  }

  return (
    <section style={sectionStyle}>
      <h2 style={sectionHeadingStyle}>Configuration</h2>
      <Suspense fallback={configSyntaxFallback}>
        <DetailSyntaxHighlighter value={configYaml} language="yaml" />
      </Suspense>
    </section>
  );
}
