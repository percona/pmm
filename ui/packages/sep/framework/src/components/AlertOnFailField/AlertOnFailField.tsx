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

import { useEffect } from 'react';
import Box from '@mui/material/Box';
import Tooltip from '@mui/material/Tooltip';
import { SwitchInput } from '@percona/peak-ui';
import {
  useController,
  useFormContext,
  type FieldValues,
} from 'react-hook-form';
import { useAlertConfig } from '@sep/api';

/**
 * Form field name submitted to the task-creation API. Matches the snake_case
 * key the backend expects on `POST /tasks` (`alert_on_fail`) so plugin forms
 * can simply spread the form values into the request payload.
 */
export const ALERT_ON_FAIL_FIELD_NAME = 'alert_on_fail';

const TOOLTIP_UNAVAILABLE = 'Configure an alert provider to use this feature';
const TOOLTIP_ERROR = 'Could not load alert configuration';

interface AlertOnFailFieldProps {
  /**
   * Initial value for the field. Mirrors the Jinja2 partial's
   * `alert_on_fail_default`. The value is reset to `false` if the
   * availability query later reports no configured providers, so a
   * persisted `true` from an edit form can never round-trip to the
   * backend when alerts are disabled.
   */
  defaultValue?: boolean;
  /** Mid-sentence singular noun for one record (e.g. `backup`). */
  itemName?: string;
}

/**
 * Renders the *Alert on failure* toggle shared by every task plugin's
 * create/edit form. Replaces the
 * `templates/tasks/partials/create-form-alert-on-failure-input.html.j2`
 * partial: when the backend reports no configured alert providers the
 * control is disabled (not hidden) and a tooltip prompts the operator to
 * configure a provider.
 *
 * A Peak UI `SwitchInput`, the same control `BoolField` renders — it was the
 * form's one checkbox among a dozen toggles, and nothing about this boolean
 * differs from theirs (PMM-15456).
 *
 * Must be rendered inside a react-hook-form `<FormProvider>`.
 */
export function AlertOnFailField({
  defaultValue = false,
  itemName = 'task',
}: AlertOnFailFieldProps) {
  const { control } = useFormContext<FieldValues>();
  const { data, isLoading, isError } = useAlertConfig();
  const available = data?.available ?? false;
  // Stay disabled while the availability query is in flight or has errored
  // so the field can't briefly appear actionable before resolving.
  const disabled = isLoading || isError || !available;
  const tooltip = isError
    ? TOOLTIP_ERROR
    : available
      ? `Enable to trigger an alert if the ${itemName} fails`
      : TOOLTIP_UNAVAILABLE;

  // Registers the field with its default and reads the current value for the
  // effect below. `SwitchInput` opens its own controller on the same name, so
  // the two share one piece of form state rather than competing for it — this
  // one is not wired to the control.
  const {
    field: { onChange, value },
  } = useController({
    name: ALERT_ON_FAIL_FIELD_NAME,
    control,
    defaultValue,
  });

  // Once we know providers are unavailable, force the form value to false so
  // the UI (`checked`) and the submitted payload stay aligned. Covers two
  // cases: (1) edit forms that mounted with `defaultValue=true` before the
  // availability query resolved as `false`; (2) a user toggling the field on
  // and then alert providers becoming unavailable mid-session.
  const knownUnavailable = !isLoading && !isError && !available;
  useEffect(() => {
    if (knownUnavailable && value) {
      onChange(false);
    }
  }, [knownUnavailable, value, onChange]);

  return (
    // `describeChild` keeps the switch's accessible name (the label text)
    // intact; the tooltip is attached via aria-describedby instead of
    // overriding aria-label. The Tooltip wraps the whole control so it stays
    // hoverable while the switch itself is disabled.
    <Tooltip title={tooltip} placement="top" describeChild>
      <Box component="span" sx={{ display: 'inline-flex' }}>
        <SwitchInput
          name={ALERT_ON_FAIL_FIELD_NAME}
          control={control}
          label="Alert on failure"
          switchFieldProps={{ disabled }}
        />
      </Box>
    </Tooltip>
  );
}
