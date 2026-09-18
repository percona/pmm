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

/**
 * Why a terminal run that did not succeed ended, for the statuses whose
 * meaning is not self-evident from the badge alone.
 *
 * `failed` is deliberately absent: it renders at error severity with the
 * backend's own reason (or a pointer to the log), rather than a fixed sentence.
 *
 * Shared between `TaskRunDetailDrawer` and `TaskLogViewer`: the drawer wraps
 * the viewer with a summary that already explained a non-failure terminal
 * status, but a caller embedding the viewer directly — ATW's results pane does,
 * with no drawer around it — only ever saw the status chip. Lives beside the
 * viewer rather than the drawer so the lower-level component owns it and the
 * drawer (which already depends on the viewer) reads it from here.
 */
export const NON_FAILURE_TERMINAL_NOTES: Partial<Record<string, string>> = {
  stopped: 'This run was stopped before it finished.',
  lost: 'The executor stopped reporting on this run, so its outcome is unknown.',
  stale:
    'This run was skipped because the executor could not place it before the staleness threshold.',
  unlaunchable:
    'The executor node could not launch this run, so the payload never executed. This is not a script failure.',
};
