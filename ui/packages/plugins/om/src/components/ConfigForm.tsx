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

import { useEffect, useMemo, useState } from 'react';
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  MenuItem,
  Stack,
  Switch,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { SETTING_HELP, SETTING_LABEL, SETTING_UNIT } from '../constants';
import {
  useOmInventoryConfig,
  useResetOmInventoryConfig,
  useUpdateOmInventoryConfig,
} from '../inventoryHooks';
import type { OmInventorySetting } from '../types';

/** The periods `SCHEDULE__period` accepts, from `sqlalchemy_celery_beat`. */
const PERIODS = ['seconds', 'minutes', 'hours', 'days'] as const;

/** Which input can edit a field. */
type InputKind = 'bool' | 'int' | 'period' | 'text';

/**
 * Choose the input from the type the app declares, not from the value.
 *
 * Guessing from the value would get `PROBE_DATABASE` right and `STALE_RUN_AFTER`
 * wrong: a `timedelta` arrives as a number of seconds and would render as a bare
 * integer with no unit beside it.
 */
function inputKind(setting: OmInventorySetting): InputKind {
  if (setting.key === 'SCHEDULE__period') {
    return 'period';
  }
  if (setting.type === 'bool') {
    return 'bool';
  }
  if (setting.type === 'int' || setting.type === 'timedelta') {
    return 'int';
  }
  return 'text';
}

/** Parse an edited value back into what the app expects on the wire. */
function toWireValue(setting: OmInventorySetting, draft: string): unknown {
  switch (inputKind(setting)) {
    case 'bool':
      return draft === 'true';
    case 'int':
      return Number(draft);
    default:
      return draft;
  }
}

/**
 * Whether a draft can be submitted.
 *
 * Every integer knob here is a count or a duration, and zero breaks each of them: a
 * zero semaphore admits nobody, a zero interval is not a schedule, a zero retention
 * keeps nothing. The app rejects them too - refusing here saves a round trip and says
 * so beside the field rather than in a banner.
 */
function isValid(setting: OmInventorySetting, draft: string): boolean {
  if (inputKind(setting) !== 'int') {
    return draft.trim().length > 0;
  }
  const parsed = Number(draft);
  return Number.isInteger(parsed) && parsed > 0;
}

/** One field, rendered as whatever can edit it. */
function SettingField({
  setting,
  draft,
  onChange,
  onReset,
  resetting,
}: {
  setting: OmInventorySetting;
  draft: string;
  onChange: (next: string) => void;
  onReset: () => void;
  resetting: boolean;
}) {
  const kind = inputKind(setting);
  const unit = SETTING_UNIT[setting.key];
  const label = SETTING_LABEL[setting.key] ?? setting.key;
  const help = SETTING_HELP[setting.key];
  const valid = isValid(setting, draft);

  return (
    <Stack direction="row" gap={2} alignItems="flex-start" sx={{ py: 0.75 }}>
      <Box sx={{ minWidth: 280, maxWidth: 280 }}>
        {kind === 'bool' ? (
          <Stack direction="row" alignItems="center" gap={1}>
            {/* The Typography beside it is adjacent, not associated, so without an
                aria-label every boolean setting is an unnamed control - a screen
                reader announces "switch, on" and nothing about which setting. */}
            <Switch
              size="small"
              checked={draft === 'true'}
              slotProps={{ input: { 'aria-label': label } }}
              onChange={(event) =>
                onChange(event.target.checked ? 'true' : 'false')
              }
            />
            <Typography variant="body2">{label}</Typography>
          </Stack>
        ) : kind === 'period' ? (
          <TextField
            select
            size="small"
            fullWidth
            label={label}
            value={draft}
            onChange={(event) => onChange(event.target.value)}
          >
            {PERIODS.map((period) => (
              <MenuItem key={period} value={period}>
                {period}
              </MenuItem>
            ))}
          </TextField>
        ) : (
          <TextField
            size="small"
            fullWidth
            label={unit ? `${label} (${unit})` : label}
            value={draft}
            error={!valid}
            helperText={
              // This TextField serves both `int` and `text` settings, and `isValid`
              // only requires a text one to be non-empty. Telling the reader of a
              // REPO_URL to enter a whole number is an instruction they cannot follow.
              valid
                ? undefined
                : kind === 'int'
                  ? 'A whole number greater than zero.'
                  : 'This field cannot be empty.'
            }
            onChange={(event) => onChange(event.target.value)}
          />
        )}
      </Box>
      <Stack sx={{ flex: 1, pt: 0.5 }} gap={0.5}>
        {help && (
          <Typography variant="caption" color="text.secondary">
            {help}
          </Typography>
        )}
        {/* Only when it is an override. Saying "from the deployment's configuration"
            on every unoverridden field is the same sentence thirteen times, which
            carries no information and buries the two or three rows that do. */}
        {setting.has_override && (
          <Stack direction="row" gap={1} alignItems="center">
            <Chip size="small" variant="outlined" label="overridden" />
            <Tooltip title="Put this back to the value the deployment configured. Without it, once a value has been set the deployed one is unreachable.">
              <Button size="small" disabled={resetting} onClick={onReset}>
                Reset
              </Button>
            </Tooltip>
          </Stack>
        )}
      </Stack>
    </Stack>
  );
}

/**
 * OM's configuration, as a form.
 *
 * Three rules, each of which is a way for this form to lie:
 *
 * - **Only runtime-changeable fields appear at all.** A field needing a restart would
 *   promise a change it cannot deliver, and showing one read-only beside editable ones
 *   just raises the question of why. `reload === 'SETTING_RELOAD_HOT'` is the filter, and it is what
 *   drops the two that are not: `CREDENTIALS_PATH`, deliberately not overridable
 *   because it names a file read on every database host and handed to a driver as a
 *   URI, and `FASTAPI_ENV`, which is the framework's rather than OM's.
 * - **The effective value is shown, never the submitted one.** The form re-reads after
 *   writing and re-seeds from what comes back. Beat runs as a forked side-car process,
 *   and a form that echoed the submission would look identical whether or not the
 *   change reached it.
 * - **Units and costs are stated.** "1800" beside "wedged after" is ambiguous in a way
 *   that matters, and an interval that dispatches a job to every host is an
 *   operational choice rather than a preference.
 *
 * Saved as one batch, because the app applies one: a single bad key rejects all of it
 * and writes nothing, so per-field saves would misrepresent what the API does.
 */
export function ConfigForm() {
  const { data: settings, isPending, isError, error } = useOmInventoryConfig();
  const update = useUpdateOmInventoryConfig();
  const reset = useResetOmInventoryConfig();
  const [drafts, setDrafts] = useState<Record<string, string>>({});

  const editable = useMemo(
    () =>
      (settings ?? []).filter(
        (setting) => setting.reload === 'SETTING_RELOAD_HOT'
      ),
    [settings]
  );

  // Re-seed from the effective values whenever they change, including after a write:
  // what the server settled on wins over what was typed. Without this the boxes would
  // keep showing a submission the app may have coerced or rejected.
  useEffect(() => {
    setDrafts(
      Object.fromEntries(
        // Nullish coerces to "" rather than the literal words "null"/"undefined":
        // setting.value is unknown, and String() would put those in the box, let isValid
        // pass them as non-empty, and save them back as text.
        editable.map((setting) => [
          setting.key,
          setting.value == null ? '' : String(setting.value),
        ])
      )
    );
  }, [editable]);

  if (isPending) {
    return <CircularProgress size={20} />;
  }
  if (isError) {
    return (
      <Alert severity="warning" sx={{ mt: 2 }}>
        Could not read OM&apos;s configuration: {(error as Error).message}
      </Alert>
    );
  }

  const dirty = editable.filter(
    (setting) => drafts[setting.key] !== String(setting.value)
  );
  const invalid = dirty.filter(
    (setting) => !isValid(setting, drafts[setting.key] ?? '')
  );

  const save = () =>
    update.mutate(
      Object.fromEntries(
        dirty.map((setting) => [
          setting.key,
          toWireValue(setting, drafts[setting.key] ?? ''),
        ])
      )
    );

  const render = (setting: OmInventorySetting) => (
    <SettingField
      key={setting.key}
      setting={setting}
      draft={drafts[setting.key] ?? ''}
      resetting={reset.isPending}
      onChange={(next) =>
        setDrafts((current) => ({ ...current, [setting.key]: next }))
      }
      onReset={() => reset.mutate(setting.key)}
    />
  );

  const basic = editable.filter((setting) => !setting.is_advanced);
  const advanced = editable.filter((setting) => setting.is_advanced);

  return (
    <Stack gap={1}>
      {/* No heading: the tab is already labelled, and a second "Configuration"
          under it would be the same word twice. */}
      <Typography variant="body2" color="text.secondary">
        Everything here takes effect without a restart. Values come from the
        deployment&apos;s configuration unless marked as overridden.
      </Typography>

      <Box>{basic.map(render)}</Box>

      {advanced.length > 0 && (
        <Accordion disableGutters elevation={0} sx={{ bgcolor: 'transparent' }}>
          <AccordionSummary expandIcon={<ExpandMoreIcon />}>
            <Typography variant="subtitle2">
              Advanced ({advanced.length})
            </Typography>
          </AccordionSummary>
          <AccordionDetails>
            {/* Collapsed, and grouped by the app's own `is_advanced` flag rather than
                a list kept here - so a setting SEP adds lands in the right section
                without this file knowing about it. */}
            <Typography variant="caption" color="text.secondary">
              Timeouts, concurrency and retention. Raising concurrency or
              lowering timeouts changes how hard a sweep leans on Nomad and on
              the hosts.
            </Typography>
            <Box sx={{ mt: 1 }}>{advanced.map(render)}</Box>
          </AccordionDetails>
        </Accordion>
      )}

      {update.isError && <Alert severity="error">{update.error.message}</Alert>}
      {reset.isError && <Alert severity="error">{reset.error.message}</Alert>}

      <Stack direction="row" gap={2} alignItems="center" sx={{ mt: 1 }}>
        <Button
          variant="contained"
          disabled={
            dirty.length === 0 || invalid.length > 0 || update.isPending
          }
          onClick={save}
        >
          {dirty.length > 1 ? `Save ${dirty.length} changes` : 'Save'}
        </Button>
        {dirty.length > 0 && invalid.length === 0 && (
          <Typography variant="body2" color="text.secondary">
            {dirty.map((setting) => setting.key).join(', ')}
          </Typography>
        )}
        {invalid.length > 0 && (
          <Typography variant="body2" color="error">
            Fix {invalid.map((setting) => setting.key).join(', ')} first - the
            app applies the batch or none of it.
          </Typography>
        )}
      </Stack>
    </Stack>
  );
}
