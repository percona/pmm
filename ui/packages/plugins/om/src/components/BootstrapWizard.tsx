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

import { useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Autocomplete,
  Box,
  Button,
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
import { useBootstrapRun, useTriggerHostBootstrap } from '../inventoryHooks';
import { useOmTopology } from '../topologyHooks';
import { RunProgress } from './RunProgress';
import type { OmHostRow } from '../types';

const DEFAULT_MONGODB_VERSION = '7.0';

/** The wizard's own steps, matching the ticket's mockups (PMM-15347/plan.md §2.5). */
const WIZARD_STEPS = ['Hosts', 'Configure', 'Review', 'Bootstrap'] as const;

/** Adamo's decided phase-1 topology: a replica set of exactly one or three members. */
function isSupportedHostCount(count: number): boolean {
  return count === 1 || count === 3;
}

/**
 * Configure -> Review -> Bootstrap for a set of hosts already selected on
 * {@link HostsPage} — the wizard's own "Hosts" step is a read-only recap of
 * that selection, not a second place to make it, so this dialog never
 * duplicates the eligibility table it was opened from.
 *
 * PMM-15347 PoC only: one or three hosts, keyFile auth, TLS off. Once
 * bootstrap is triggered the dialog stays open on its own "Bootstrap" step
 * polling {@link useBootstrapRun} live, rather than closing on submit the way
 * a plain form would - watching it land is the point.
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
  const [environment, setEnvironment] = useState('');
  const [cluster, setCluster] = useState('');
  const [runId, setRunId] = useState<string | null>(null);
  const bootstrap = useTriggerHostBootstrap();
  const run = useBootstrapRun(runId);
  const topology = useOmTopology();

  // Suggestions only -- typing anything not listed here creates it, the same as
  // ManagementService's own Environment/Cluster fields let an operator do today.
  // Drawn from the estate's current topology rather than a dedicated endpoint,
  // since none exists (there is no "list environments" RPC); a name only
  // appears here once some service is actually labelled with it.
  const environmentOptions = useMemo(() => {
    const names = (topology.data?.environments ?? [])
      .map((environmentDoc) => environmentDoc.env_name)
      .filter((name): name is string => Boolean(name));
    return Array.from(new Set(names)).sort();
  }, [topology.data]);

  // Scoped to the chosen environment once one is picked, matching how the
  // estate itself nests clusters under an environment -- typing a cluster name
  // that exists only in a different environment is still a new cluster here,
  // not a hidden collision.
  const clusterOptions = useMemo(() => {
    const environments = topology.data?.environments ?? [];
    const scope = environment
      ? environments.filter(
          (environmentDoc) => environmentDoc.env_name === environment
        )
      : environments;
    const names = scope.flatMap((environmentDoc) =>
      environmentDoc.clusters.map((clusterDoc) => clusterDoc.name)
    );
    return Array.from(
      new Set(names.filter((name): name is string => Boolean(name)))
    ).sort();
  }, [topology.data, environment]);

  // A fresh selection (a new open, not a re-render of the same one) resets the
  // wizard back to its first step - opening it on a different host set must not
  // carry over a previous run id or an in-flight mutation's error.
  useEffect(() => {
    setActiveStep(0);
    setReplicaSetName('');
    setMongodbVersion(DEFAULT_MONGODB_VERSION);
    setEnvironment('');
    setCluster('');
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
      environment: environment.trim() || undefined,
      cluster: cluster.trim() || undefined,
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
              Percona Server for MongoDB, installed through the Nomad client and
              initialized as a {hosts.length}-member replica set.
              Proof-of-concept scope only — keyFile auth, TLS off.
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
            <Autocomplete
              freeSolo
              options={environmentOptions}
              inputValue={environment}
              onInputChange={(_event, value) => setEnvironment(value)}
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="Environment"
                  helperText="Optional. Pick an existing environment or type a new name to create one."
                />
              )}
            />
            <Autocomplete
              freeSolo
              options={clusterOptions}
              inputValue={cluster}
              onInputChange={(_event, value) => setCluster(value)}
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="Cluster"
                  helperText="Optional. Pick an existing cluster or type a new name to create one."
                />
              )}
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
              <br />
              Environment: <strong>{environment || '—'}</strong>
              <br />
              Cluster: <strong>{cluster || '—'}</strong>
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
