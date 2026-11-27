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

import { useEffect, useRef, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  LinearProgress,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import { useNavigate } from 'react-router-dom';
import { OM_ROUTE_AUTOMATIONS } from '../constants';
import { useOmBase } from '../useOmBase';
import { OmError } from './OmError';
import {
  isOperationBusy,
  useCancelOperation,
  useConfirmOperation,
  useConfirmUninstallLivePlan,
  useOmOperation,
  useTriggerUninstall,
  useUninstallReport,
} from '../operationHooks';
import type {
  OmUninstallReadiness,
  OmUninstallReport,
  OmUninstallReportMember,
} from '../types';

/**
 * What each readiness verdict means to an operator. The two states read as progress; the
 * refusals say what to fix. Keyed by the backend's own codes so the dialog never invents a
 * verdict the server did not report.
 */
const READINESS_LABEL: Record<OmUninstallReadiness, string> = {
  ready: 'Ready to remove',
  checking: 'Checking…',
  marker_absent:
    'No PMM install marker on this host. PMM only removes what it installed, so it will not touch this one.',
  marker_mismatch:
    'This host now holds a different PMM install than the one being removed. Refusing, so a live deployment is never torn down.',
  host_unreachable: 'The host is unreachable, so it cannot be torn down.',
  cannot_elevate:
    'The automation agent cannot elevate on this host, so it cannot remove the files.',
  drift:
    'The installed packages changed since PMM installed them. Adopt the live set to proceed.',
};

const REFUSALS: readonly OmUninstallReadiness[] = [
  'marker_absent',
  'marker_mismatch',
  'host_unreachable',
  'cannot_elevate',
];

/** A member's readiness as a status chip: the colour marks the state, never carries the message. */
const ReadinessChip = ({ readiness }: { readiness: OmUninstallReadiness }) => {
  if (readiness === 'ready') {
    return (
      <Chip size="small" color="success" variant="outlined" label="Ready" />
    );
  }
  if (readiness === 'checking') {
    return <Chip size="small" label="Checking…" />;
  }
  if (readiness === 'drift') {
    return <Chip size="small" color="warning" label="Packages changed" />;
  }
  return <Chip size="small" color="error" label="Cannot remove" />;
};

/** A member's one-line summary: host, a readiness chip, and what the teardown deletes. */
const MemberRow = ({ member }: { member: OmUninstallReportMember }) => {
  const paths = [
    member.data_path,
    member.config_file,
    member.keyfile_path,
    member.log_path,
  ].filter((path): path is string => Boolean(path));
  return (
    <Box sx={{ border: 1, borderColor: 'divider', borderRadius: 1, p: 1.5 }}>
      <Stack
        direction="row"
        justifyContent="space-between"
        alignItems="center"
        spacing={1}
      >
        <Typography variant="subtitle2">{member.host}</Typography>
        <ReadinessChip readiness={member.readiness} />
      </Stack>
      {member.packages.length > 0 && (
        <Typography
          variant="caption"
          color="text.secondary"
          sx={{ display: 'block' }}
        >
          Packages: {member.packages.join(', ')}
        </Typography>
      )}
      {paths.length > 0 && (
        <Typography
          variant="caption"
          color="text.secondary"
          sx={{ display: 'block', wordBreak: 'break-all' }}
        >
          Deletes: {paths.join(', ')}
        </Typography>
      )}
    </Box>
  );
};

/** The readiness report body: the members, the no-backup notice, and any refusals. */
const ReportBody = ({ report }: { report: OmUninstallReport }) => {
  const refused = report.members.filter((member) =>
    REFUSALS.includes(member.readiness)
  );
  return (
    <Stack spacing={1.5}>
      <Typography variant="body2">
        This removes the cluster{' '}
        <strong>{report.cluster_name ?? report.cluster_id}</strong> from every
        host below. The members are stopped and the packages, data and config
        PMM installed are deleted. This cannot be undone.
      </Typography>
      <Alert severity="warning">No backup is taken or verified first.</Alert>
      {refused.length > 0 && (
        <OmError
          placement="item"
          title={
            refused.length === 1
              ? 'One member cannot be removed yet'
              : `${refused.length} members cannot be removed yet`
          }
          messages={refused.map(
            (member) => `${member.host}: ${READINESS_LABEL[member.readiness]}`
          )}
        />
      )}
      <Stack spacing={1}>
        {report.members.map((member) => (
          <MemberRow key={member.member_id} member={member} />
        ))}
      </Stack>
    </Stack>
  );
};

/**
 * The dialog that triggers an uninstall, polls it to a readiness report, and confirms it.
 *
 * It follows the operation lifecycle, not a one-shot delete: opening it triggers the
 * operation once (to `checking`), it polls the report to `awaiting_confirmation`, renders
 * what will be removed, and only a typed confirm phrase releases it. Dismissing before the
 * teardown runs cancels the operation so it does not hold the cluster; dismissing a running
 * one leaves the teardown alone.
 *
 * Confirming hands over to Automations rather than watching the teardown here, as an
 * install does: the dialog closes and the Uninstalls tab opens with this operation
 * expanded, so the teardown's steps stay readable after the dialog is gone.
 */
export const UninstallDialog = ({
  clusterId,
  clusterName,
  onClose,
}: {
  clusterId: string;
  clusterName: string | null;
  onClose: () => void;
}) => {
  const trigger = useTriggerUninstall();
  const navigate = useNavigate();
  const omBase = useOmBase();
  const [operationId, setOperationId] = useState<string | null>(null);
  const triggered = useRef(false);
  const [typed, setTyped] = useState('');

  useEffect(() => {
    if (triggered.current) {
      return;
    }
    triggered.current = true;
    trigger.mutate(clusterId, {
      onSuccess: (response) => setOperationId(response.operation_id),
    });
    // The trigger fires exactly once per open; the mutation object is stable.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [clusterId]);

  const { data: opData } = useOmOperation(operationId);
  const operation = opData?.operation;
  const { data: report, isError: reportError } = useUninstallReport(
    operationId,
    true,
    opData?.operation?.version
  );
  const confirm = useConfirmOperation();
  const cancel = useCancelOperation();
  const adoptLivePlan = useConfirmUninstallLivePlan();

  const status = operation?.status;
  const running =
    status === 'running' || status === 'queued' || status === 'cancelling';
  const terminal =
    status === 'succeeded' ||
    status === 'cancelled' ||
    status === 'rolled_back' ||
    status === 'abandoned';

  const dismiss = () => {
    // Cancel an operation still holding the cluster, so a dismissed trigger does not block
    // the next attempt; never cancel a running teardown or one already settled.
    if (operation && isOperationBusy(operation) && !running) {
      cancel.mutate(
        { operationId: operation.id, version: operation.version },
        { onSettled: onClose }
      );
      return;
    }
    onClose();
  };

  const confirmReady =
    report?.ready === true &&
    operation !== undefined &&
    typed === report.confirm_phrase;

  const onConfirm = () => {
    if (!operation || !report) {
      return;
    }
    confirm.mutate(
      {
        operationId: operation.id,
        version: operation.version,
        confirmPhrase: report.confirm_phrase,
      },
      {
        onSuccess: () => {
          onClose();
          navigate(
            `${omBase}/${OM_ROUTE_AUTOMATIONS}?tab=uninstalls&expand=${encodeURIComponent(operation.id)}`
          );
        },
      }
    );
  };

  const onAdoptLivePlan = () => {
    if (!operation) {
      return;
    }
    adoptLivePlan.mutate({
      operationId: operation.id,
      version: operation.version,
    });
  };

  const busy = confirm.isPending || cancel.isPending || adoptLivePlan.isPending;
  const title = `Uninstall ${clusterName ?? 'cluster'}`;

  return (
    <Dialog open onClose={dismiss} maxWidth="sm" fullWidth>
      <DialogTitle>{title}</DialogTitle>
      <DialogContent dividers>
        {trigger.isError && (
          <OmError
            placement="item"
            title="Could not start the uninstall"
            messages={trigger.error.message}
          />
        )}
        {reportError && (
          <OmError
            placement="item"
            title="Could not load the readiness report"
            messages="Retrying…"
          />
        )}

        {terminal && (
          <Alert severity={status === 'succeeded' ? 'success' : 'info'}>
            {status === 'succeeded'
              ? 'The cluster was removed.'
              : `The uninstall is ${status?.replace('_', ' ')}.`}
          </Alert>
        )}

        {running && (
          <Stack spacing={1}>
            <Typography variant="body2">Uninstalling…</Typography>
            <LinearProgress />
          </Stack>
        )}

        {!running && !terminal && !report && (
          <Stack spacing={1}>
            <Typography variant="body2">Checking readiness…</Typography>
            <LinearProgress />
          </Stack>
        )}

        {!running && !terminal && report && (
          <Stack spacing={2}>
            <ReportBody report={report} />

            {report.drifted && (
              <Alert
                severity="warning"
                action={
                  <Button
                    color="warning"
                    variant="outlined"
                    size="small"
                    sx={{ whiteSpace: 'nowrap' }}
                    disabled={busy}
                    onClick={onAdoptLivePlan}
                  >
                    Adopt live packages
                  </Button>
                }
              >
                The installed packages changed since install. Adopt the live set
                to re-check and proceed.
              </Alert>
            )}

            {report.ready && (
              <Box>
                <Typography variant="body2" sx={{ mb: 1 }}>
                  Type <strong>{report.confirm_phrase}</strong> to confirm.
                </Typography>
                <TextField
                  fullWidth
                  size="small"
                  value={typed}
                  autoComplete="off"
                  placeholder={report.confirm_phrase}
                  onChange={(event) => setTyped(event.target.value)}
                />
              </Box>
            )}

            {confirm.isError && (
              <OmError
                placement="item"
                title="Could not confirm"
                messages={confirm.error.message}
              />
            )}
          </Stack>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={dismiss} disabled={busy}>
          {terminal ? 'Close' : 'Cancel'}
        </Button>
        {!running && !terminal && (
          <Button
            variant="contained"
            color="error"
            disabled={!confirmReady || busy}
            onClick={onConfirm}
          >
            Remove cluster
          </Button>
        )}
      </DialogActions>
    </Dialog>
  );
};
