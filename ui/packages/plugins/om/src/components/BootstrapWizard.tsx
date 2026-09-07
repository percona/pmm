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
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  Step,
  StepLabel,
  Stepper,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TextField,
  Typography,
} from '@mui/material';
import {
  BOOTSTRAP_RUN_COLOR,
  BOOTSTRAP_RUN_LABEL,
  BOOTSTRAP_STEP_COLOR,
  BOOTSTRAP_STEP_LABEL,
} from '../constants';
import { isHostRollingBack } from '../api';
import { useBootstrapRun, useTriggerHostBootstrap } from '../inventoryHooks';
import type {
  OmBootstrapHost,
  OmBootstrapStep,
  OmGetBootstrapRunResponse,
  OmHostRow,
} from '../types';

const DEFAULT_MONGODB_VERSION = '7.0';

/** The wizard's own steps, matching the ticket's mockups (PMM-15347/plan.md §2.5). */
const WIZARD_STEPS = ['Hosts', 'Configure', 'Review', 'Bootstrap'] as const;

/** Adamo's decided phase-1 topology: a replica set of exactly one or three members. */
function isSupportedHostCount(count: number): boolean {
  return count === 1 || count === 3;
}

const StepChip = ({ step }: { step: OmBootstrapStep }) => (
  <Chip
    size="small"
    label={`${step.name}: ${BOOTSTRAP_STEP_LABEL[step.status] ?? step.status}${
      step.attempt_count > 1 ? ` (attempt ${step.attempt_count})` : ''
    }`}
    color={BOOTSTRAP_STEP_COLOR[step.status] ?? 'default'}
    variant={step.status === 'running' ? 'outlined' : 'filled'}
    title={step.detail ?? undefined}
  />
);

/**
 * One host's progress: its own steps ordinarily, or its rollback steps once
 * the run has actually started tearing it down - never both at once, since a
 * host being rolled back has nothing left to show from its forward attempt
 * that the rollback list doesn't already explain.
 */
const HostProgress = ({ host }: { host: OmBootstrapHost }) => {
  const rollingBack = isHostRollingBack(host);
  return (
    <Box>
      <Typography variant="subtitle2" gutterBottom>
        {host.host}
        {rollingBack ? ' — rolling back' : ''}
      </Typography>
      <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap">
        {(rollingBack ? host.rollback_steps : host.steps).map((step) => (
          <StepChip key={step.name} step={step} />
        ))}
      </Stack>
    </Box>
  );
};

/** A run's full progress: its own status, every host, and the run-level steps. */
const RunProgress = ({ run }: { run: OmGetBootstrapRunResponse }) => (
  <Stack spacing={2}>
    <Alert
      severity={
        run.status === 'succeeded'
          ? 'success'
          : run.status === 'failed' || run.status === 'rolled_back'
            ? 'error'
            : 'info'
      }
    >
      Run {run.run_id}:{' '}
      <Chip
        size="small"
        label={BOOTSTRAP_RUN_LABEL[run.status] ?? run.status}
        color={BOOTSTRAP_RUN_COLOR[run.status] ?? 'default'}
      />
      {run.error ? ` — ${run.error}` : ''}
    </Alert>
    {run.hosts.map((host) => (
      <HostProgress key={host.host} host={host} />
    ))}
    {run.run_steps.length > 0 && (
      <Box>
        <Typography variant="subtitle2" gutterBottom>
          Replica set
        </Typography>
        <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap">
          {run.run_steps.map((step) => (
            <StepChip key={step.name} step={step} />
          ))}
        </Stack>
      </Box>
    )}
  </Stack>
);

/**
 * Configure -> Review -> Bootstrap for a set of hosts already selected on
 * {@link HostsPage} — the wizard's own "Hosts" step is a read-only recap of
 * that selection, not a second place to make it, so this dialog never
 * duplicates the eligibility table it was opened from.
 *
 * PMM-15347 PoC only: one or three hosts, keyFile auth, TLS off, no project
 * or cluster. Once bootstrap is triggered the dialog stays open on its own
 * "Bootstrap" step polling {@link useBootstrapRun} live, rather than closing
 * on submit the way a plain form would - watching it land is the point.
 */
export const BootstrapWizardDialog = ({
  hosts,
  onClose,
  onBootstrapped,
}: {
  hosts: OmHostRow[];
  onClose: () => void;
  onBootstrapped: () => void;
}) => {
  const [activeStep, setActiveStep] = useState(0);
  const [replicaSetName, setReplicaSetName] = useState('');
  const [mongodbVersion, setMongodbVersion] = useState(DEFAULT_MONGODB_VERSION);
  const [runId, setRunId] = useState<string | null>(null);
  const bootstrap = useTriggerHostBootstrap();
  const run = useBootstrapRun(runId);

  // A fresh selection (a new open, not a re-render of the same one) resets the
  // wizard back to its first step - opening it on a different host set must not
  // carry over a previous run id or an in-flight mutation's error.
  useEffect(() => {
    setActiveStep(0);
    setReplicaSetName('');
    setMongodbVersion(DEFAULT_MONGODB_VERSION);
    setRunId(null);
    bootstrap.reset();
    // bootstrap is a fresh object every render (useMutation), so it is deliberately
    // left out of the dependency list - including it would reset the wizard on every
    // keystroke-triggered re-render, not just on a genuinely new selection.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [hosts]);

  if (hosts.length === 0) {
    return null;
  }

  const handleTriggerBootstrap = async () => {
    const accepted = await bootstrap.mutateAsync({
      nodeIds: hosts.map((host) => host.node_id),
      replicaSetName,
      mongodbVersion,
    });
    setRunId(accepted.run_id);
    setActiveStep(3);
  };

  return (
    <Dialog open onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>
        Bootstrap {hosts.length === 1 ? hosts[0].name : `${hosts.length} hosts`}
      </DialogTitle>
      <DialogContent>
        <Stepper activeStep={activeStep} sx={{ mb: 3, mt: 1 }}>
          {WIZARD_STEPS.map((label) => (
            <Step key={label}>
              <StepLabel>{label}</StepLabel>
            </Step>
          ))}
        </Stepper>

        {activeStep === 0 && (
          <Stack spacing={2}>
            {!isSupportedHostCount(hosts.length) && (
              <Alert severity="error">
                Select exactly one host for a single-member replica set, or
                three for a three-member one. {hosts.length} selected.
              </Alert>
            )}
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Host</TableCell>
                  <TableCell>Address</TableCell>
                  <TableCell>Operating system</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {hosts.map((host) => (
                  <TableRow key={host.node_id}>
                    <TableCell>{host.name}</TableCell>
                    <TableCell>{host.address ?? '—'}</TableCell>
                    <TableCell>{host.os ?? '—'}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </Stack>
        )}

        {activeStep === 1 && (
          <Stack spacing={2} sx={{ mt: 1 }}>
            <DialogContentText>
              Percona Server for MongoDB, installed through the Nomad client
              and initialized as a {hosts.length}-member replica set.
              Proof-of-concept scope only — keyFile auth, TLS off, no project
              or cluster yet.
            </DialogContentText>
            <TextField
              label="Replica set name"
              value={replicaSetName}
              onChange={(event) => setReplicaSetName(event.target.value)}
              required
              autoFocus
              fullWidth
            />
            <TextField
              label="MongoDB version"
              value={mongodbVersion}
              onChange={(event) => setMongodbVersion(event.target.value)}
              required
              fullWidth
              helperText="Only the major version selects the install source, e.g. 7.0."
            />
          </Stack>
        )}

        {activeStep === 2 && (
          <Stack spacing={2} sx={{ mt: 1 }}>
            <Alert severity="warning">
              Bootstrap will modify the selected hosts. MongoDB packages,
              configuration files, data directories, and systemd services will
              be created according to this plan.
            </Alert>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Host</TableCell>
                  <TableCell>Operating system</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {hosts.map((host) => (
                  <TableRow key={host.node_id}>
                    <TableCell>{host.name}</TableCell>
                    <TableCell>{host.os ?? '—'}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            <Typography variant="body2">
              Replica set: <strong>{replicaSetName || '—'}</strong>
              <br />
              MongoDB version: <strong>{mongodbVersion || '—'}</strong>
            </Typography>
            {bootstrap.isError && (
              <Alert severity="error">{bootstrap.error.message}</Alert>
            )}
          </Stack>
        )}

        {activeStep === 3 &&
          (run.data ? (
            <RunProgress run={run.data} />
          ) : run.isPending ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', p: 2 }}>
              <CircularProgress size={24} />
            </Box>
          ) : run.isError ? (
            <Alert severity="error">{(run.error as Error).message}</Alert>
          ) : null)}
      </DialogContent>
      <DialogActions>
        {activeStep === 0 && (
          <>
            <Button onClick={onClose}>Cancel</Button>
            <Button
              variant="contained"
              disabled={!isSupportedHostCount(hosts.length)}
              onClick={() => setActiveStep(1)}
            >
              Configure
            </Button>
          </>
        )}
        {activeStep === 1 && (
          <>
            <Button onClick={() => setActiveStep(0)}>Back</Button>
            <Button
              variant="contained"
              disabled={!replicaSetName.trim() || !mongodbVersion.trim()}
              onClick={() => setActiveStep(2)}
            >
              Review
            </Button>
          </>
        )}
        {activeStep === 2 && (
          <>
            <Button onClick={() => setActiveStep(1)}>Back</Button>
            <Button
              variant="contained"
              disabled={bootstrap.isPending}
              onClick={handleTriggerBootstrap}
            >
              Bootstrap
            </Button>
          </>
        )}
        {activeStep === 3 && (
          <Button
            variant="contained"
            onClick={() => {
              onBootstrapped();
            }}
          >
            Done
          </Button>
        )}
      </DialogActions>
    </Dialog>
  );
};
