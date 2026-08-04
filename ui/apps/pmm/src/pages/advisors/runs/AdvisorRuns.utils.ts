import { AdvisorRun } from 'types/advisors.types';

// Formats a run's wall-clock length. Runs are recorded when they start and
// stamped when they finish, so this is real elapsed time, not the span between
// the first and last insight.
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

export const isRunning = (run: AdvisorRun): boolean => !run.finishedAt;
