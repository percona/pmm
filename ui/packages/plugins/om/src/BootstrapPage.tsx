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
import { useNavigate, useSearchParams } from 'react-router-dom';
import {
  Alert,
  Autocomplete,
  Box,
  Button,
  Checkbox,
  CircularProgress,
  Step,
  StepLabel,
  Stepper,
  Stack,
  Tab,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Tabs,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import { OM_ROUTE_AUTOMATIONS, OM_ROUTE_HOSTS } from './constants';
import { OmHeader } from './components/OmHeader';
import { toHostRows } from './inventory';
import { useOmInventoryHosts, useTriggerHostBootstrap } from './inventoryHooks';
import { useOmTopology } from './topologyHooks';
import { useOmBase } from './useOmBase';
import type { OmBootstrapMemberConfig, OmHostRow } from './types';

const DEFAULT_MONGODB_VERSION = '7.0';
// Match the fixed values every bootstrap run used before these became
// configurable (PMM-15347/plan.md §6 Phase A) -- see the SEP-side migration
// this mirrors for why.
const DEFAULT_DATA_PATH = '/var/lib/mongo';
const DEFAULT_LOG_PATH = '/var/log/mongodb/mongod.log';
const DEFAULT_PORT = '27017';
const DEFAULT_BIND_IP = '0.0.0.0';

/**
 * The wizard's own steps. Matches the ticket's mockups (PMM-15347/plan.md
 * §2.5) through Review; there is no fourth "Bootstrap" page here any more --
 * triggering a run navigates straight to Automations (plan.md §6) rather than
 * showing progress on a page the reader then has to remember to leave.
 */
const WIZARD_STEPS = ['Hosts', 'Configure', 'Review'] as const;

/** Adamo's decided phase-1 topology: a replica set of exactly one or three members. */
function isSupportedHostCount(count: number): boolean {
  return count === 1 || count === 3;
}

/**
 * The Configure step's Security tab (PMM-15347/plan.md §6 Phase A) -- every
 * control here is fixed or disabled, not a form: keyFile is the only
 * intra-cluster auth mechanism this PoC builds, and LDAP/KMIP are explicitly
 * out of scope, matching mockup `03-configure-security.png` rendering both as
 * locked, visible-but-inert controls rather than hiding them outright. TLS and
 * encryption-at-rest aren't offered here at all yet -- neither is implemented
 * (plan.md §6 Phase C), so a control for either would be a choice with no
 * effect.
 */
const SecurityTab = () => (
  <Stack spacing={2}>
    <Tooltip title="The only intra-cluster authentication mechanism this proof of concept builds.">
      <TextField
        label="Intra-cluster authentication"
        value="keyFile"
        disabled
        fullWidth
      />
    </Tooltip>
    <Tooltip title="LDAP integration is out of scope for this proof of concept.">
      <TextField
        label="LDAP"
        value="Disabled for this POC"
        disabled
        fullWidth
      />
    </Tooltip>
    <Tooltip title="KMIP/KMS integration is out of scope for this proof of concept.">
      <TextField
        label="KMIP / KMS"
        value="Disabled for this POC"
        disabled
        fullWidth
      />
    </Tooltip>
    <Typography variant="body2" color="text.secondary">
      TLS and encryption-at-rest configuration are not available yet.
    </Typography>
  </Stack>
);

/** MongoDB's own defaults for a member `rs.initiate()` names no override for. */
function defaultMemberConfig(): OmBootstrapMemberConfig {
  return { priority: 1, votes: true, hidden: false, delay_secs: 0 };
}

/** One entry per host, all at MongoDB's own defaults - the starting point every fresh selection resets to. */
function defaultMemberConfigs(
  hosts: OmHostRow[]
): Record<string, OmBootstrapMemberConfig> {
  return Object.fromEntries(
    hosts.map((host) => [host.node_id, defaultMemberConfig()])
  );
}

/**
 * True once every host's own settings would produce the same `rs.initiate()`
 * PMM already sent before per-member settings existed - used to decide
 * whether `member_configs` is worth sending at all, and whether the Review
 * step's table is worth rendering.
 */
function hasNonDefaultMemberConfig(
  memberConfigs: Record<string, OmBootstrapMemberConfig>
): boolean {
  const fallback = defaultMemberConfig();
  return Object.values(memberConfigs).some(
    (config) =>
      config.priority !== fallback.priority ||
      config.votes !== fallback.votes ||
      config.hidden !== fallback.hidden ||
      config.delay_secs !== fallback.delay_secs
  );
}

/**
 * Per-host replica-set election settings (PMM-15347/plan.md §6 Phase B),
 * matching mockup `02-configure-general.png`'s table.
 *
 * A delayed member (`delay_secs > 0`) forces priority to 0 and votes off in
 * the same action that sets the delay, rather than leaving an operator to
 * discover MongoDB's own rule (`rs.initiate()` rejects a delayed member that
 * can vote or become primary) only once TriggerHostBootstrap rejects the
 * request server-side - the two controls grey out to show why they moved.
 */
const ElectionSettingsTable = ({
  hosts,
  memberConfigs,
  onChange,
}: {
  hosts: OmHostRow[];
  memberConfigs: Record<string, OmBootstrapMemberConfig>;
  onChange: (nodeId: string, config: OmBootstrapMemberConfig) => void;
}) => (
  <Table size="small">
    <TableHead>
      <TableRow>
        <TableCell>Host</TableCell>
        <TableCell align="right">Priority</TableCell>
        <TableCell align="center">Votes</TableCell>
        <TableCell align="center">Hidden</TableCell>
        <TableCell align="right">Delay (seconds)</TableCell>
      </TableRow>
    </TableHead>
    <TableBody>
      {hosts.map((host) => {
        const config = memberConfigs[host.node_id] ?? defaultMemberConfig();
        const delayed = config.delay_secs > 0;
        return (
          <TableRow key={host.node_id}>
            <TableCell>{host.name}</TableCell>
            <TableCell align="right">
              <TextField
                type="number"
                size="small"
                value={config.priority}
                disabled={delayed}
                onChange={(event) =>
                  onChange(host.node_id, {
                    ...config,
                    priority: Number(event.target.value),
                  })
                }
                slotProps={{
                  htmlInput: { min: 0, max: 1000, style: { width: 64 } },
                }}
              />
            </TableCell>
            <TableCell align="center">
              <Checkbox
                checked={config.votes}
                disabled={delayed}
                onChange={(event) =>
                  onChange(host.node_id, {
                    ...config,
                    votes: event.target.checked,
                  })
                }
              />
            </TableCell>
            <TableCell align="center">
              <Checkbox
                checked={config.hidden}
                onChange={(event) =>
                  onChange(host.node_id, {
                    ...config,
                    hidden: event.target.checked,
                  })
                }
              />
            </TableCell>
            <TableCell align="right">
              <Tooltip title="A delayed member cannot vote or become primary - setting this also turns those off.">
                <TextField
                  type="number"
                  size="small"
                  value={config.delay_secs}
                  onChange={(event) => {
                    const delaySecs = Math.max(0, Number(event.target.value));
                    onChange(host.node_id, {
                      ...config,
                      delay_secs: delaySecs,
                      priority: delaySecs > 0 ? 0 : config.priority,
                      votes: delaySecs > 0 ? false : config.votes,
                    });
                  }}
                  slotProps={{ htmlInput: { min: 0, style: { width: 80 } } }}
                />
              </Tooltip>
            </TableCell>
          </TableRow>
        );
      })}
    </TableBody>
  </Table>
);

/**
 * Configure -> Review -> Bootstrap for a set of hosts already selected on
 * {@link HostsPage} — a page rather than a modal (PMM-15347/plan.md §6 Phase A)
 * so an in-flight run keeps a URL, survives a refresh, and reads like the rest
 * of OM's pages rather than a form floating over them.
 *
 * The host selection itself is carried across as the `?hosts=` query param
 * (comma-separated node ids) rather than router state, precisely so a refresh
 * doesn't lose it — this page's own "Hosts" step is a read-only recap of that
 * selection, not a second place to make it, so it never duplicates
 * `HostsPage`'s eligibility table.
 *
 * PMM-15347 PoC only: one or three hosts, keyFile auth, TLS off. Once
 * bootstrap is triggered this navigates to Automations with the new run's row
 * unfolded (`?expand=<run_id>`, see {@link AutomationsPage}'s own doc
 * comment) rather than showing progress here — Automations is already the
 * page a run's progress survives a refresh on, so watching it there from the
 * start avoids two places that render the same thing.
 */
export const BootstrapPage = () => {
  const navigate = useNavigate();
  const omBase = useOmBase();
  const [params] = useSearchParams();
  const hostsQuery = useOmInventoryHosts();

  const selectedIds = useMemo(
    () =>
      new Set(
        (params.get('hosts') ?? '').split(',').filter((id) => id.length > 0)
      ),
    [params]
  );
  const hosts = useMemo(
    () =>
      toHostRows(hostsQuery.data).filter((host) =>
        selectedIds.has(host.node_id)
      ),
    [hostsQuery.data, selectedIds]
  );

  const backToHosts = () => navigate(`${omBase}/${OM_ROUTE_HOSTS}`);

  const [activeStep, setActiveStep] = useState(0);
  const [configTab, setConfigTab] = useState<'general' | 'security'>('general');
  const [replicaSetName, setReplicaSetName] = useState('');
  const [mongodbVersion, setMongodbVersion] = useState(DEFAULT_MONGODB_VERSION);
  const [environment, setEnvironment] = useState('');
  const [cluster, setCluster] = useState('');
  const [dataPath, setDataPath] = useState(DEFAULT_DATA_PATH);
  const [logPath, setLogPath] = useState(DEFAULT_LOG_PATH);
  const [port, setPort] = useState(DEFAULT_PORT);
  const [bindIp, setBindIp] = useState(DEFAULT_BIND_IP);
  const [memberConfigs, setMemberConfigs] = useState<
    Record<string, OmBootstrapMemberConfig>
  >({});
  const bootstrap = useTriggerHostBootstrap();
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

  const portNumber = Number(port);
  const isConfigValid =
    replicaSetName.trim() !== '' &&
    mongodbVersion.trim() !== '' &&
    dataPath.trim() !== '' &&
    logPath.trim() !== '' &&
    bindIp.trim() !== '' &&
    Number.isInteger(portNumber) &&
    portNumber >= 1 &&
    portNumber <= 65535;

  // A fresh selection (a different ?hosts= than last render) resets the wizard
  // back to its first step - landing on this page for a different host set must
  // not carry over a previous run id or an in-flight mutation's error.
  useEffect(() => {
    setActiveStep(0);
    setConfigTab('general');
    setReplicaSetName('');
    setMongodbVersion(DEFAULT_MONGODB_VERSION);
    setEnvironment('');
    setCluster('');
    setDataPath(DEFAULT_DATA_PATH);
    setLogPath(DEFAULT_LOG_PATH);
    setPort(DEFAULT_PORT);
    setBindIp(DEFAULT_BIND_IP);
    bootstrap.reset();
    // bootstrap is a fresh object every render (useMutation), so it is deliberately
    // left out of the dependency list - including it would reset the wizard on every
    // keystroke-triggered re-render, not just on a genuinely new selection.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [params]);

  // A separate effect, keyed on the joined host id list rather than `hosts`
  // itself or `params`: `hosts` gets a fresh array reference on every
  // background refetch of the same selection, which must not wipe an
  // operator's in-progress election-settings edits, and `hosts` loads
  // asynchronously off `hostsQuery`, so resetting only on `params` could fire
  // before the real host list is known at all.
  const hostIdsKey = hosts.map((host) => host.node_id).join(',');
  useEffect(() => {
    setMemberConfigs(defaultMemberConfigs(hosts));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [hostIdsKey]);

  if (hostsQuery.isLoading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (hosts.length === 0) {
    return (
      <Stack gap={2}>
        <OmHeader title="Bootstrap" />
        <Alert severity="warning">
          No hosts selected. Select hosts to bootstrap from the Hosts page.
        </Alert>
        <Box>
          <Button variant="contained" onClick={backToHosts}>
            Back to Hosts
          </Button>
        </Box>
      </Stack>
    );
  }

  const handleTriggerBootstrap = async () => {
    const accepted = await bootstrap.mutateAsync({
      nodeIds: hosts.map((host) => host.node_id),
      replicaSetName,
      mongodbVersion,
      environment: environment.trim() || undefined,
      cluster: cluster.trim() || undefined,
      dataPath,
      logPath,
      port: Number(port),
      bindIp,
      memberConfigs,
    });
    navigate(`${omBase}/${OM_ROUTE_AUTOMATIONS}?expand=${accepted.run_id}`);
  };

  return (
    <Stack gap={2}>
      <OmHeader
        title={`Bootstrap ${hosts.length === 1 ? hosts[0].name : `${hosts.length} hosts`}`}
      />
      <Stepper activeStep={activeStep} sx={{ mb: 1 }}>
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
              Select exactly one host for a single-member replica set, or three
              for a three-member one. {hosts.length} selected.
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
        <Stack spacing={2} sx={{ maxWidth: hosts.length > 1 ? 720 : 480 }}>
          <Typography variant="body2" color="text.secondary">
            Percona Server for MongoDB, installed through the Nomad client and
            initialized as a {hosts.length}-member replica set. Proof-of-concept
            scope only — keyFile auth, TLS off.
          </Typography>
          <Tabs
            value={configTab}
            onChange={(_event, value) => setConfigTab(value)}
          >
            <Tab label="General" value="general" />
            <Tab label="Security" value="security" />
          </Tabs>
          {configTab === 'general' && (
            <Stack spacing={2} sx={{ maxWidth: 480 }}>
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
              <TextField
                label="Data path"
                value={dataPath}
                onChange={(event) => setDataPath(event.target.value)}
                required
                fullWidth
                helperText="Per host. Where mongod stores its data."
              />
              <TextField
                label="Log path"
                value={logPath}
                onChange={(event) => setLogPath(event.target.value)}
                required
                fullWidth
                helperText="Per host. Where mongod writes its log file."
              />
              <TextField
                label="Port"
                type="number"
                value={port}
                onChange={(event) => setPort(event.target.value)}
                required
                fullWidth
                slotProps={{ htmlInput: { min: 1, max: 65535 } }}
              />
              <TextField
                label="Bind IP"
                value={bindIp}
                onChange={(event) => setBindIp(event.target.value)}
                required
                fullWidth
                helperText="The interface(s) mongod listens on."
              />
            </Stack>
          )}
          {configTab === 'general' && hosts.length > 1 && (
            <Stack spacing={1} sx={{ maxWidth: 'none' }}>
              <Typography variant="subtitle2">
                Replica set election settings
              </Typography>
              <ElectionSettingsTable
                hosts={hosts}
                memberConfigs={memberConfigs}
                onChange={(nodeId, config) =>
                  setMemberConfigs((current) => ({
                    ...current,
                    [nodeId]: config,
                  }))
                }
              />
            </Stack>
          )}
          {configTab === 'security' && <SecurityTab />}
        </Stack>
      )}

      {activeStep === 2 && (
        <Stack
          spacing={2}
          sx={{
            maxWidth: hasNonDefaultMemberConfig(memberConfigs) ? 720 : 480,
          }}
        >
          <Alert severity="warning">
            Bootstrap will modify the selected hosts. MongoDB packages,
            configuration files, data directories, and systemd services will be
            created according to this plan.
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
          <Table size="small">
            <TableBody>
              {(
                [
                  ['Replica set', replicaSetName || '—'],
                  ['MongoDB version', mongodbVersion || '—'],
                  ['Environment', environment || '—'],
                  ['Cluster', cluster || '—'],
                  ['Data path', dataPath],
                  ['Log path', logPath],
                  ['Port', port],
                  ['Bind IP', bindIp],
                ] as const
              ).map(([setting, value]) => (
                <TableRow key={setting}>
                  <TableCell component="th" scope="row">
                    {setting}
                  </TableCell>
                  <TableCell>{value}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          {hasNonDefaultMemberConfig(memberConfigs) && (
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Host</TableCell>
                  <TableCell align="right">Priority</TableCell>
                  <TableCell align="center">Votes</TableCell>
                  <TableCell align="center">Hidden</TableCell>
                  <TableCell align="right">Delay (seconds)</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {hosts.map((host) => {
                  const config =
                    memberConfigs[host.node_id] ?? defaultMemberConfig();
                  return (
                    <TableRow key={host.node_id}>
                      <TableCell>{host.name}</TableCell>
                      <TableCell align="right">{config.priority}</TableCell>
                      <TableCell align="center">
                        {config.votes ? 'Yes' : 'No'}
                      </TableCell>
                      <TableCell align="center">
                        {config.hidden ? 'Yes' : 'No'}
                      </TableCell>
                      <TableCell align="right">{config.delay_secs}</TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          )}
          {bootstrap.isError && (
            <Alert severity="error">{bootstrap.error.message}</Alert>
          )}
        </Stack>
      )}

      <Stack direction="row" spacing={1} sx={{ mt: 1 }}>
        {activeStep === 0 && (
          <>
            <Button onClick={backToHosts}>Cancel</Button>
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
              disabled={!isConfigValid}
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
      </Stack>
    </Stack>
  );
};
