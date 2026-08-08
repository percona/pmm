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

import Badge from '@mui/material/Badge';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import { STEPLESS_STEP_KEY, STEPLESS_STEP_LABEL } from './ExecutionEventsPanel';

export interface LogStepTabsProps {
  steps: string[];
  activeStep: string | undefined;
  unreadSteps: Set<string>;
  onSelect: (step: string) => void;
}

function stepLabel(step: string): string {
  return step === STEPLESS_STEP_KEY ? STEPLESS_STEP_LABEL : step;
}

export function LogStepTabs({
  steps,
  activeStep,
  unreadSteps,
  onSelect,
}: LogStepTabsProps) {
  if (steps.length === 0) {
    return null;
  }

  // Clamp to a valid step so MUI Tabs doesn't warn when activeStep tracks a
  // different set (e.g. log steps vs execution-event steps).
  const resolvedValue =
    activeStep !== undefined && steps.includes(activeStep)
      ? activeStep
      : steps[0];

  return (
    <Tabs
      value={resolvedValue}
      onChange={(_, value: string) => onSelect(value)}
      variant="scrollable"
      scrollButtons="auto"
      sx={{ minHeight: 36, '& .MuiTab-root': { minHeight: 36, py: 0.5 } }}
    >
      {steps.map((step) => (
        <Tab
          key={step}
          value={step}
          label={
            <Badge
              color="primary"
              variant="dot"
              invisible={!unreadSteps.has(step) || step === activeStep}
              sx={{ '& .MuiBadge-badge': { right: -8, top: 2 } }}
            >
              <span>{stepLabel(step)}</span>
            </Badge>
          }
        />
      ))}
    </Tabs>
  );
}
