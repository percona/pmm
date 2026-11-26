import {
  AdvisorCheckTriggeredBy,
  type AdvisorRun,
  AdvisorRunStatus,
} from 'types/advisors.types';
import { TRIGGERED_BY_LABEL } from '../insights/AdvisorInsights.utils';
import { formatUtcTime } from '../scheduler/AdvisorScheduler.utils';
import { Messages } from './AdvisorRuns.messages';

// how often pages refresh while a run is in progress, to follow its progress
export const ACTIVE_RUN_POLL_INTERVAL_MS = 10_000;

// real elapsed time: runs are stamped on start and on finish
export const formatDuration = (run: AdvisorRun): string | null => {
  if (!run.finishedAt) {
    return null;
  }

  const ms =
    new Date(run.finishedAt).getTime() - new Date(run.startedAt).getTime();
  if (!Number.isFinite(ms) || ms < 0) {
    return null;
  }

  const totalSeconds = Math.round(ms / 1000);
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;

  if (hours > 0) {
    return `${hours}h ${minutes}m`;
  }
  if (minutes > 0) {
    return `${minutes}m ${String(seconds).padStart(2, '0')}s`;
  }
  return `${seconds}s`;
};

// queued runs count too: they start within a minute and the page keeps polling
export const isRunning = (run: AdvisorRun): boolean =>
  run.status === AdvisorRunStatus.queued ||
  run.status === AdvisorRunStatus.running;

// the schedule's name for a scheduled run, "User · <name>" for one started with Run now
export const triggeredByLabel = (run: AdvisorRun): string => {
  const actor = TRIGGERED_BY_LABEL[run.triggeredBy];
  if (!run.scheduleName) {
    return actor || '—';
  }
  return run.triggeredBy === AdvisorCheckTriggeredBy.user
    ? `${actor} · ${run.scheduleName}`
    : run.scheduleName;
};

const formatWait = (minutes: number): string => {
  const hours = Math.floor(minutes / 60);
  const rest = minutes % 60;
  if (!hours) {
    return `${minutes} min`;
  }
  return rest ? `${hours} h ${rest} min` : `${hours} h`;
};

// why a scheduled run started late; null for a run that started on time
export const waitedForText = (run: AdvisorRun): string | null => {
  const blocker = run.waitedForRun;
  if (!blocker || !run.scheduledFor) {
    return null;
  }
  const scheduled = new Date(run.scheduledFor);
  const waited = Math.round(
    (new Date(run.startedAt).getTime() - scheduled.getTime()) / 60_000
  );
  const who =
    blocker.scheduleName ||
    (blocker.triggeredBy === AdvisorCheckTriggeredBy.user
      ? Messages.tooltips.userRun
      : Messages.tooltips.scheduledRun);
  return Messages.tooltips.waited(
    formatUtcTime(scheduled),
    formatWait(waited),
    formatUtcTime(new Date(blocker.startedAt)),
    who
  );
};
