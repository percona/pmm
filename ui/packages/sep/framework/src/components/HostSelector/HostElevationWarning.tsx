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

import { useWatch } from 'react-hook-form';
import { Alert } from '@mui/material';
import type { SepComponents } from '@sep/api';
import { useHosts } from '../../hooks/useHosts';
import { hostValueId } from './HostSelector';

/** Whether a snippet never, optionally, or always runs with `sudo`. */
export type SnippetSudoRequirement =
  SepComponents['schemas']['SnippetSudoRequirement'];

/** A snippet the form will launch, as far as elevation is concerned. */
export interface SnippetElevation {
  title: string;
  /** `null` or absent when the server predates the field; never warns. */
  sudo?: SnippetSudoRequirement | null;
}

export interface HostElevationWarningProps {
  /** react-hook-form name of the host field the warning describes. */
  name: string;
  /** react-hook-form name of the bool field optional-sudo snippets follow. */
  sudoName: string;
  /** The snippets the form will launch on the selected host. */
  snippets: readonly SnippetElevation[];
}

/**
 * The snippets a dispatch would launch with `sudo`.
 *
 * Mirrors how the backend applies a form's single sudo choice: a snippet that
 * always or never elevates ignores it, and only an optional-sudo one follows it.
 */
export function snippetsLaunchedWithSudo<T extends SnippetElevation>(
  snippets: readonly T[],
  sudoChosen: boolean
): T[] {
  return snippets.filter(
    (snippet) =>
      snippet.sudo === 'always' || (sudoChosen && snippet.sudo === 'optional')
  );
}

/**
 * Non-blocking warning that the selected executor cannot launch some of the
 * snippets about to run on it, because they run with `sudo` and the host cannot
 * elevate. The executor would otherwise refuse them as unlaunchable only after
 * dispatch.
 *
 * Only a host measured unable warns. One never observed stays silent: nothing
 * is known about it, and a warning on every such host would be noise. It does
 * not block the submit, since the capability is a periodic measurement and can
 * be out of date.
 *
 * Must render inside the form that owns `name` and `sudoName`.
 */
export function HostElevationWarning({
  name,
  sudoName,
  snippets,
}: HostElevationWarningProps) {
  const hostValue = useWatch({ name });
  const sudoChosen = useWatch({ name: sudoName }) === true;
  const { data: hosts } = useHosts();

  const hostId = hostValueId(hostValue);
  const host = hosts?.find((candidate) => candidate.id === hostId);
  if (host?.can_elevate !== false) {
    return null;
  }
  const unlaunchable = snippetsLaunchedWithSudo(snippets, sudoChosen);
  if (unlaunchable.length === 0) {
    return null;
  }
  const sudoIsOptionalForAll = unlaunchable.every(
    (snippet) => snippet.sudo === 'optional'
  );

  return (
    <Alert severity="warning" sx={{ mt: 1 }}>
      {host.name} cannot run commands with sudo, so these snippets will fail to
      launch on it: {unlaunchable.map((snippet) => snippet.title).join(', ')}.{' '}
      {sudoIsOptionalForAll
        ? 'Turn sudo off, or choose another host.'
        : 'Choose another host.'}
    </Alert>
  );
}
