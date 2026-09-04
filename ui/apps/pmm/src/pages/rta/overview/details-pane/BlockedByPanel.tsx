import Box from '@mui/material/Box';
import { alpha } from '@mui/material/styles';
import Divider from '@mui/material/Divider';
import Grid from '@mui/material/Grid';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import LockOutlinedIcon from '@mui/icons-material/LockOutlined';
import { Chip, CodeBlock } from '@percona/peak-ui';
import { FC } from 'react';
import { BlockingTransaction } from 'types/rta.types';
import { formatDurationSeconds, parseDuration } from 'utils/duration.utils';
import { Messages } from './BlockedByPanel.messages';

export interface Props {
  // Every transaction the agent reported as holding this statement up. May be empty: the
  // lock graph and the statement list are read milliseconds apart, so a statement can be
  // flagged as waiting with no holder in the same snapshot.
  blockers: BlockingTransaction[];
  // The lock the waiting statement asked for. A property of the waiter, identical across its
  // blockers, so it is passed once rather than read off whichever blocker leads the list.
  lockedTable?: string;
  lockedIndex?: string;
}

// MySQL reports an idle connection as "Sleep": it is inside an open transaction and is
// running nothing, so its statement is the one that took the lock rather than a current one.
const IDLE_COMMAND = 'Sleep';

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

// BlockedByPanel explains why a statement is stuck. The data arrives with the statement in
// the same collection cycle, so nothing here is fetched on open.
const BlockedByPanel: FC<Props> = ({ blockers, lockedTable, lockedIndex }) => {
  // Transactions that are not themselves waiting. Resolving those is what frees the
  // statement — but there can be several, and then no single one is the answer.
  const roots = blockers.filter((blocker) => blocker.root);
  const sole =
    roots.length === 1
      ? roots[0]
      : roots.length === 0 && blockers.length === 1
        ? blockers[0]
        : undefined;
  // With no single culprit, lead with whichever transaction is at the head of the chain so
  // the pane still shows a concrete statement, while the heading stays honest about the count.
  const primary = sole ?? roots[0] ?? blockers[0];
  const others = blockers.filter((blocker) => blocker !== primary);

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

  const isIdle = primary.blockingCommand === IDLE_COMMAND;
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
              : Messages.blockedByMany(blockers.length)}
          </Typography>
        </Stack>
        <Stack direction="row" alignItems="center" gap={1.5}>
          {/* Suppressed rather than rendered as a dangling "waiting " when the agent
              reported no duration. */}
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
        {primary.blockingQuery && (
          <Stack gap={1}>
            <Typography variant="caption" color="text.secondary">
              {Messages.blockerStatement}
            </Typography>
            <CodeBlock
              language="sql"
              wrap
              copyable
              content={primary.blockingQuery}
              data-testid="blocker-query"
            />
            {isIdle && (
              <Typography variant="caption" color="text.secondary">
                {Messages.idleNote}
              </Typography>
            )}
          </Stack>
        )}

        <Grid container spacing={2}>
          <Fact
            title={Messages.titles.blockerState}
            value={
              isIdle && blockerAge
                ? Messages.idleInTransaction(blockerAge)
                : primary.blockingCommand
            }
          />
          <Fact
            title={Messages.titles.blockerUser}
            value={primary.blockingUsername}
          />
          <Fact title={Messages.titles.lockedTable} value={lockedTable} />
          <Fact title={Messages.titles.lockedIndex} value={lockedIndex} />
        </Grid>

        {others.length > 0 && (
          <>
            <Divider />
            <Stack gap={1}>
              <Typography variant="caption" color="text.secondary">
                {Messages.otherBlockers(others.length)}
              </Typography>
              <Stack gap={0.5}>
                {others.map((blocker) => (
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
                    {blocker.root && (
                      <Chip color="warning" label={Messages.root} />
                    )}
                  </Stack>
                ))}
              </Stack>
            </Stack>
          </>
        )}

        <Divider />
        <Typography variant="caption" color="text.secondary">
          {sole
            ? Messages.resolveHint(sole.blockingConnId)
            : roots.length > 1
              ? Messages.resolveHintRoots(roots.length)
              : Messages.resolveHintCycle}
        </Typography>
      </Stack>
    </PanelFrame>
  );
};

export default BlockedByPanel;
