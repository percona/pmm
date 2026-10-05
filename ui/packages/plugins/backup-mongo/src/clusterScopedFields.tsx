/**
 * Drive the Task section's service and executor from the selected cluster.
 *
 * Both MongoDB Backups forms open by asking for a *Database Service* and an
 * *Execution Host* -- two things an operator should not have to supply. PBM's
 * configuration is cluster-wide, so naming one service and one host asks for a
 * precision the operation does not have, and nothing stops the two disagreeing: a
 * host that is not a member of the chosen service's cluster fails late, from PBM,
 * with an unclear error.
 *
 * So the cluster switcher above the tabs becomes the single input, and these two
 * fields are hidden and filled from it. The executor is a *secondary* running a live
 * pbm-agent -- see `pickExecutor` for why, and for the config-server preference on a
 * sharded cluster.
 *
 * This lives in the renderer rather than in SEP's form model on purpose. SEP would
 * need a new `ClusterRef` marker beside `ServiceRef`/`HostRef`, widening the change
 * into the shared form framework every app depends on; the fields it already has are
 * exactly what the backend needs, and filling them correctly is a presentation
 * concern. The form still submits `service_id` and `hostname`, so nothing downstream
 * changes.
 */
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';

import { Alert, Box, Chip, Typography } from '@mui/material';
import {
  ClusterHealthyIcon,
  ClusterInoperationalIcon,
  Tooltip,
} from '@percona/peak-ui';
import { useFormContext } from 'react-hook-form';

import { ClusterSwitcher } from './ClusterSwitcher';
import { pickExecutor, type PbmCluster, type PbmMember } from './pbmClusters';
import type { RenderFieldOverride } from '@sep/framework';

interface ClusterScope {
  cluster?: PbmCluster;
  executor?: PbmMember;
}

/**
 * The selected cluster, shared with the forms below.
 *
 * Context rather than props because `BackupMongoApp` must stay free of hooks: its
 * route tree is introspected by calling it as a plain function, which only works
 * while it renders without mounting. Holding the selection here keeps that true.
 */
const ClusterScopeContext = createContext<ClusterScope>({});

/** Field names this module owns, hidden from the form and set programmatically. */
export const CLUSTER_DERIVED_FIELDS = ['service_id', 'hostname'] as const;

/**
 * Write the derived service and executor into the form whenever the cluster changes.
 *
 * `shouldDirty: false` so filling a field the operator never saw does not mark the
 * form dirty and trigger an unsaved-changes prompt on navigation.
 *
 * @param cluster The selected cluster, or `undefined` while topology is loading.
 * @param executor The member chosen to run pbm, or `undefined` when none is live.
 */
export function ClusterDerivedFieldsEffect({
  cluster,
  executor,
}: {
  cluster?: PbmCluster;
  executor?: PbmMember;
}) {
  const { setValue } = useFormContext();

  useEffect(() => {
    if (!cluster || !executor) {
      return;
    }
    // service_id identifies *what* is being backed up; hostname is *where* the
    // command runs. On a sharded cluster these are deliberately the same member --
    // pbm is driven from the node it runs on.
    setValue('service_id', executor.serviceId, {
      shouldDirty: false,
      shouldValidate: false,
    });
    setValue('hostname', executor.serviceName, {
      shouldDirty: false,
      shouldValidate: false,
    });
  }, [cluster, executor, setValue]);

  return null;
}

/**
 * Show what the hidden fields were set to.
 *
 * Hiding a field an operator used to control is only acceptable if the result is
 * visible: this says which member will run the command and why it was chosen, so a
 * surprising backup target is diagnosable rather than invisible.
 *
 * @param cluster The selected cluster.
 * @param executor The member chosen to run pbm.
 */
export function ClusterScopeSummary({
  cluster,
  executor,
}: {
  cluster?: PbmCluster;
  executor?: PbmMember;
}) {
  if (!cluster) {
    return null;
  }

  if (!executor) {
    return (
      <Alert severity="error" sx={{ mb: 2 }}>
        No member of <strong>{cluster.name}</strong> has a live pbm-agent, so
        there is nothing to run the command on. Check <code>pbm status</code> on
        the cluster.
      </Alert>
    );
  }

  const secondary = executor.role !== 'P';
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 1,
        mb: 2,
        flexWrap: 'wrap',
      }}
    >
      <Tooltip
        title={
          cluster.allHealthy ? 'All agents healthy' : 'Some agents unhealthy'
        }
      >
        {cluster.allHealthy ? (
          <ClusterHealthyIcon fontSize="small" />
        ) : (
          <ClusterInoperationalIcon fontSize="small" />
        )}
      </Tooltip>
      <Typography variant="body2">
        <strong>{cluster.name}</strong>
        {cluster.sharded &&
          ` · sharded, ${cluster.replicaSets.length} replica sets`}
      </Typography>
      <Chip
        size="small"
        label={`runs on ${executor.serviceName}`}
        color={secondary ? 'default' : 'warning'}
      />
      <Typography variant="body2" color="text.secondary">
        {secondary
          ? executor.configServer
            ? 'secondary on the config replica set'
            : 'secondary'
          : 'primary — no healthy secondary available'}
      </Typography>
    </Box>
  );
}

/**
 * Whether the form should hide this field because the cluster supplies it.
 *
 * @param name The field name.
 * @returns True when the cluster switcher owns this field.
 */
export function isClusterDerived(name: string): boolean {
  return (CLUSTER_DERIVED_FIELDS as readonly string[]).includes(name);
}

/**
 * Render the cluster switcher and share its selection with the tabs below.
 *
 * @param children The tabs and routes the selection scopes.
 */
export function ClusterScopeProvider({ children }: { children: ReactNode }) {
  const [cluster, setCluster] = useState<PbmCluster | undefined>();
  const onChange = useCallback(
    (next: PbmCluster | undefined) => setCluster(next),
    []
  );
  const value = useMemo(
    () => ({ cluster, executor: cluster ? pickExecutor(cluster) : undefined }),
    [cluster]
  );

  return (
    <ClusterScopeContext.Provider value={value}>
      {/*
        Above the tabs because the cluster scopes all three of them. PBM's own
        configuration is cluster-wide rather than per-node, so "which deployment"
        is a question that precedes "configure, back up or restore".
      */}
      <ClusterSwitcher onChange={onChange} />
      {children}
    </ClusterScopeContext.Provider>
  );
}

/** Fill the hidden fields and show what they were set to. */
function ClusterScopeFields() {
  const { cluster, executor } = useContext(ClusterScopeContext);
  return (
    <>
      <ClusterDerivedFieldsEffect cluster={cluster} executor={executor} />
      <ClusterScopeSummary cluster={cluster} executor={executor} />
    </>
  );
}

/**
 * Wrap a form's own field override so the cluster supplies service and executor.
 *
 * Composed rather than replacing: the backups form still wants its task-name
 * suggestion and the restores form its backup-name picker. The derived fields are
 * removed from the layout entirely, and the effect rides along on `task_name` --
 * which every one of these forms renders -- so it mounts exactly once per form.
 *
 * Call this at module level. It closes over nothing, which is what lets
 * `BackupMongoApp` stay hook-free and keeps each override's identity stable.
 *
 * @param inner The form's existing override, if it has one.
 */
export function clusterScopedRenderField(
  inner?: RenderFieldOverride
): RenderFieldOverride {
  return (args) => {
    if (isClusterDerived(args.field.name)) {
      return null;
    }
    const rendered = inner ? inner(args) : args.renderDefault();
    if (args.field.name !== 'task_name') {
      return rendered;
    }
    return (
      <>
        <ClusterScopeFields />
        {rendered}
      </>
    );
  };
}
