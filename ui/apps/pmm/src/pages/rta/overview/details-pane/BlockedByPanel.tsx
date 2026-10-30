import Box from '@mui/material/Box';
import { alpha } from '@mui/material/styles';
import Divider from '@mui/material/Divider';
import Grid from '@mui/material/Grid';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import LockOutlinedIcon from '@mui/icons-material/LockOutlined';
import { Chip, CodeBlock } from '@percona/peak-ui';
import { FC } from 'react';
import { BlockingTransaction, LockType } from 'types/rta.types';
import { ServiceType } from 'types/services.types';
import { formatDurationSeconds, parseDuration } from 'utils/duration.utils';
import {
  blockingRoots,
  soleBlockerOf,
} from 'pages/rta/overview/table/OverviewTable.utils';
import { Messages } from './BlockedByPanel.messages';
import { TruncatedChip } from 'pages/rta/components/truncated-chip';

export interface Props {
  // May be empty: the lock graph and the statement list are read milliseconds apart.
  blockers: BlockingTransaction[];
  // A property of the waiter, identical across its blockers.
  lockedTable?: string;
  lockedIndex?: string;
  lockType?: LockType;
  requestedLockMode?: string;
  technology?: ServiceType;
}

// Unknown lock types render no row rather than "Lock type: —".
const lockTypeLabel = (lockType?: LockType): string | undefined => {
  switch (lockType) {
    case LockType.row:
      return Messages.lockTypes.row;
    case LockType.metadata:
      return Messages.lockTypes.metadata;
    default:
      return undefined;
  }
};

// An idle connection's statement is its last one, not necessarily the one that took the lock.
// MySQL reports one as "Sleep", PostgreSQL as "idle" or "idle in transaction".
const IDLE_COMMAND = 'Sleep';
const isIdleCommand = (command: string) =>
  command === IDLE_COMMAND ||
  command === 'idle' ||
  command.startsWith('idle in transaction');

// LOCK TABLE and row locks need a transaction, so a PostgreSQL session idle outside one can only
// be holding a session-level lock, such as one from pg_advisory_lock().
const holdsSessionLock = (
  blocker: BlockingTransaction,
  technology?: ServiceType
) => technology === ServiceType.posgresql && blocker.blockingCommand === 'idle';

const durationText = (duration?: string | null): string =>
  duration ? formatDurationSeconds(parseDuration(duration) / 1000) : '';

const Fact: FC<{ title: string; value?: string }> = ({ title, value }) =>
  value ? (
    <Grid size={{ xs: 6 }}>
      <Stack gap={0.5}>
        <Typography variant="caption" color="text.secondary">
          {title}
        </Typography>
        <Typography variant="body2" fontFamily="Roboto Mono, monospace">
          {value}
        </Typography>
      </Stack>
    </Grid>
  ) : null;

const PanelFrame: FC<React.PropsWithChildren> = ({ children }) => (
  <Box
    data-testid="blocked-by-panel"
    sx={{
      border: 1,
      borderColor: 'warning.main',
      borderRadius: 1,
      overflow: 'hidden',
    }}
  >
    {children}
  </Box>
);

// Shows each blocker's mode so a chain of mixed modes reads as a sequence.
const BlockerList: FC<{ title: string; blockers: BlockingTransaction[] }> = ({
  title,
  blockers,
}) => (
  <Stack gap={1}>
    <Typography variant="caption" color="text.secondary">
      {title}
    </Typography>
    <Stack gap={0.5}>
      {blockers.map((blocker) => (
        <Stack
          key={String(blocker.blockingConnId)}
          direction="row"
          alignItems="center"
          gap={1}
        >
          <Typography
            variant="body2"
            fontFamily="Roboto Mono, monospace"
            color="text.secondary"
          >
            {blocker.blockingConnId}
          </Typography>
          {blocker.blockingLockMode && (
            <Typography
              variant="caption"
              fontFamily="Roboto Mono, monospace"
              color="text.secondary"
            >
              {blocker.blockingLockMode}
            </Typography>
          )}
          {blocker.root && <Chip color="warning" label={Messages.root} />}
        </Stack>
      ))}
    </Stack>
  </Stack>
);

// Nothing is fetched on open: the data arrives with the statement.
const BlockedByPanel: FC<Props> = ({
  blockers,
  lockedTable,
  lockedIndex,
  lockType,
  requestedLockMode,
  technology,
}) => {
  // Shared helpers, so this pane, the table chip and the CSV name the same transaction.
  const roots = blockingRoots(blockers);
  const sole = soleBlockerOf(blockers);
  // Heading and hint must quote the same number; non-roots are queued behind the roots.
  const culpritCount = roots.length > 1 ? roots.length : blockers.length;
  // With no single culprit, still lead with a concrete statement from the head of the chain.
  const primary = sole ?? roots[0] ?? blockers[0];
  // A second root must be resolved too, while a non-root clears on its own.
  const others = blockers.filter((blocker) => blocker !== primary);
  const otherRoots = others.filter((blocker) => blocker.root);
  // No root is a lock chain: InnoDB and MDL roll a real deadlock back at once.
  const queuedAhead = others.filter((blocker) => !blocker.root);

  if (!primary) {
    return (
      <PanelFrame>
        <Stack
          direction="row"
          alignItems="center"
          gap={1}
          sx={{
            px: 2,
            py: 1.5,
            backgroundColor: (theme) => alpha(theme.palette.warning.main, 0.12),
          }}
        >
          <LockOutlinedIcon fontSize="small" color="warning" />
          <Typography variant="body2" fontFamily="Poppins" fontWeight="600">
            {Messages.blockedUnknownHolder}
          </Typography>
        </Stack>
        <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
          {Messages.unknownBlocker}
        </Typography>
      </PanelFrame>
    );
  }

  const isIdle = isIdleCommand(primary.blockingCommand);
  const blockerAge = durationText(primary.blockerTransactionDuration);
  const waitText = durationText(primary.waitDuration);

  return (
    <PanelFrame>
      <Stack
        direction="row"
        alignItems="center"
        justifyContent="space-between"
        gap={1}
        sx={{
          px: 2,
          py: 1.5,
          backgroundColor: (theme) => alpha(theme.palette.warning.main, 0.12),
        }}
      >
        <Stack direction="row" alignItems="center" gap={1}>
          <LockOutlinedIcon fontSize="small" color="warning" />
          <Typography
            variant="body2"
            fontFamily="Poppins"
            fontWeight="600"
            data-testid="blocked-by-heading"
          >
            {sole
              ? Messages.blockedByOne(sole.blockingConnId)
              : Messages.blockedByMany(culpritCount)}
          </Typography>
        </Stack>
        <Stack direction="row" alignItems="center" gap={1.5}>
          {waitText && (
            <Typography
              variant="body2"
              fontFamily="Roboto Mono, monospace"
              color="text.secondary"
              data-testid="blocked-wait-duration"
            >
              {Messages.waitingFor(waitText)}
            </Typography>
          )}
        </Stack>
      </Stack>

      <Stack gap={2} sx={{ p: 2 }}>
        {/* An idle connection's last statement survives only in events_statements_current,
            disabled by default on MariaDB; a missing section would read as PMM losing it. */}
        {!primary.blockingQuery && (
          <Stack gap={1}>
            <Typography variant="caption" color="text.secondary">
              {Messages.blockerStatement}
            </Typography>
            <Typography
              variant="body2"
              color="text.secondary"
              data-testid="blocker-query-unavailable"
            >
              {isIdle && technology === ServiceType.mysql
                ? blockerAge
                  ? Messages.noStatementIdleInTransaction
                  : Messages.noStatementIdle
                : Messages.noStatement}
            </Typography>
          </Stack>
        )}
        {primary.blockingQuery && (
          <Stack gap={1}>
            <Stack direction="row" alignItems="center" gap={1}>
              <Typography variant="caption" color="text.secondary">
                {Messages.blockerStatement}
              </Typography>
              {primary.blockingQueryTruncated && (
                <TruncatedChip
                  dataTestId="blocker-query-truncated"
                  technology={technology}
                />
              )}
            </Stack>
            <CodeBlock
              language="sql"
              wrap
              copyable
              content={primary.blockingQuery}
              // Room for the floating copy button, which otherwise covers the first line's end.
              sx={{ pr: 5 }}
              data-testid="blocker-query"
            />
            {isIdle && (
              <Typography variant="caption" color="text.secondary">
                {blockerAge
                  ? Messages.idleNote
                  : Messages.idleNoteNoTransaction}
              </Typography>
            )}
          </Stack>
        )}

        <Grid container spacing={2}>
          <Fact
            title={Messages.titles.blockerState}
            value={
              isIdle && blockerAge
                ? Messages.idleInTransaction(
                    primary.blockingCommand,
                    blockerAge
                  )
                : primary.blockingCommand
            }
          />
          <Fact
            title={Messages.titles.blockerUser}
            value={primary.blockingUsername}
          />
          <Fact title={Messages.titles.lockedTable} value={lockedTable} />
          <Fact title={Messages.titles.lockedIndex} value={lockedIndex} />
          <Fact
            title={Messages.titles.lockType}
            value={lockTypeLabel(lockType)}
          />
          <Fact
            title={Messages.titles.requestedMode}
            value={requestedLockMode}
          />
          {/* With several blockers, each one's mode is listed below instead. */}
          {sole && (
            <Fact
              title={Messages.titles.blockingMode}
              value={sole.blockingLockMode}
            />
          )}
        </Grid>

        {others.length > 0 && (
          <>
            <Divider />
            <Stack gap={1.5}>
              {otherRoots.length > 0 && (
                <BlockerList
                  title={Messages.otherRootBlockers(otherRoots.length)}
                  blockers={otherRoots}
                />
              )}
              {queuedAhead.length > 0 && (
                <BlockerList
                  title={Messages.otherBlockers(queuedAhead.length)}
                  blockers={queuedAhead}
                />
              )}
            </Stack>
          </>
        )}

        <Divider />
        <Typography variant="caption" color="text.secondary">
          {sole
            ? !sole.root
              ? Messages.resolveHintWaiting(sole.blockingConnId)
              : lockType === LockType.metadata
                ? Messages.resolveHintMetadata(sole.blockingConnId)
                : holdsSessionLock(sole, technology)
                  ? Messages.resolveHintSessionLock(sole.blockingConnId)
                  : Messages.resolveHint(sole.blockingConnId)
            : roots.length > 1
              ? Messages.resolveHintRoots(culpritCount, queuedAhead.length)
              : Messages.resolveHintAllWaiting(culpritCount)}
        </Typography>
      </Stack>
    </PanelFrame>
  );
};

export default BlockedByPanel;
