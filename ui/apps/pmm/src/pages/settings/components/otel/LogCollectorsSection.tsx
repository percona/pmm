import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  MenuItem,
  Stack,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import AddIcon from '@mui/icons-material/Add';
import SettingsOutlinedIcon from '@mui/icons-material/SettingsOutlined';
import { FC, useEffect, useMemo, useState } from 'react';
import { enqueueSnackbar } from 'notistack';
import {
  agentId,
  collectorLabels,
  OtelCollectorAgent,
  OtelLogSource,
  parseLogSourcesFromLabels,
  pmmAgentId,
} from 'api/inventoryOtel';
import { type LogParserPreset } from 'api/logParserPresets';
import {
  useChangeOtelCollectorLogSources,
  useInventoryNodes,
  useOtelCollectors,
  usePmmAgents,
} from 'hooks/api/useOtelCollectors';
import { useLogParserPresets } from 'hooks/api/useLogParserPresets';
import { Messages } from '../../Settings.messages';

function apiErrorMessage(err: unknown): string {
  if (err instanceof Error) return err.message;
  return Messages.unauthorized;
}

function presetOptions(presets: LogParserPreset[]): string[] {
  const names = presets.map((p) => p.name);
  if (!names.includes('raw')) names.unshift('raw');
  return names;
}

const LogSourcesEditor: FC<{
  rows: OtelLogSource[];
  presetNames: string[];
  onChange: (rows: OtelLogSource[]) => void;
}> = ({ rows, presetNames, onChange }) => {
  const m = Messages.otel.collectors;

  const updateRow = (index: number, patch: Partial<OtelLogSource>) => {
    onChange(rows.map((r, i) => (i === index ? { ...r, ...patch } : r)));
  };

  const removeRow = (index: number) => {
    onChange(rows.filter((_, i) => i !== index));
  };

  const addRow = () => {
    onChange([...rows, { path: '', preset: 'raw' }]);
  };

  return (
    <Stack gap={1.5}>
      {rows.length === 0 && (
        <Typography variant="body2" color="text.secondary">
          {m.noSources}
        </Typography>
      )}
      {rows.map((row, index) => (
        <Stack key={`${index}-${row.path}`} direction={{ xs: 'column', sm: 'row' }} gap={1} alignItems="flex-start">
          <TextField
            label={m.pathLabel}
            value={row.path}
            onChange={(e) => updateRow(index, { path: e.target.value })}
            size="small"
            fullWidth
          />
          <TextField
            select
            label={m.presetLabel}
            value={row.preset || 'raw'}
            onChange={(e) => updateRow(index, { preset: e.target.value })}
            size="small"
            sx={{ minWidth: 180 }}
          >
            {presetNames.map((name) => (
              <MenuItem key={name} value={name}>
                {name}
              </MenuItem>
            ))}
          </TextField>
          <IconButton aria-label="remove" onClick={() => removeRow(index)} sx={{ mt: { sm: 0.5 } }}>
            <DeleteOutlineIcon />
          </IconButton>
        </Stack>
      ))}
      <Button startIcon={<AddIcon />} onClick={addRow} size="small" sx={{ alignSelf: 'flex-start' }}>
        {m.addSource}
      </Button>
    </Stack>
  );
};

const ConfigureCollectorDialog: FC<{
  collector: OtelCollectorAgent | null;
  nodeLabel: string;
  presetNames: string[];
  onClose: () => void;
}> = ({ collector, nodeLabel, presetNames, onClose }) => {
  const [rows, setRows] = useState<OtelLogSource[]>([]);
  const changeSources = useChangeOtelCollectorLogSources();
  const m = Messages.otel.collectors;

  useEffect(() => {
    if (!collector) return;
    setRows(parseLogSourcesFromLabels(collectorLabels(collector)));
  }, [collector]);

  const save = async () => {
    if (!collector) return;
    const cleaned = rows
      .map((r) => ({ path: r.path.trim(), preset: (r.preset || 'raw').trim() }))
      .filter((r) => r.path);
    try {
      await changeSources.mutateAsync({ agentId: agentId(collector), logSources: cleaned });
      enqueueSnackbar(m.saved, { variant: 'success' });
      onClose();
    } catch (err) {
      enqueueSnackbar(apiErrorMessage(err), { variant: 'error' });
    }
  };

  return (
    <Dialog open={collector !== null} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>{m.configureTitle(nodeLabel)}</DialogTitle>
      <DialogContent>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          {m.configureHelp}
        </Typography>
        <LogSourcesEditor rows={rows} presetNames={presetNames} onChange={setRows} />
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{m.cancel}</Button>
        <Button variant="contained" onClick={save} disabled={changeSources.isPending}>
          {changeSources.isPending ? m.saving : m.save}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export const LogCollectorsSection: FC = () => {
  const { data: collectors = [], isLoading, isError } = useOtelCollectors();
  const { data: pmmAgents = [] } = usePmmAgents();
  const { data: nodes = [] } = useInventoryNodes();
  const { data: presets = [] } = useLogParserPresets();
  const [active, setActive] = useState<OtelCollectorAgent | null>(null);

  const m = Messages.otel.collectors;
  const presetNames = useMemo(() => presetOptions(presets), [presets]);

  const nodeNameByPmmAgent = useMemo(() => {
    const pmmToNode = new Map<string, string>();
    for (const a of pmmAgents) {
      const id = a.agentId ?? a.agent_id ?? '';
      const nodeId = a.runsOnNodeId ?? a.runs_on_node_id ?? '';
      if (id && nodeId) pmmToNode.set(id, nodeId);
    }
    const nodeNames = new Map<string, string>();
    for (const n of nodes) {
      const id = n.nodeId ?? n.node_id ?? '';
      if (id) nodeNames.set(id, n.name ?? id);
    }
    const labelByPmm = new Map<string, string>();
    for (const [pmmId, nodeId] of pmmToNode) {
      labelByPmm.set(pmmId, nodeNames.get(nodeId) ?? nodeId);
    }
    return labelByPmm;
  }, [pmmAgents, nodes]);

  const labelFor = (c: OtelCollectorAgent) => {
    const node = nodeNameByPmmAgent.get(pmmAgentId(c));
    return node ?? pmmAgentId(c) ?? agentId(c);
  };

  return (
    <Box>
      <Typography variant="h6" sx={{ mb: 0.5 }}>
        {m.title}
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        {m.description}
      </Typography>

      {isLoading && (
        <Typography variant="body2" color="text.secondary">
          {m.loading}
        </Typography>
      )}
      {isError && (
        <Typography variant="body2" color="error">
          {m.loadError}
        </Typography>
      )}

      {!isLoading && !isError && collectors.length === 0 && (
        <Typography variant="body2" color="text.secondary">
          {m.empty}
        </Typography>
      )}

      {!isLoading && !isError && collectors.length > 0 && (
        <Stack gap={1}>
          {collectors.map((c) => {
            const sources = parseLogSourcesFromLabels(collectorLabels(c));
            const label = labelFor(c);
            return (
              <Stack
                key={agentId(c)}
                direction="row"
                alignItems="center"
                gap={1}
                sx={{ py: 1, px: 1.5, borderRadius: 1, border: 1, borderColor: 'divider' }}
              >
                <Box sx={{ flex: 1, minWidth: 0 }}>
                  <Typography variant="subtitle2">{label}</Typography>
                  <Typography variant="caption" color="text.secondary" display="block">
                    {m.agentMeta(agentId(c), c.status ?? '—', sources.length)}
                  </Typography>
                </Box>
                <Tooltip title={m.configure}>
                  <IconButton size="small" onClick={() => setActive(c)}>
                    <SettingsOutlinedIcon fontSize="small" />
                  </IconButton>
                </Tooltip>
              </Stack>
            );
          })}
        </Stack>
      )}

      <ConfigureCollectorDialog
        collector={active}
        nodeLabel={active ? labelFor(active) : ''}
        presetNames={presetNames}
        onClose={() => setActive(null)}
      />
    </Box>
  );
};
