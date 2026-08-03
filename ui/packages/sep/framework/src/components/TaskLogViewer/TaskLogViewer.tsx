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

import DownloadIcon from '@mui/icons-material/Download';
import Badge from '@mui/material/Badge';
import Box from '@mui/material/Box';
import FormControl from '@mui/material/FormControl';
import FormControlLabel from '@mui/material/FormControlLabel';
import IconButton from '@mui/material/IconButton';
import MenuItem from '@mui/material/MenuItem';
import Paper from '@mui/material/Paper';
import Select from '@mui/material/Select';
import Stack from '@mui/material/Stack';
import Switch from '@mui/material/Switch';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { useEffect, useMemo, useRef, useState } from 'react';
import { useExecutionEvents } from '../../hooks/useExecutionEvents';
import { useLogDownload } from '../../hooks/useLogDownload';
import { useTaskLogs, type LogType } from '../../hooks/useTaskLogs';
import { ExecutionEventsPanel } from './ExecutionEventsPanel';
import { LogOutputPane } from './LogOutputPane';
import { LogStepTabs } from './LogStepTabs';
import { StatusBadge, type BadgeStatus } from './StatusBadge';
import { StreamErrorBlock } from './StreamErrorBlock';

type TopTab = 'stdout' | 'stderr' | 'events';

export const DEFAULT_LOG_TAIL_LINES = 1000;

export const LOG_TAIL_LINE_OPTIONS = [
  { label: '100', value: '100' },
  { label: '1000', value: '1000' },
  { label: '5000', value: '5000' },
  { label: 'All', value: 'all' },
] as const;

export type LogTailLineChoice = (typeof LOG_TAIL_LINE_OPTIONS)[number]['value'];

const LOG_TAIL_STORAGE_KEY = 'sep.taskLogViewer.tail';

const DEFAULT_LOG_TAIL_CHOICE = '1000' satisfies LogTailLineChoice;

function readStoredLogTailChoice(): LogTailLineChoice {
  if (globalThis.localStorage === undefined) {
    return DEFAULT_LOG_TAIL_CHOICE;
  }
  const stored = globalThis.localStorage.getItem(LOG_TAIL_STORAGE_KEY);
  if (stored === 'all') {
    return 'all';
  }
  if (stored === '100' || stored === '1000' || stored === '5000') {
    return stored;
  }
  return DEFAULT_LOG_TAIL_CHOICE;
}

function logTailChoiceToParam(choice: LogTailLineChoice): number | undefined {
  return choice === 'all' ? undefined : Number(choice);
}

export interface TaskLogViewerProps {
  taskHistoryId: number | string;
  taskStatus?: string;
  height?: number | string;
}

function isRunningStatus(status?: string): boolean {
  return (status ?? '').toLowerCase() === 'running';
}

function resolveBadgeStatus(
  finishStatus: ReturnType<typeof useTaskLogs>['finishStatus'],
  error: ReturnType<typeof useTaskLogs>['error']
): BadgeStatus | undefined {
  if (error) {
    return error.code === 410 ? 'executor-gone' : 'stream-error';
  }
  return finishStatus;
}

export function TaskLogViewer({
  taskHistoryId,
  taskStatus,
  height = 480,
}: TaskLogViewerProps) {
  const running = isRunningStatus(taskStatus);
  const [logTailChoice, setLogTailChoice] = useState<LogTailLineChoice>(
    readStoredLogTailChoice
  );
  const tailLines = logTailChoiceToParam(logTailChoice);
  const effectiveTailLines = running ? undefined : tailLines;
  const { textByStep, stepOrder, finishStatus, error } = useTaskLogs(
    taskHistoryId,
    effectiveTailLines
  );
  const { eventsByStep, stepOrder: eventStepOrder } = useExecutionEvents(
    taskHistoryId,
    running
  );

  const [topTab, setTopTab] = useState<TopTab>('stdout');
  const [activeStep, setActiveStep] = useState<string | undefined>();
  const [wrap, setWrap] = useState(false);

  const [unreadTypes, setUnreadTypes] = useState<Set<LogType>>(new Set());
  const [unreadEvents, setUnreadEvents] = useState(false);
  const [unreadSteps, setUnreadSteps] = useState<Set<string>>(new Set());

  const prevLogSizesRef = useRef<Record<string, number>>({});
  const prevEventCountRef = useRef(0);

  // Reset view state when switching to a different task history
  useEffect(() => {
    setActiveStep(undefined);
    setUnreadTypes(new Set());
    setUnreadSteps(new Set());
    setUnreadEvents(false);
    prevLogSizesRef.current = {};
    prevEventCountRef.current = 0;
  }, [taskHistoryId]);

  useEffect(() => {
    if (!activeStep && stepOrder.length > 0) {
      setActiveStep(stepOrder[0]);
    }
  }, [stepOrder, activeStep]);

  // Track unread notifications based on accumulated text growth
  useEffect(() => {
    for (const step of stepOrder) {
      const pane = textByStep[step];
      if (!pane) {
        continue;
      }
      (['stdout', 'stderr'] as const).forEach((type) => {
        const key = `${step}_${type}`;
        const size = pane[type].length;
        const prev = prevLogSizesRef.current[key] ?? 0;
        if (size > prev) {
          if (topTab !== type) {
            setUnreadTypes((prevSet) => {
              if (prevSet.has(type)) {
                return prevSet;
              }
              const next = new Set(prevSet);
              next.add(type);
              return next;
            });
          }
          if (activeStep !== step) {
            setUnreadSteps((prevSet) => {
              if (prevSet.has(step)) {
                return prevSet;
              }
              const next = new Set(prevSet);
              next.add(step);
              return next;
            });
          }
        }
        prevLogSizesRef.current[key] = size;
      });
    }
  }, [textByStep, stepOrder, topTab, activeStep]);

  // Events unread badge on top tab
  useEffect(() => {
    const total = Object.values(eventsByStep).reduce(
      (sum, list) => sum + list.length,
      0
    );
    if (total > prevEventCountRef.current && topTab !== 'events') {
      setUnreadEvents(true);
    }
    prevEventCountRef.current = total;
  }, [eventsByStep, topTab]);

  const handleTopTab = (value: TopTab) => {
    setTopTab(value);
    if (value === 'events') {
      setUnreadEvents(false);
    } else {
      setUnreadTypes((prev) => {
        if (!prev.has(value)) {
          return prev;
        }
        const next = new Set(prev);
        next.delete(value);
        return next;
      });
    }
  };

  const handleStepSelect = (step: string) => {
    setActiveStep(step);
    setUnreadSteps((prev) => {
      if (!prev.has(step)) {
        return prev;
      }
      const next = new Set(prev);
      next.delete(step);
      return next;
    });
  };

  const stepsForTabs = useMemo(
    () => (topTab === 'events' ? eventStepOrder : stepOrder),
    [topTab, eventStepOrder, stepOrder]
  );

  const currentPaneText = useMemo(() => {
    if (topTab === 'events' || !activeStep) {
      return '';
    }
    const pane = textByStep[activeStep];
    if (!pane) {
      return '';
    }
    return pane[topTab] ?? '';
  }, [textByStep, activeStep, topTab]);

  const download = useLogDownload();
  const handleDownload = () => {
    if (topTab === 'events') {
      return;
    }
    const filename = `task-${taskHistoryId}-${activeStep ?? 'step'}-${topTab}.log`;
    download(filename, currentPaneText);
  };

  const handleLogTailChange = (choice: LogTailLineChoice) => {
    setLogTailChoice(choice);
    if (globalThis.localStorage !== undefined) {
      globalThis.localStorage.setItem(LOG_TAIL_STORAGE_KEY, choice);
    }
  };

  const badgeStatus = resolveBadgeStatus(finishStatus, error);

  return (
    <Paper variant="outlined" sx={{ display: 'flex', flexDirection: 'column' }}>
      <Stack
        direction="row"
        alignItems="center"
        sx={{ px: 1, pt: 1, borderBottom: 1, borderColor: 'divider' }}
      >
        <Tabs
          value={topTab}
          onChange={(_, v: TopTab) => handleTopTab(v)}
          sx={{ flex: 1, minHeight: 40 }}
        >
          <Tab
            value="stdout"
            label={
              <Badge
                color="primary"
                variant="dot"
                invisible={!unreadTypes.has('stdout')}
              >
                <span>stdout</span>
              </Badge>
            }
          />
          <Tab
            value="stderr"
            label={
              <Badge
                color="primary"
                variant="dot"
                invisible={!unreadTypes.has('stderr')}
              >
                <span>stderr</span>
              </Badge>
            }
          />
          <Tab
            value="events"
            label={
              <Badge color="primary" variant="dot" invisible={!unreadEvents}>
                <span>Execution events</span>
              </Badge>
            }
          />
        </Tabs>
        <Stack direction="row" alignItems="center" spacing={1} sx={{ pr: 1 }}>
          {badgeStatus && <StatusBadge status={badgeStatus} />}
          <Tooltip
            title={
              running
                ? 'Line cap applies to finished task logs only'
                : 'Limit how many lines are loaded from the server'
            }
          >
            <FormControl size="small" sx={{ minWidth: 96 }} disabled={running}>
              <Select
                value={logTailChoice}
                onChange={(event) =>
                  handleLogTailChange(event.target.value as LogTailLineChoice)
                }
                aria-label="Log lines to show"
                disabled={running}
                renderValue={(value) => (
                  <Typography variant="body2" component="span">
                    {value === 'all' ? 'All lines' : `Last ${value}`}
                  </Typography>
                )}
                sx={{
                  '& .MuiSelect-select': {
                    py: 0.75,
                    display: 'flex',
                    alignItems: 'center',
                  },
                }}
              >
                {LOG_TAIL_LINE_OPTIONS.map((option) => (
                  <MenuItem key={option.value} value={option.value}>
                    {option.label === 'All'
                      ? 'All lines'
                      : `Last ${option.label}`}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          </Tooltip>
          <FormControlLabel
            control={
              <Switch
                size="small"
                checked={wrap}
                onChange={(_, checked) => setWrap(checked)}
              />
            }
            label="Wrap"
            slotProps={{ typography: { variant: 'body2' } }}
          />
          <Tooltip title="Download log">
            <span>
              <IconButton
                size="small"
                onClick={handleDownload}
                disabled={topTab === 'events' || !currentPaneText}
                aria-label="Download log"
              >
                <DownloadIcon fontSize="small" />
              </IconButton>
            </span>
          </Tooltip>
        </Stack>
      </Stack>

      {error && (
        <Box sx={{ p: 1 }}>
          <StreamErrorBlock error={error} />
        </Box>
      )}

      <Box sx={{ flex: 1, minHeight: 0 }}>
        {topTab === 'events' ? (
          <ExecutionEventsPanel
            eventsByStep={eventsByStep}
            activeStep={activeStep}
            height={height}
          />
        ) : (
          <LogOutputPane text={currentPaneText} wrap={wrap} height={height} />
        )}
      </Box>

      <Box sx={{ borderTop: 1, borderColor: 'divider', px: 1 }}>
        <LogStepTabs
          steps={stepsForTabs}
          activeStep={activeStep}
          unreadSteps={unreadSteps}
          onSelect={handleStepSelect}
        />
      </Box>
    </Paper>
  );
}
