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
import Checkbox from '@mui/material/Checkbox';
import FormControlLabel from '@mui/material/FormControlLabel';
import Tooltip from '@mui/material/Tooltip';
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

const TOOLTIP_AVAILABLE = 'Enable to trigger an alert if the task fails';
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
}

/**
 * Renders the *Alert on failure* checkbox shared by every task plugin's
 * create/edit form. Replaces the
 * `templates/tasks/partials/create-form-alert-on-failure-input.html.j2`
 * partial: when the backend reports no configured alert providers the
 * checkbox is disabled (not hidden) and a tooltip prompts the operator to
 * configure a provider.
 *
 * Must be rendered inside a react-hook-form `<FormProvider>`.
 */
export function AlertOnFailField({
  defaultValue = false,
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
      ? TOOLTIP_AVAILABLE
      : TOOLTIP_UNAVAILABLE;

  const {
    field: { onChange, onBlur, value, ref, name },
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
    // `describeChild` keeps the checkbox's accessible name (the
    // FormControlLabel text) intact; the tooltip is attached via
    // aria-describedby instead of overriding aria-label. The Tooltip wraps
    // FormControlLabel directly — only the inner Checkbox is disabled, so
    // FormControlLabel itself stays hoverable and propagates the
    // describedby relationship as close to the form control as possible.
    <Tooltip title={tooltip} placement="top" describeChild>
      <FormControlLabel
        label="Alert on failure"
        control={
          <Checkbox
            name={name}
            inputRef={ref}
            checked={!!value}
            onChange={(_, checked) => onChange(checked)}
            onBlur={onBlur}
            disabled={disabled}
          />
        }
      />
    </Tooltip>
  );
}
