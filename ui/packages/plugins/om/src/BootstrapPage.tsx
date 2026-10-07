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
import { enqueueSnackbar } from 'notistack';
import { useNavigate, useSearchParams } from 'react-router-dom';
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Alert,
  AlertTitle,
  Autocomplete,
  Box,
  Button,
  Checkbox,
  CircularProgress,
  MenuItem,
  Stack,
  Step,
  StepLabel,
  Stepper,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TextField,
  Typography,
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { OM_ROUTE_AUTOMATIONS, OM_ROUTE_NODES } from './constants';
import {
  MEMBER_CONSTRAINT_PHRASE,
  constrainMemberConfig,
  defaultMemberConfig,
  memberConstraint,
  validateElectionPlan,
} from './electionPlan';
import { OmHeader } from './components/OmHeader';
import { dataDirFreeBytes, toHostRows } from './inventory';
import { useOmInventoryHosts, useTriggerHostBootstrap } from './inventoryHooks';
import { NodeNamesLinked } from './components/NodeNamesLinked';
import { HostReadiness } from './components/HostReadiness';
import { useOmTopology } from './topologyHooks';
import { useOmBase } from './useOmBase';
import type { OmBootstrapMemberConfig, OmHostRow } from './types';
import { OmError } from './components/OmError';

const DEFAULT_MONGODB_VERSION = '7.0';
// Match the fixed values every bootstrap run used before these became
// configurable (PMM-15347/plan.md §6 Phase A) -- see the PMM Extensions-side migration
// this mirrors for why.
const DEFAULT_DATA_PATH = '/var/lib/mongo';
const DEFAULT_LOG_PATH = '/var/log/mongodb/mongod.log';
const DEFAULT_PORT = '27017';
const DEFAULT_BIND_IP = '0.0.0.0';
const GIB = 1024 ** 3;
// The install's pre_check refuses less (MIN_DATA_DISK_BYTES in PMM Extensions'
// om_bootstrap); this only warns before the run is started.
const MIN_DATA_DIR_FREE_BYTES = 5 * GIB;

/**
 * The wizard's own steps. Matches the ticket's mockups (PMM-15347/plan.md
 * §2.5) through Review; there is no fourth "Bootstrap" page here any more --
 * triggering a run navigates straight to Automations (plan.md §6) rather than
 * showing progress on a page the reader then has to remember to leave.
 */
const WIZARD_STEPS = ['Nodes', 'Configure', 'Review'] as const;

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
/**
 * The security posture of a developer-preview install, stated where it is read.
 *
 * This used to be a "Security" tab holding three permanently disabled text fields -
 * "inputs that are not inputs", which was the complaint. A tab that can never be
 * edited is worse than the same facts stated where the user is already looking, so the
 * tab is gone and this renders on the Configure step and again at Review.
 *
 * It must keep naming the mechanism and what is unavailable: the tab was the only place
 * that said keyFile, LDAP and KMIP/KMS at all, and dropping that would make the
 * rather than better.
 */
const SecurityPosture = () => (
  <Alert severity="info">
    <AlertTitle>Security in this developer preview</AlertTitle>
    Members authenticate to each other with a shared keyFile, and client
    connections are <strong>not encrypted</strong> - TLS is off. LDAP, KMIP/KMS
    and encryption at rest are not available yet, and none of them can be
    configured here.
  </Alert>
);

/**
 * How mongod's bindIp is chosen.
 *
 * `own` is the default and the safe one: each member binds to its own address.
 * That is why BootstrapMemberConfig carries a bind_ip at all - a three-member set has
 * three different addresses, so one run-level value can only be 0.0.0.0 or wrong for
 * two of the three.
 *
 * Localhost was the other suggestion and cannot work: members of a replica set have to
 * reach each other.
 */
type BindMode = 'own' | 'all' | 'custom';

const ALL_INTERFACES = '0.0.0.0';

/** BootstrapMemberConfig.priority's own ceiling in om.proto. */
const MAX_MEMBER_PRIORITY = 1000;

/** delay_secs's wire type is uint32; om.proto sets no narrower limit than that. */
const MAX_DELAY_SECS = 4294967295;

/**
 * Whether one host's election settings would actually be accepted by
 * TriggerHostBootstrap - the free-typed number inputs below only clamp what a
 * spinner arrow can reach (HTML `min`/`max` do not stop a typed value, and
 * `Number(...)` on empty or non-numeric input is `NaN`, not 0), so Review must
 * check the parsed values itself before letting a run through with one that
 * would otherwise be rejected server-side or silently coerced.
 */
function isMemberConfigValid(config: OmBootstrapMemberConfig): boolean {
  return (
    Number.isInteger(config.priority) &&
    config.priority >= 0 &&
    config.priority <= MAX_MEMBER_PRIORITY &&
    Number.isInteger(config.delay_secs) &&
    config.delay_secs >= 0 &&
    config.delay_secs <= MAX_DELAY_SECS
  );
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
 * Every edit goes through {@link constrainMemberConfig}, which applies MongoDB's
 * own member rules (a delayed member cannot vote or become primary and is kept
 * hidden; a hidden or non-voting one cannot become primary) in the same action, rather than leaving
 * an operator to discover them once `rs.initiate()` rejects the set minutes into a
 * run. The controls a rule holds grey out, and the Effect column says which rule,
 * in words, so the side effect never lives only in a hover.
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
        <TableCell>Effect</TableCell>
      </TableRow>
    </TableHead>
    <TableBody>
      {hosts.map((host) => {
        const config = memberConfigs[host.node_id] ?? defaultMemberConfig();
        const constraint = memberConstraint(config);
        const update = (next: Partial<OmBootstrapMemberConfig>) =>
          onChange(host.node_id, constrainMemberConfig({ ...config, ...next }));
        return (
          <TableRow key={host.node_id} data-testid="om-election-row">
            <TableCell>{host.name}</TableCell>
            <TableCell align="right">
              <TextField
                type="number"
                size="small"
                value={config.priority}
                disabled={constraint !== null}
                onChange={(event) =>
                  update({ priority: Number(event.target.value) })
                }
                slotProps={{
                  htmlInput: {
                    min: 0,
                    max: 1000,
                    style: { width: 64 },
                    'aria-label': `Priority for ${host.name}`,
                  },
                }}
              />
            </TableCell>
            <TableCell align="center">
              <Checkbox
                checked={config.votes}
                disabled={constraint === 'delayed'}
                onChange={(event) => update({ votes: event.target.checked })}
                slotProps={{
                  input: { 'aria-label': `Votes for ${host.name}` },
                }}
              />
            </TableCell>
            <TableCell align="center">
              <Checkbox
                checked={config.hidden}
                disabled={constraint === 'delayed'}
                onChange={(event) => update({ hidden: event.target.checked })}
                slotProps={{
                  input: { 'aria-label': `Hidden for ${host.name}` },
                }}
              />
            </TableCell>
            <TableCell align="right">
              <TextField
                type="number"
                size="small"
                value={config.delay_secs}
                onChange={(event) =>
                  update({
                    delay_secs: Math.max(0, Number(event.target.value)),
                  })
                }
                slotProps={{
                  htmlInput: {
                    min: 0,
                    style: { width: 80 },
                    'aria-label': `Delay for ${host.name}`,
                  },
                }}
              />
            </TableCell>
            <TableCell>
              {constraint && (
                <Typography variant="body2" color="text.secondary">
                  {MEMBER_CONSTRAINT_PHRASE[constraint]}
                </Typography>
              )}
            </TableCell>
          </TableRow>
        );
      })}
    </TableBody>
  </Table>
);

/** What each election setting means, for a reader who has not set one before. */
const ELECTION_SETTING_HELP = [
  [
    'Priority',
    'How likely the member is to become primary. 0 means it never will.',
  ],
  ['Votes', 'Whether the member takes part in electing the primary.'],
  ['Hidden', 'Applications cannot see or read from the member.'],
  [
    'Delay',
    'Keeps the member deliberately behind the primary by this many seconds, as a safety net.',
  ],
] as const;

/**
 * Configure -> Review -> Bootstrap for a set of hosts already selected on
 * {@link NodesPage} — a page rather than a modal (PMM-15347/plan.md §6 Phase A)
 * so an in-flight run keeps a URL, survives a refresh, and reads like the rest
 * of OM's pages rather than a form floating over them.
 *
 * The host selection itself is carried across as the `?nodes=` query param
 * (comma-separated node ids) rather than router state, precisely so a refresh
 * doesn't lose it — this page's own "Hosts" step is a read-only recap of that
 * selection, not a second place to make it, so it never duplicates
 * `NodesPage`'s eligibility table.
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
        (params.get('nodes') ?? '').split(',').filter((id) => id.length > 0)
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

  const backToHosts = () => navigate(`${omBase}/${OM_ROUTE_NODES}`);

  const blockedHosts = useMemo(
    () => hosts.filter((host) => !host.automation_eligible),
    [hosts]
  );

  // The scan measures the default data path only, so this says nothing about another.
  const lowDiskHosts = useMemo(
    () =>
      hosts.flatMap((host) => {
        const free = dataDirFreeBytes(host);
        return free !== null && free < MIN_DATA_DIR_FREE_BYTES
          ? [`${host.name} (${(free / GIB).toFixed(1)} GiB)`]
          : [];
      }),
    [hosts]
  );

  const knownNodeNames = useMemo(
    () => (hostsQuery.data ?? []).map((host) => host.name),
    [hostsQuery.data]
  );

  const [activeStep, setActiveStep] = useState(0);
  const [replicaSetName, setReplicaSetName] = useState('');
  const [mongodbVersion, setMongodbVersion] = useState(DEFAULT_MONGODB_VERSION);
  const [environment, setEnvironment] = useState('');
  const [cluster, setCluster] = useState('');
  const [dataPath, setDataPath] = useState(DEFAULT_DATA_PATH);
  const [logPath, setLogPath] = useState(DEFAULT_LOG_PATH);
  const [port, setPort] = useState(DEFAULT_PORT);
  // A mode rather than a free-typed address. The field defaulted to 0.0.0.0 with the
  // helper "The interface(s) mongod listens on" - a database reachable from every
  // network the machine is on, with nothing beside the field saying so.
  const [bindMode, setBindMode] = useState<BindMode>('own');
  const [customBindIp, setCustomBindIp] = useState(DEFAULT_BIND_IP);

  // What the run as a whole asks for. In `own` mode every member names its own
  // address, so this is never consulted -- it is set to the first node's address
  // rather than 0.0.0.0 so that a member somehow left unnamed still does not land on
  // every interface. om.proto requires it to be non-empty.
  const effectiveBindIp =
    bindMode === 'all'
      ? ALL_INTERFACES
      : bindMode === 'custom'
        ? customBindIp
        : (hosts[0]?.address ?? ALL_INTERFACES);

  // Per-member only in `own` mode: the other two modes are one value for the run, and
  // sending it per member as well would be the same fact twice.
  const memberBindIps = useMemo(
    () =>
      bindMode === 'own'
        ? new Map(
            hosts
              .filter((host) => host.address)
              .map((host) => [host.node_id, host.address as string])
          )
        : new Map<string, string>(),
    [bindMode, hosts]
  );
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

  const electionPlan = useMemo(
    () =>
      validateElectionPlan(
        hosts.map((host) => ({
          name: host.name,
          config: memberConfigs[host.node_id] ?? defaultMemberConfig(),
        }))
      ),
    [hosts, memberConfigs]
  );

  const portNumber = Number(port);
  // Everything keeping Review disabled, said beside it rather than left to a greyed-out
  // button. All of them at once: fixing one must not uncover the next.
  const reviewBlockers = [
    replicaSetName.trim() === '' && 'a replica set name',
    mongodbVersion.trim() === '' && 'a MongoDB version',
    dataPath.trim() === '' && 'a data path',
    logPath.trim() === '' && 'a log path',
    !(Number.isInteger(portNumber) && portNumber >= 1 && portNumber <= 65535) &&
      'a port from 1 to 65535',
    bindMode === 'custom' &&
      customBindIp.trim() === '' &&
      'an address to listen on',
    ...hosts
      .filter(
        (host) =>
          !isMemberConfigValid(
            memberConfigs[host.node_id] ?? defaultMemberConfig()
          )
      )
      .map((host) => `valid election settings for ${host.name}`),
    electionPlan.errors.length > 0 &&
      'election settings that can elect a primary (see above)',
  ].filter((blocker): blocker is string => typeof blocker === 'string');
  const isConfigValid = reviewBlockers.length === 0;
  const configureBlockers = [
    !isSupportedHostCount(hosts.length) &&
      `Select exactly one node for a single-member replica set, or three for a three-member one. ${hosts.length} selected.`,
    blockedHosts.length > 0 &&
      `${
        blockedHosts.length === 1
          ? '1 selected node cannot'
          : `${blockedHosts.length} selected nodes cannot`
      } be installed onto. Fix each one on the node itself, or go back and change the selection.`,
  ].filter((blocker): blocker is string => typeof blocker === 'string');

  // A fresh selection (a different ?nodes= than last render) resets the wizard
  // back to its first step - landing on this page for a different host set must
  // not carry over a previous run id or an in-flight mutation's error.
  useEffect(() => {
    setActiveStep(0);
    setReplicaSetName('');
    setMongodbVersion(DEFAULT_MONGODB_VERSION);
    setEnvironment('');
    setCluster('');
    setDataPath(DEFAULT_DATA_PATH);
    setLogPath(DEFAULT_LOG_PATH);
    setPort(DEFAULT_PORT);
    setBindMode('own');
    setCustomBindIp(DEFAULT_BIND_IP);
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

  if (hostsQuery.isError) {
    return (
      <Stack gap={2}>
        <OmHeader title="Install MongoDB" />
        <OmError
          placement="load"
          title="Could not load the nodes"
          messages={hostsQuery.error.message}
        />
        <Box>
          <Button variant="contained" onClick={backToHosts}>
            Back to Nodes
          </Button>
        </Box>
      </Stack>
    );
  }

  if (hosts.length === 0) {
    return (
      <Stack gap={2}>
        <OmHeader title="Install MongoDB" />
        <Alert severity="warning">
          No nodes selected. Pick the nodes to install on from the Nodes page.
        </Alert>
        <Box>
          <Button variant="contained" onClick={backToHosts}>
            Back to Nodes
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
      bindIp: effectiveBindIp,
      // Merged rather than replaced: a member may already carry election settings
      // from the Advanced section, and the address is one more field on the same
      // entry. Nodes appear here that named no election settings at all, which is
      // what makes `own` mode work for an untouched form.
      memberConfigs: Object.fromEntries(
        hosts.map((host) => {
          const config = memberConfigs[host.node_id] ?? defaultMemberConfig();
          const bindIp = memberBindIps.get(host.node_id);
          return [
            host.node_id,
            bindIp ? { ...config, bind_ip: bindIp } : config,
          ];
        })
      ),
    });
    enqueueSnackbar(
      `Install started on ${hosts.length === 1 ? hosts[0].name : `${hosts.length} nodes`}`,
      { variant: 'success' }
    );
    navigate(`${omBase}/${OM_ROUTE_AUTOMATIONS}?expand=${accepted.run_id}`);
  };

  return (
    <Stack gap={2}>
      <OmHeader
        title={`Install MongoDB on ${hosts.length === 1 ? hosts[0].name : `${hosts.length} nodes`}`}
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
          {/* Moved here from Review (P8, by way of PMM-15661's own out-of-scope
              list). This is the step where the user decides whether to begin, and
              a statement of what will be created belongs at the decision rather
              than after the form is filled in. */}
          <Alert severity="warning">
            This will modify the selected nodes. MongoDB packages, configuration
            files, data directories, and systemd services will be created on
            each of them.
          </Alert>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Node</TableCell>
                <TableCell>Address</TableCell>
                <TableCell>Operating system</TableCell>
                <TableCell>Ready to install</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {hosts.map((host) => (
                <TableRow key={host.node_id}>
                  <TableCell>{host.name}</TableCell>
                  <TableCell>{host.address ?? '—'}</TableCell>
                  <TableCell>{host.os ?? '—'}</TableCell>
                  <TableCell>
                    <HostReadiness host={host} omBase={omBase} />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Stack>
      )}

      {activeStep === 1 && (
        <Stack spacing={2} sx={{ maxWidth: hosts.length > 1 ? 720 : 480 }}>
          <Typography variant="body2" color="text.secondary">
            Percona Server for MongoDB, installed and initialized as a{' '}
            {hosts.length}-member replica set.
          </Typography>
          <SecurityPosture />
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
              helperText="On every node. Where mongod stores its data."
            />
            {dataPath.trim() === DEFAULT_DATA_PATH &&
              lowDiskHosts.length > 0 && (
                <Alert severity="warning" data-testid="om-low-disk-warning">
                  The install needs at least {MIN_DATA_DIR_FREE_BYTES / GIB} GiB
                  free at the data path, and the last scan found less at{' '}
                  {DEFAULT_DATA_PATH} on {lowDiskHosts.join(', ')}. Choose a
                  data path on a larger disk, or free space there first.
                </Alert>
              )}
            <TextField
              label="Log path"
              value={logPath}
              onChange={(event) => setLogPath(event.target.value)}
              required
              fullWidth
              helperText="On every node. Where mongod writes its log file."
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
              select
              label="Listen on"
              value={bindMode}
              onChange={(event) => setBindMode(event.target.value as BindMode)}
              fullWidth
              helperText="Which interfaces mongod accepts connections on."
            >
              <MenuItem value="own">
                {hosts.length === 1
                  ? "This node's own address"
                  : "Each node's own address"}
              </MenuItem>
              <MenuItem value="all">All interfaces (0.0.0.0)</MenuItem>
              <MenuItem value="custom">Custom…</MenuItem>
            </TextField>
            {bindMode === 'own' && (
              <Typography variant="caption" color="text.secondary">
                {hosts.map((host) => host.address ?? host.name).join(', ')}
              </Typography>
            )}
            {bindMode === 'custom' && (
              <TextField
                label="Bind IP"
                value={customBindIp}
                onChange={(event) => setCustomBindIp(event.target.value)}
                required
                fullWidth
                helperText="Applied to every selected node."
              />
            )}
            {/* Beside the field, not in a paragraph at the top of the step: the
                  warning is about the value that is selected right now. */}
            {effectiveBindIp === ALL_INTERFACES && (
              <Alert severity="warning">
                On {ALL_INTERFACES} mongod accepts connections from every
                network each node is attached to. With TLS off in this developer
                preview, those connections are unencrypted.
              </Alert>
            )}
          </Stack>
          {hosts.length > 1 && (
            <Accordion
              variant="outlined"
              disableGutters
              sx={{ maxWidth: 'none' }}
            >
              <AccordionSummary
                expandIcon={<ExpandMoreIcon />}
                id="om-election-settings-header"
                aria-controls="om-election-settings"
              >
                <Stack>
                  <Typography variant="subtitle2">
                    Advanced: election settings
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    Defaults: priority 1, votes on, not hidden, no delay.
                  </Typography>
                </Stack>
              </AccordionSummary>
              <AccordionDetails>
                <Stack spacing={2}>
                  <Stack component="dl" spacing={0.5} sx={{ m: 0 }}>
                    {ELECTION_SETTING_HELP.map(([setting, help]) => (
                      <Box key={setting} sx={{ display: 'flex', gap: 0.5 }}>
                        <Typography
                          component="dt"
                          variant="body2"
                          sx={{ fontWeight: 600 }}
                        >
                          {setting}:
                        </Typography>
                        <Typography
                          component="dd"
                          variant="body2"
                          color="text.secondary"
                          sx={{ m: 0 }}
                        >
                          {help}
                        </Typography>
                      </Box>
                    ))}
                  </Stack>
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
              </AccordionDetails>
            </Accordion>
          )}
          {/* Outside the accordion: a plan that cannot run has to say so even
              when the settings causing it are folded away. */}
          {electionPlan.errors.length > 0 && (
            <OmError
              placement="item"
              title="This replica set could not elect a primary"
              messages={electionPlan.errors}
              itemTestId="om-election-error"
            />
          )}
          {electionPlan.warnings.length > 0 && (
            <OmError
              placement="item"
              severity="warning"
              messages={electionPlan.warnings}
              itemTestId="om-election-warning"
            />
          )}
        </Stack>
      )}

      {activeStep === 2 && (
        <Stack
          spacing={2}
          sx={{
            maxWidth: hasNonDefaultMemberConfig(memberConfigs) ? 720 : 480,
          }}
        >
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Node</TableCell>
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
                  [
                    'Listen on',
                    bindMode === 'own'
                      ? hosts
                          .map(
                            (host) =>
                              `${host.name}: ${host.address ?? effectiveBindIp}`
                          )
                          .join(', ')
                      : effectiveBindIp,
                  ],
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
                  <TableCell>Node</TableCell>
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
          {/* Every node the refusal names becomes a link to that node and its
              newest scan, which is what makes "fix it on the node" followable.
              The message itself is the backend's -- see NodeNamesLinked for why
              the names are matched rather than parsed. */}
          {bootstrap.isError && (
            <OmError
              placement="action"
              title="Could not start the install"
              messages={
                <NodeNamesLinked
                  text={bootstrap.error.message}
                  nodeNames={knownNodeNames}
                  omBase={omBase}
                />
              }
            />
          )}
        </Stack>
      )}

      <Stack direction="row" spacing={1} alignItems="center" sx={{ mt: 1 }}>
        {activeStep === 0 && (
          <>
            <Button onClick={backToHosts}>Cancel</Button>
            <Button
              variant="contained"
              // Blocks rather than warns. Every one of these conditions describes
              // a run that cannot succeed, so letting the user through would only
              // move the failure to the final button - which is the complaint
              // itself.
              disabled={configureBlockers.length > 0}
              onClick={() => setActiveStep(1)}
            >
              Configure
            </Button>
            {configureBlockers.length > 0 && (
              <OmError
                placement="action"
                title="Cannot configure yet"
                messages={configureBlockers}
              />
            )}
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
            {!isConfigValid && (
              <OmError
                placement="action"
                severity="info"
                title="Review needs"
                messages={reviewBlockers}
                itemTestId="om-review-blocker"
              />
            )}
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
              Install MongoDB
            </Button>
          </>
        )}
      </Stack>
    </Stack>
  );
};
