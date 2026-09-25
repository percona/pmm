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

import CheckIcon from '@mui/icons-material/Check';
import CloseFullscreenIcon from '@mui/icons-material/CloseFullscreen';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import DownloadIcon from '@mui/icons-material/Download';
import FullscreenIcon from '@mui/icons-material/Fullscreen';
import FullscreenExitIcon from '@mui/icons-material/FullscreenExit';
import OpenInFullIcon from '@mui/icons-material/OpenInFull';
import Accordion from '@mui/material/Accordion';
import AccordionDetails from '@mui/material/AccordionDetails';
import AccordionSummary from '@mui/material/AccordionSummary';
import Alert from '@mui/material/Alert';
import Badge from '@mui/material/Badge';
import Box from '@mui/material/Box';
import Dialog from '@mui/material/Dialog';
import FormControl from '@mui/material/FormControl';
import FormControlLabel from '@mui/material/FormControlLabel';
import IconButton from '@mui/material/IconButton';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Paper from '@mui/material/Paper';
import Select from '@mui/material/Select';
import Stack from '@mui/material/Stack';
import Switch from '@mui/material/Switch';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { useEffect, useId, useMemo, useRef, useState } from 'react';
import { RUNNING_STATUSES, type TaskHistoryStatus } from '@sep/api';
import { useCopyToClipboard } from '../../hooks/useCopyToClipboard';
import { useExecutionEvents } from '../../hooks/useExecutionEvents';
import { useLogDownload } from '../../hooks/useLogDownload';
import {
  useTaskLogs,
  type FinishStatus,
  type LogType,
  type StepText,
} from '../../hooks/useTaskLogs';
import { ExecutionEventsPanel } from './ExecutionEventsPanel';
import { LogOutputPane } from './LogOutputPane';
import { LogStepTabs } from './LogStepTabs';
import { StatusBadge, type BadgeStatus } from './StatusBadge';
import { StreamErrorBlock } from './StreamErrorBlock';
import { NON_FAILURE_TERMINAL_NOTES } from './terminalRunNotes';

type TopTab = 'stdout' | 'stderr';

/** A tab label wide enough to lose no character of "stdout" or "stderr". */
const TAB_LABEL_SX = { whiteSpace: 'nowrap' } as const;

/** Execution events carry no unread indicator; a stable empty set for that prop. */
const NO_UNREAD_STEPS: Set<string> = new Set();

export const DEFAULT_LOG_TAIL_LINES = 1000;

export const LOG_TAIL_LINE_OPTIONS = [
  { label: '100', value: '100' },
  { label: '1000', value: '1000' },
  { label: '5000', value: '5000' },
  { label: 'All', value: 'all' },
] as const;

export type LogTailLineChoice = (typeof LOG_TAIL_LINE_OPTIONS)[number]['value'];

const NUMERIC_LOG_TAIL_OPTIONS = LOG_TAIL_LINE_OPTIONS.map((option) =>
  Number(option.value)
).filter((value) => Number.isFinite(value));

/**
 * Smallest numeric cap on offer. A proven-complete log at or below this size
 * looks identical under every option, so the select has nothing left to do.
 * Derived from the options list so changing the list moves the threshold.
 *
 * Falls back to 0 when the list holds no numeric option: `Math.min()` of an
 * empty list is Infinity, which would hide the select for every finished task.
 */
const SMALLEST_LOG_TAIL_OPTION =
  NUMERIC_LOG_TAIL_OPTIONS.length > 0
    ? Math.min(...NUMERIC_LOG_TAIL_OPTIONS)
    : 0;

/**
 * LazyLog's own default row height, in pixels. Mirrored here rather than
 * imported because the library exposes it only as a prop default; the pane
 * relies on it to turn a line count into a pixel height, so a library change
 * would need this constant changed with it.
 */
const LOG_ROW_HEIGHT_PX = 19;

/**
 * Room for the search bar LazyLog renders above its rows. Its own
 * `SEARCH_BAR_HEIGHT`, which the library does not export — a pixel short here
 * clips the last row of a report that would otherwise have fit exactly.
 */
const LOG_PANE_CHROME_PX = 45;

/**
 * Shortest the pane ever gets. A two-line report still needs to look like a
 * pane rather than a stray sentence, and the search bar has to fit.
 */
const MIN_PANE_HEIGHT_PX = 140;

/** The cap the expand toggle raises the pane to — about a screen of output. */
const EXPANDED_PANE_HEIGHT_PX = 900;

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
  /**
   * Tallest the output pane gets before the user expands it. A number is a
   * ceiling the pane is fitted within, so a report shorter than it renders at
   * its own height rather than in a box padded out with blank space. A CSS
   * string cannot be fitted against without measuring, so it is applied as-is
   * and the expand toggle goes away with it.
   */
  maxHeight?: number | string;
  /** Mid-sentence singular noun for one record (e.g. `backup`). */
  itemName?: string;
}

/**
 * The terminal statuses a `finish` frame is supposed to carry. The frame is
 * parsed without validation, and the backend does send `finish` with a
 * non-terminal status (e.g. `running`) when it reconciles a run whose
 * allocation is placed but no step has started, so a live log is only treated
 * as complete when its status is one of these. Keyed on the union so a new
 * member cannot be added without deciding it here.
 */
const TERMINAL_FINISH_STATUS: Record<FinishStatus, true> = {
  success: true,
  failed: true,
  stopped: true,
  lost: true,
  stale: true,
  unlaunchable: true,
};

/** Where the uncapped stream a running history opened stands. */
interface LiveLog {
  historyId: TaskLogViewerProps['taskHistoryId'];
  state: 'open' | 'complete' | 'ended';
}

/**
 * Whether a loosely-typed status means the run is still going.
 *
 * `taskStatus` arrives as a bare string, so the canonical set from `@sep/api`
 * is probed rather than compared against a literal. That set counts `pending`
 * as running, which the local comparison this replaces did not: a run in the
 * seconds between launch and first output was treated as finished, so its
 * execution events came over one-shot REST and its log never streamed —
 * exactly the window someone watching a backup start is looking at.
 *
 * The case fold is kept from that comparison. The enum is lower-case on the
 * wire, but the prop is typed as a bare string and callers outside the
 * generated client have passed other casings.
 */
function isRunningStatus(status?: string): boolean {
  return RUNNING_STATUSES.has(
    (status ?? '').toLowerCase() as TaskHistoryStatus
  );
}

/**
 * Line count, saturating at `limit + 1`. Callers only need to know whether a
 * pane is over the threshold, so a large log stops being scanned as soon as it
 * provably is — no full pass over megabytes of "All lines" output.
 */
function countLinesUpTo(text: string, limit: number): number {
  if (text === '') {
    return 0;
  }
  let lines = 0;
  for (let index = 0; index < text.length; index += 1) {
    if (text[index] === '\n') {
      lines += 1;
      if (lines > limit) {
        return lines;
      }
    }
  }
  // A trailing fragment without its newline is still a line on screen.
  return text.endsWith('\n') ? lines : lines + 1;
}

/**
 * Largest line count across every step and both streams, saturating at
 * `limit + 1`. The line-cap decision uses this rather than the visible pane so
 * the control does not appear and disappear as the user moves between step or
 * stream tabs.
 */
function maxPaneLineCountUpTo(
  textByStep: Record<string, StepText>,
  limit: number
): number {
  let max = 0;
  for (const pane of Object.values(textByStep)) {
    max = Math.max(
      max,
      countLinesUpTo(pane.stdout, limit),
      countLinesUpTo(pane.stderr, limit)
    );
    if (max > limit) {
      return max;
    }
  }
  return max;
}

/**
 * Pixel height the pane wants for `text`, before any cap is applied.
 *
 * Counts newlines only, so a wrapped line is measured as one row. That
 * under-measures a report full of very long lines, which then scrolls inside a
 * pane shorter than its content — the same behaviour as before this sizing
 * existed, and the honest alternative would mean measuring the rendered pane
 * width and re-measuring on every resize. Over-measuring is the failure worth
 * avoiding: it leaves blank space under a short report, which is exactly the
 * complaint the fixed height caused.
 */
function fitPaneHeight(text: string, maxHeight: number): number {
  const rowsInCap = Math.ceil(maxHeight / LOG_ROW_HEIGHT_PX);
  const lines = countLinesUpTo(text, rowsInCap);
  return lines * LOG_ROW_HEIGHT_PX + LOG_PANE_CHROME_PX;
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

/**
 * The stream to open on when nothing has chosen one yet.
 *
 * Prefers stdout — most scripts' interesting output goes there — and falls
 * back to stderr only when the active step's stdout is empty and its stderr
 * is not, so a script that wrote its failure to stderr does not read "No
 * output" on a finished run while the real reason sits one tab over.
 */
function preferredTopTab(
  activeStep: string | undefined,
  textByStep: Record<string, StepText>
): TopTab {
  if (!activeStep) {
    return 'stdout';
  }
  const pane = textByStep[activeStep];
  if (pane && pane.stdout === '' && pane.stderr !== '') {
    return 'stderr';
  }
  return 'stdout';
}

export function TaskLogViewer({
  taskHistoryId,
  taskStatus,
  maxHeight = 480,
  itemName = 'task',
}: TaskLogViewerProps) {
  const running = isRunningStatus(taskStatus);
  const logTailLabelId = useId();
  const [logTailChoice, setLogTailChoice] = useState<LogTailLineChoice>(
    readStoredLogTailChoice
  );
  // The uncapped stream a running history opened. Re-fetching it capped once
  // the polled status turns terminal only blanks the pane and loses the scroll
  // position, and that status can arrive before the stream's own `finish`, so
  // the stream is kept until it ends. Ending with a terminal `finish` means it
  // already holds the whole log and is kept for good; a stream cut short, or
  // whose `finish` is non-terminal, is reloaded once the run is over. Under
  // the "All" cap that reload has the same history and tail as the live
  // stream, so it is requested explicitly rather than through the tail.
  const [liveLog, setLiveLog] = useState<LiveLog | null>(null);
  const liveLogState =
    liveLog !== null && liveLog.historyId === taskHistoryId
      ? liveLog.state
      : undefined;
  const keepLiveLog = liveLogState === 'open' || liveLogState === 'complete';
  const reloadLiveLog = liveLogState === 'ended' && !running;
  const tailLines = logTailChoiceToParam(logTailChoice);
  const effectiveTailLines = running || keepLiveLog ? undefined : tailLines;
  const { textByStep, stepOrder, streamStatus, finishStatus, error } =
    useTaskLogs(taskHistoryId, effectiveTailLines, reloadLiveLog ? 1 : 0);

  const liveLogHistoryIdRef = useRef(taskHistoryId);
  useEffect(() => {
    // On the render that switches histories, the stream state still belongs
    // to the previous stream: useTaskLogs only resets it in this same commit.
    if (liveLogHistoryIdRef.current !== taskHistoryId) {
      liveLogHistoryIdRef.current = taskHistoryId;
      setLiveLog(running ? { historyId: taskHistoryId, state: 'open' } : null);
      return;
    }
    // Only the stream opened while running is tracked, and it settles once:
    // the reload that follows an `ended` stream must not be taken for it.
    if (streamStatus === 'finished' || streamStatus === 'error') {
      if (liveLogState !== 'open') {
        return;
      }
      const complete =
        streamStatus === 'finished' &&
        finishStatus !== undefined &&
        Object.prototype.hasOwnProperty.call(
          TERMINAL_FINISH_STATUS,
          finishStatus
        );
      setLiveLog({
        historyId: taskHistoryId,
        state: complete ? 'complete' : 'ended',
      });
    } else if (running && liveLogState === undefined) {
      setLiveLog({ historyId: taskHistoryId, state: 'open' });
    }
  }, [running, streamStatus, finishStatus, taskHistoryId, liveLogState]);
  const { eventsByStep, stepOrder: eventStepOrder } = useExecutionEvents(
    taskHistoryId,
    running
  );

  // `undefined` means the stream/step choice has not been overridden by a
  // click, so the effective tab below keeps auto-picking whichever of
  // stdout/stderr has content. Any explicit click fixes it from then on.
  const [manualTopTab, setManualTopTab] = useState<TopTab | undefined>();
  const [activeStep, setActiveStep] = useState<string | undefined>();
  const [wrap, setWrap] = useState(true);
  const [expanded, setExpanded] = useState(false);
  const [fullScreen, setFullScreen] = useState(false);

  const [unreadTypes, setUnreadTypes] = useState<Set<LogType>>(new Set());
  const [unreadSteps, setUnreadSteps] = useState<Set<string>>(new Set());

  // Execution events keep their own step selection, independent of the log
  // pane's: an event can fire for a step that never wrote a log line, so
  // sharing one step state would make that step's events unreachable.
  const [activeEventStep, setActiveEventStep] = useState<string | undefined>();

  const prevLogSizesRef = useRef<Record<string, number>>({});

  // Reset view state when switching to a different task history
  useEffect(() => {
    setManualTopTab(undefined);
    setActiveStep(undefined);
    setActiveEventStep(undefined);
    setUnreadTypes(new Set());
    setUnreadSteps(new Set());
    setExpanded(false);
    setFullScreen(false);
    prevLogSizesRef.current = {};
  }, [taskHistoryId]);

  useEffect(() => {
    if (!activeStep && stepOrder.length > 0) {
      setActiveStep(stepOrder[0]);
    }
  }, [stepOrder, activeStep]);

  const topTab: TopTab =
    manualTopTab ?? preferredTopTab(activeStep, textByStep);

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

  const handleTopTab = (value: TopTab) => {
    setManualTopTab(value);
    setUnreadTypes((prev) => {
      if (!prev.has(value)) {
        return prev;
      }
      const next = new Set(prev);
      next.delete(value);
      return next;
    });
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

  const currentPaneText = useMemo(() => {
    if (!activeStep) {
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
    const filename = `task-${taskHistoryId}-${activeStep ?? 'step'}-${topTab}.log`;
    download(filename, currentPaneText);
  };

  const clipboard = useCopyToClipboard();
  const handleCopy = () => {
    clipboard.copy(currentPaneText);
  };

  const fittable = typeof maxHeight === 'number';
  const baseMaxHeight = fittable ? maxHeight : 0;
  const expandedMaxHeight = Math.max(baseMaxHeight, EXPANDED_PANE_HEIGHT_PX);
  const maxPaneHeight = expanded ? expandedMaxHeight : baseMaxHeight;
  const wantedPaneHeight = fittable
    ? fitPaneHeight(currentPaneText, maxPaneHeight)
    : 0;
  // Only the unexpanded ceiling is asked about: once expanded, the toggle's job
  // is to offer the way back regardless of how much content is left over.
  const contentOverflows = fittable && wantedPaneHeight > baseMaxHeight;

  const paneHeight = fullScreen
    ? '100%'
    : fittable
      ? Math.min(maxPaneHeight, Math.max(MIN_PANE_HEIGHT_PX, wantedPaneHeight))
      : maxHeight;

  const handleLogTailChange = (choice: LogTailLineChoice) => {
    // Only a cap has to clear the live-log record to be fetched again. "All"
    // keeps it tracked, so a stream still open after the run ends is reloaded
    // if it ends cut short.
    if (logTailChoiceToParam(choice) !== undefined) {
      setLiveLog(null);
    }
    setLogTailChoice(choice);
    if (globalThis.localStorage !== undefined) {
      globalThis.localStorage.setItem(LOG_TAIL_STORAGE_KEY, choice);
    }
  };

  const badgeStatus = resolveBadgeStatus(finishStatus, error);

  // A run is "finished" once the stream itself says so (a `finish` SSE event
  // carrying a terminal status) or, absent that, once the caller's own status
  // prop says the run is not running — the case for a viewer just mounted
  // against an already-terminal history row, before its stream has caught up.
  //
  // Picks the empty-pane wording ("No output" reads as final, where "No output
  // yet." promises more may still arrive) and decides whether the pane tails.
  // Tailing deliberately keys on this rather than on `running`: `taskStatus` is
  // optional, and `SnippetExecutionAccordion` mounts the viewer against a
  // just-launched task without it, so a live log would otherwise sit still
  // while output arrived off-screen. Reading the stream instead also stops the
  // tail the moment a `finish` event lands, which the status prop — frozen at
  // whatever the caller last rendered — would never do.
  const hasFinished =
    Boolean(finishStatus) || (taskStatus !== undefined && !running);

  const nonFailureNote = finishStatus
    ? NON_FAILURE_TERMINAL_NOTES[finishStatus]
    : undefined;

  // Hide the line cap once a finished history has streamed a log that is
  // provably complete and short enough that every option would show the same
  // thing. Gated on the terminal stream status so the control does not flicker
  // while lines are still arriving.
  const showLogTailSelect = useMemo(() => {
    if (running || streamStatus !== 'finished') {
      return true;
    }
    // Saturated at one over the threshold: any pane above it keeps the select
    // regardless of the requested cap, so the exact count no longer matters.
    const maxLines = maxPaneLineCountUpTo(textByStep, SMALLEST_LOG_TAIL_OPTION);
    if (maxLines > SMALLEST_LOG_TAIL_OPTION) {
      return true;
    }
    // A pane sitting exactly at the requested cap may have been trimmed
    // server-side, so only a count strictly below the cap proves completeness.
    return effectiveTailLines !== undefined && maxLines >= effectiveTailLines;
  }, [running, streamStatus, textByStep, effectiveTailLines]);

  const body = (
    <Paper
      variant="outlined"
      sx={{
        display: 'flex',
        flexDirection: 'column',
        // Full screen the Paper owns the dialog surface: no rounded corners or
        // border floating against the viewport edge, and a definite height so
        // the output pane's `100%` has something to resolve against.
        ...(fullScreen && { height: '100%', border: 0, borderRadius: 0 }),
      }}
    >
      <Stack
        direction="row"
        alignItems="center"
        flexWrap="wrap"
        rowGap={1}
        sx={{ px: 1, pt: 1, borderBottom: 1, borderColor: 'divider' }}
      >
        <Tabs
          value={topTab}
          onChange={(_, v: LogType) => handleTopTab(v)}
          sx={{ minHeight: 40, flexShrink: 0 }}
        >
          <Tab
            value="stdout"
            sx={TAB_LABEL_SX}
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
            sx={TAB_LABEL_SX}
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
        </Tabs>
        <Box sx={{ flex: 1 }} />
        <Stack direction="row" alignItems="center" spacing={1} sx={{ pr: 1 }}>
          {badgeStatus && <StatusBadge status={badgeStatus} />}
          {showLogTailSelect && (
            <Tooltip
              title={
                running
                  ? `Line cap applies to finished ${itemName} logs only`
                  : 'Limit how many lines are loaded from the server'
              }
            >
              <FormControl
                size="small"
                sx={{ minWidth: 128 }}
                disabled={running}
              >
                {/*
                  A visible label, not just the `aria-label` this replaced: on
                  its own, "Last 1000" names neither what is being counted nor
                  that it can be changed.
                */}
                <InputLabel id={logTailLabelId}>Lines loaded</InputLabel>
                <Select
                  labelId={logTailLabelId}
                  label="Lines loaded"
                  value={logTailChoice}
                  onChange={(event) =>
                    handleLogTailChange(event.target.value as LogTailLineChoice)
                  }
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
          )}
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
          <Tooltip
            title={
              clipboard.copied
                ? 'Copied'
                : clipboard.failed
                  ? 'Could not copy — use Download instead'
                  : 'Copy this output'
            }
          >
            <span>
              <IconButton
                size="small"
                onClick={handleCopy}
                disabled={!currentPaneText}
                aria-label={clipboard.copied ? 'Copied' : 'Copy log'}
                color={clipboard.failed ? 'error' : undefined}
              >
                {clipboard.copied ? (
                  <CheckIcon fontSize="small" color="success" />
                ) : (
                  <ContentCopyIcon fontSize="small" />
                )}
              </IconButton>
            </span>
          </Tooltip>
          <Tooltip title="Download log">
            <span>
              <IconButton
                size="small"
                onClick={handleDownload}
                disabled={!currentPaneText}
                aria-label="Download log"
              >
                <DownloadIcon fontSize="small" />
              </IconButton>
            </span>
          </Tooltip>
          {!fullScreen && (
            <Tooltip
              title={
                expanded
                  ? 'Collapse to the default height'
                  : contentOverflows
                    ? 'Expand the output'
                    : 'The whole output already fits'
              }
            >
              <span>
                <IconButton
                  size="small"
                  onClick={() => setExpanded((previous) => !previous)}
                  disabled={!expanded && !contentOverflows}
                  aria-label={expanded ? 'Collapse output' : 'Expand output'}
                >
                  {expanded ? (
                    <CloseFullscreenIcon fontSize="small" />
                  ) : (
                    <OpenInFullIcon fontSize="small" />
                  )}
                </IconButton>
              </span>
            </Tooltip>
          )}
          <Tooltip title={fullScreen ? 'Exit full screen' : 'Full screen'}>
            <IconButton
              size="small"
              onClick={() => setFullScreen((previous) => !previous)}
              aria-label={fullScreen ? 'Exit full screen' : 'Full screen'}
            >
              {fullScreen ? (
                <FullscreenExitIcon fontSize="small" />
              ) : (
                <FullscreenIcon fontSize="small" />
              )}
            </IconButton>
          </Tooltip>
        </Stack>
      </Stack>

      {error && (
        <Box sx={{ p: 1 }}>
          <StreamErrorBlock error={error} />
        </Box>
      )}

      {!error && nonFailureNote && (
        <Box sx={{ p: 1 }}>
          <Alert severity="warning" data-testid="task-log-viewer-note">
            {nonFailureNote}
          </Alert>
        </Box>
      )}

      <Box sx={{ borderBottom: 1, borderColor: 'divider', px: 1 }}>
        <LogStepTabs
          steps={stepOrder}
          activeStep={activeStep}
          unreadSteps={unreadSteps}
          onSelect={handleStepSelect}
        />
      </Box>

      <Box sx={{ flex: fullScreen ? 1 : 'none', minHeight: 0 }}>
        <LogOutputPane
          text={currentPaneText}
          wrap={wrap}
          height={paneHeight}
          follow={!hasFinished}
          emptyLabel={hasFinished ? 'No output' : 'No output yet.'}
        />
      </Box>

      <Accordion
        disableGutters
        sx={{ flexShrink: 0, '&:before': { display: 'none' } }}
      >
        <AccordionSummary expandIcon={<ExpandMoreIcon />}>
          <Typography variant="body2" color="text.secondary">
            Technical details
          </Typography>
        </AccordionSummary>
        <AccordionDetails sx={{ p: 0 }}>
          <Box sx={{ borderBottom: 1, borderColor: 'divider', px: 1 }}>
            <LogStepTabs
              steps={eventStepOrder}
              activeStep={activeEventStep}
              unreadSteps={NO_UNREAD_STEPS}
              onSelect={setActiveEventStep}
            />
          </Box>
          <ExecutionEventsPanel
            eventsByStep={eventsByStep}
            activeStep={activeEventStep}
            height={240}
          />
        </AccordionDetails>
      </Accordion>
    </Paper>
  );

  // Moving the whole viewer into the dialog — rather than overlaying a copy of
  // the pane — keeps one set of tabs, one search and one selected step, so
  // nothing has to be re-found after going full screen. The cost is that the
  // pane remounts across the transition, dropping whatever was typed into
  // LazyLog's search box.
  if (fullScreen) {
    return (
      <Dialog
        fullScreen
        open
        onClose={() => setFullScreen(false)}
        aria-label="Log output, full screen"
      >
        {body}
      </Dialog>
    );
  }

  return body;
}
