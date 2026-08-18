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

import { useEffect, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  Stack,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import {
  usePomInventoryConfig,
  useResetPomInventoryConfig,
  useUpdatePomInventoryConfig,
} from '../inventoryHooks';
import type { PomInventorySetting } from '../types';

/**
 * The field holding how often the estate is swept.
 *
 * `SCHEDULE` is a nested model, so the configuration listing expands it into leaves
 * and there is no key called `SCHEDULE` in what comes back. The parent key is
 * accepted on write and is the only spelling that can express "off", which is a
 * difference a form driven purely off the listing could not discover.
 */
const EVERY_KEY = 'SCHEDULE__every';
const PERIOD_KEY = 'SCHEDULE__period';

/** Read one setting's effective value out of the listing. */
function valueOf(
  settings: PomInventorySetting[] | undefined,
  key: string
): unknown {
  return settings?.find((setting) => setting.key === key)?.value;
}

function settingOf(
  settings: PomInventorySetting[] | undefined,
  key: string
): PomInventorySetting | undefined {
  return settings?.find((setting) => setting.key === key);
}

/**
 * How often POM sweeps the estate, and what that costs.
 *
 * Three rules, each of which is a way for this form to lie:
 *
 * - **Only runtime-changeable fields get an input.** Anything else needs a restart, so
 *   an input for it promises a change it cannot deliver. The rest are listed read-only
 *   with where their value came from.
 * - **The effective value is shown, never the submitted one.** The form re-reads after
 *   writing and renders what comes back. Beat runs as a forked side-car process, and a
 *   form that echoed the submission would look identical whether or not the change
 *   reached it.
 * - **The unit and the cost are stated.** A sweep dispatches a job to every host and
 *   takes tens of seconds, so the interval is an operational choice rather than a
 *   preference, and "10" means nothing without "minutes" beside it.
 */
export function ScheduleForm() {
  const { data: settings, isPending, isError, error } = usePomInventoryConfig();
  const update = useUpdatePomInventoryConfig();
  const reset = useResetPomInventoryConfig();
  const [every, setEvery] = useState('');

  const effectiveEvery = valueOf(settings, EVERY_KEY);
  const period = String(valueOf(settings, PERIOD_KEY) ?? 'minutes');
  const everySetting = settingOf(settings, EVERY_KEY);

  // Re-seed the input from the effective value whenever it changes, including after a
  // write: what the server settled on wins over what was typed. Without this the box
  // would keep showing a submission the app may have coerced or rejected.
  useEffect(() => {
    if (effectiveEvery !== undefined && effectiveEvery !== null) {
      setEvery(String(effectiveEvery));
    }
  }, [effectiveEvery]);

  if (isPending) {
    return <CircularProgress size={20} />;
  }
  if (isError) {
    return (
      <Alert severity="warning">
        Could not read POM&apos;s configuration: {(error as Error).message}
      </Alert>
    );
  }

  const hot = (settings ?? []).filter((setting) => setting.reload === 'hot');
  const cold = (settings ?? []).filter((setting) => setting.reload !== 'hot');
  const dirty = every !== String(effectiveEvery ?? '');
  const parsed = Number(every);
  const valid = Number.isInteger(parsed) && parsed > 0;

  return (
    <Stack gap={2}>
      <Box>
        <Typography variant="subtitle2" gutterBottom>
          Schedule
        </Typography>
        <Stack direction="row" gap={1} alignItems="flex-start">
          <TextField
            size="small"
            label={`Sweep every (${period})`}
            value={every}
            error={Boolean(every) && !valid}
            helperText={
              every && !valid
                ? 'A whole number of periods, greater than zero.'
                : 'Each sweep dispatches a job to every host and takes tens of seconds.'
            }
            onChange={(event) => setEvery(event.target.value)}
            sx={{ maxWidth: 260 }}
          />
          <Button
            size="small"
            variant="contained"
            disabled={!dirty || !valid || update.isPending}
            onClick={() => update.mutate({ [EVERY_KEY]: parsed })}
            sx={{ mt: 0.5 }}
          >
            Save
          </Button>
          {everySetting?.has_override && (
            <Tooltip title="Put this back to the value the deployment configured. Without this, once a value has been set the deployed one is unreachable.">
              <Button
                size="small"
                disabled={reset.isPending}
                onClick={() => reset.mutate(EVERY_KEY)}
                sx={{ mt: 0.5 }}
              >
                Reset
              </Button>
            </Tooltip>
          )}
        </Stack>
        {update.isError && (
          <Alert severity="error" sx={{ mt: 1 }}>
            {update.error.message}
          </Alert>
        )}
        <Typography variant="caption" color="text.secondary">
          {everySetting?.has_override
            ? 'Overridden here.'
            : 'From the deployment’s configuration.'}
        </Typography>
      </Box>

      <Box>
        <Typography variant="subtitle2" gutterBottom>
          Other settings
        </Typography>
        <Typography variant="caption" color="text.secondary">
          {hot.length} take effect immediately; {cold.length} need a restart and
          are shown read-only.
        </Typography>
        <Stack gap={0.5} sx={{ mt: 1 }}>
          {(settings ?? []).map((setting) => (
            <Stack
              key={setting.key}
              direction="row"
              gap={1}
              alignItems="center"
            >
              <Typography variant="body2" sx={{ minWidth: 220 }}>
                {setting.key}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                {String(setting.value)}
              </Typography>
              {setting.has_override && (
                <Chip size="small" variant="outlined" label="overridden" />
              )}
              {setting.reload !== 'hot' && (
                <Tooltip title="Changing this needs a restart, so it is not editable here.">
                  <Chip size="small" variant="outlined" label="restart" />
                </Tooltip>
              )}
            </Stack>
          ))}
        </Stack>
      </Box>
    </Stack>
  );
}
