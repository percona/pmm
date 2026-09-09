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

import Alert from '@mui/material/Alert';
import type { SxProps, Theme } from '@mui/material/styles';
import { actionErrorMessage } from './actionErrorMessage';

export interface ActionErrorAlertProps {
  /** Raw failure — a mutation's `error`, or the value held by `useActionError`. */
  error: unknown;
  /** Shown only when the error carries no message of its own. */
  fallback?: string;
  /** Renders the dismiss button when given (e.g. `mutation.reset`). */
  onClose?: () => void;
  sx?: SxProps<Theme>;
  testId?: string;
}

/**
 * Report a failed action from SEP's own component tree.
 *
 * Every mutation in SEP can be refused — the API restricts state-changing
 * routes to admins — so a failure that is only enqueued as a toast is invisible
 * wherever the host application mounts no snackbar provider. This alert renders
 * in the failing component's own tree, so it does not depend on that host
 * contract, and it carries the server's reason rather than a generic sentence.
 *
 * Renders nothing when there is no error, so it can sit unconditionally in a
 * layout.
 */
export function ActionErrorAlert({
  error,
  fallback,
  onClose,
  sx,
  testId = 'action-error-alert',
}: ActionErrorAlertProps) {
  if (error === null || error === undefined) {
    return null;
  }

  return (
    <Alert severity="error" onClose={onClose} sx={sx} data-testid={testId}>
      {actionErrorMessage(error, fallback)}
    </Alert>
  );
}
