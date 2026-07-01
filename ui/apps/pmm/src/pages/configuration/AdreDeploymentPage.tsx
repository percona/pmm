import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  Divider,
  Stack,
  Switch,
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
import { FC, SyntheticEvent, useEffect, useMemo, useState } from 'react';
import { useSnackbar } from 'notistack';
import { Page } from 'components/page';
import { useUser } from 'contexts/user';
import {
  useAdreDeployment,
  useApplyAdreDeployment,
  useDeleteAdreDeploymentModel,
  useDeleteAdreDeploymentSkill,
  useProvisionAdreDeployment,
  useUpdateAdreDeploymentConfig,
  useUpdateAdreDeploymentModels,
  useUpdateAdreDeploymentPmmUrl,
  useUpsertAdreDeploymentSkill,
} from 'hooks/api/useAdre';
import type {
  AdreDeployment,
  AdreDeploymentModel,
  AdreDeploymentModelInput,
  AdreDeploymentSkill,
} from 'api/adre';

const pageProps = {
  title: 'AI Assistant Deployment',
  fullWidth: true as const,
  surface: 'paper' as const,
  footer: null,
};

const monospace = { fontFamily: 'Roboto Mono, monospace' };

function errMessage(e: unknown): string {
  const resp = (e as { response?: { data?: { message?: string; error?: string } } })?.response?.data;
  return resp?.message ?? resp?.error ?? (e as Error)?.message ?? 'Request failed';
}

const AdreDeploymentPage: FC = () => {
  const { user } = useUser();
  const isAdmin = user?.isPMMAdmin ?? false;
  const { enqueueSnackbar } = useSnackbar();
  const { data, isLoading, isError } = useAdreDeployment({ enabled: isAdmin });

  const [tab, setTab] = useState(0);

  if (!isAdmin) {
    return (
      <Page {...pageProps}>
        <Card variant="outlined">
          <CardContent>
            <Alert severity="info">
              The AI Assistant deployment configuration is available to PMM administrators only.
            </Alert>
          </CardContent>
        </Card>
      </Page>
    );
  }

  if (isLoading) {
    return (
      <Page {...pageProps}>
        <Typography>Loading…</Typography>
      </Page>
    );
  }

  if (isError || !data) {
    return (
      <Page {...pageProps}>
        <Card variant="outlined">
          <CardContent>
            <Alert severity="error">Failed to load deployment configuration.</Alert>
          </CardContent>
        </Card>
      </Page>
    );
  }

  // Go marshals empty slices as JSON null; normalize so the tabs can map() safely.
  const safeData: AdreDeployment = {
    configYaml: data.configYaml ?? '',
    models: data.models ?? [],
    skills: data.skills ?? [],
    provisioning: data.provisioning ?? {
      pmmUrl: '',
      tokenConfigured: false,
      holmesApiKeyConfigured: false,
      restartRequired: false,
      renderStatus: '',
      configDir: '',
    },
  };

  const notify = {
    onError: (e: unknown) => enqueueSnackbar(errMessage(e), { variant: 'error' }),
    onOk: (m: string) => enqueueSnackbar(m, { variant: 'success' }),
  };

  return (
    <Page {...pageProps}>
      <Box sx={{ flex: 1, minHeight: 0, overflowY: 'auto', p: 2 }}>
        {safeData.provisioning.restartRequired && (
          <Alert severity="warning" sx={{ mb: 2 }}>
            Configuration changed. Click <strong>Apply</strong>, then restart the HolmesGPT container to take effect.
          </Alert>
        )}
        <Tabs value={tab} onChange={(_: SyntheticEvent, v: number) => setTab(v)} sx={{ mb: 2 }}>
          <Tab label="Models & Keys" />
          <Tab label="config.yaml" />
          <Tab label="Skills" />
          <Tab label="Provisioning & Apply" />
        </Tabs>

        {tab === 0 && <ModelsTab data={safeData} {...notify} />}
        {tab === 1 && <ConfigTab data={safeData} {...notify} />}
        {tab === 2 && <SkillsTab data={safeData} {...notify} />}
        {tab === 3 && <ProvisioningTab data={safeData} {...notify} />}
      </Box>
    </Page>
  );
};

interface TabProps {
  data: AdreDeployment;
  onError: (e: unknown) => void;
  onOk: (m: string) => void;
}

type ModelRow = AdreDeploymentModel & { apiKey: string };

const localModelHelp = (
  <Box sx={{ p: 0.5 }}>
    <Typography variant="caption" sx={{ fontWeight: 700, display: 'block', mb: 0.5 }}>
      Local / self-hosted models (via LiteLLM)
    </Typography>
    <Box
      component="pre"
      sx={{ m: 0, fontFamily: 'Roboto Mono, monospace', fontSize: '0.72rem', whiteSpace: 'pre-wrap' }}
    >
      {`OpenAI-compatible (vLLM, LM Studio, llama.cpp, LiteLLM proxy):
  LiteLLM model:  openai/<your-model>
  API base:       http://host:port/v1
  API key:        none   (placeholder — LiteLLM requires one)

Ollama (native):
  LiteLLM model:  ollama_chat/llama3
  API base:       http://host:11434
  API key:        (leave blank)

Extra params (YAML), e.g.:
  temperature: 1
  num_ctx: 8192

The endpoint must be reachable from the HolmesGPT container
(use the service name on the shared docker network).`}
    </Box>
  </Box>
);

const modelHelpTooltip = (
  <Tooltip arrow title={localModelHelp} slotProps={{ tooltip: { sx: { maxWidth: 540 } } }}>
    <Box component="span" sx={{ ml: 0.5, cursor: 'help', color: 'primary.light', fontWeight: 700 }}>
      (?)
    </Box>
  </Tooltip>
);

const ModelsTab: FC<TabProps> = ({ data, onError, onOk }) => {
  const [rows, setRows] = useState<ModelRow[]>([]);
  const save = useUpdateAdreDeploymentModels();
  const del = useDeleteAdreDeploymentModel();

  const savedNames = useMemo(() => new Set(data.models.map((m) => m.name)), [data.models]);

  useEffect(() => {
    setRows(data.models.map((m) => ({ ...m, apiKey: '' })));
  }, [data.models]);

  const update = (i: number, patch: Partial<ModelRow>) =>
    setRows((r) => r.map((row, idx) => (idx === i ? { ...row, ...patch } : row)));

  const addRow = () =>
    setRows((r) => [...r, { name: '', litellmModel: '', apiBase: '', keyConfigured: false, apiKey: '', extraParams: '' }]);

  const onDelete = async (i: number) => {
    const row = rows[i];
    if (row.name && savedNames.has(row.name)) {
      try {
        await del.mutateAsync(row.name);
        onOk('Model deleted');
      } catch (e) {
        onError(e);
      }
      return;
    }
    setRows((r) => r.filter((_, idx) => idx !== i));
  };

  const onSave = async () => {
    const payload: AdreDeploymentModelInput[] = rows
      .filter((r) => r.name.trim() && r.litellmModel.trim())
      .map((r) => ({
        name: r.name.trim(),
        litellmModel: r.litellmModel.trim(),
        apiBase: r.apiBase,
        apiKey: r.apiKey, // empty = keep existing
        extraParams: r.extraParams,
      }));
    try {
      await save.mutateAsync(payload);
      onOk('Models saved');
    } catch (e) {
      onError(e);
    }
  };

  return (
    <Card variant="outlined">
      <CardContent>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          Models render into <code>model_list.yaml</code>. Provider keys are stored here (not as env vars) and are write-only — leave a key blank to keep the existing one. The <strong>default</strong> chat / fast model is set in the <strong>config.yaml</strong> tab (<code>model:</code> / <code>fast_model:</code>), which is what HolmesGPT honors. For a <strong>local / self-hosted</strong> model, set <strong>API base</strong> to its endpoint URL and use a provider prefix in <strong>LiteLLM model</strong> — see the {modelHelpTooltip} next to “LiteLLM model”.
        </Typography>
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>Name</TableCell>
              <TableCell>LiteLLM model{modelHelpTooltip}</TableCell>
              <TableCell>API base (endpoint)</TableCell>
              <TableCell>API key</TableCell>
              <TableCell>Extra params (YAML)</TableCell>
              <TableCell align="right" />
            </TableRow>
          </TableHead>
          <TableBody>
            {rows.map((r, i) => (
              <TableRow key={i}>
                <TableCell><TextField size="small" value={r.name} onChange={(e) => update(i, { name: e.target.value })} placeholder="gpt-5.4" /></TableCell>
                <TableCell><TextField size="small" value={r.litellmModel} onChange={(e) => update(i, { litellmModel: e.target.value })} placeholder="openai/gpt-4.1" /></TableCell>
                <TableCell><TextField size="small" value={r.apiBase} onChange={(e) => update(i, { apiBase: e.target.value })} placeholder="http://host:port/v1" /></TableCell>
                <TableCell>
                  <TextField
                    size="small"
                    type="password"
                    value={r.apiKey}
                    onChange={(e) => update(i, { apiKey: e.target.value })}
                    placeholder={r.keyConfigured ? 'saved — leave blank to keep' : 'sk-…'}
                  />
                </TableCell>
                <TableCell>
                  <TextField
                    size="small"
                    multiline
                    value={r.extraParams}
                    onChange={(e) => update(i, { extraParams: e.target.value })}
                    placeholder={'temperature: 1\nnum_ctx: 8192'}
                    slotProps={{ input: { sx: { fontFamily: 'Roboto Mono, monospace', fontSize: '0.8rem' } } }}
                  />
                </TableCell>
                <TableCell align="right">
                  <Button size="small" color="error" onClick={() => onDelete(i)}>Delete</Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
        <Stack direction="row" spacing={2} sx={{ mt: 2 }}>
          <Button variant="outlined" onClick={addRow}>Add model</Button>
          <Button variant="contained" onClick={onSave} disabled={save.isPending}>Save models</Button>
        </Stack>
      </CardContent>
    </Card>
  );
};

const ConfigTab: FC<TabProps> = ({ data, onError, onOk }) => {
  const [yaml, setYaml] = useState(data.configYaml);
  const save = useUpdateAdreDeploymentConfig();

  useEffect(() => setYaml(data.configYaml), [data.configYaml]);

  const onSave = async () => {
    try {
      await save.mutateAsync(yaml);
      onOk('config.yaml saved');
    } catch (e) {
      onError(e);
    }
  };

  return (
    <Card variant="outlined">
      <CardContent>
        <Alert severity="warning" sx={{ mb: 2 }}>
          config.yaml defines Holmes toolsets, which run shell commands. Edit with care — admin-only and audit-logged.
        </Alert>
        <TextField
          label="config.yaml"
          multiline
          minRows={20}
          fullWidth
          value={yaml}
          onChange={(e) => setYaml(e.target.value)}
          slotProps={{ input: { sx: monospace } }}
        />
        <Stack direction="row" spacing={2} sx={{ mt: 2 }}>
          <Button variant="contained" onClick={onSave} disabled={save.isPending}>Save config.yaml</Button>
        </Stack>
      </CardContent>
    </Card>
  );
};

const SkillsTab: FC<TabProps> = ({ data, onError, onOk }) => {
  const [selected, setSelected] = useState<string>('');
  const [draftName, setDraftName] = useState('');
  const [draftDesc, setDraftDesc] = useState('');
  const [draftBody, setDraftBody] = useState('');
  const upsert = useUpsertAdreDeploymentSkill();
  const del = useDeleteAdreDeploymentSkill();

  const current = useMemo<AdreDeploymentSkill | undefined>(
    () => data.skills.find((s) => s.name === selected),
    [data.skills, selected]
  );

  useEffect(() => {
    setDraftName(current?.name ?? '');
    setDraftDesc(current?.description ?? '');
    setDraftBody(current?.body ?? '');
  }, [current]);

  const startNew = () => {
    setSelected('');
    setDraftName('');
    setDraftDesc('');
    setDraftBody('---\nname: my-skill\ndescription: …\n---\n\n# My skill\n');
  };

  const onSave = async () => {
    try {
      await upsert.mutateAsync({ name: draftName.trim(), description: draftDesc, body: draftBody });
      setSelected(draftName.trim());
      onOk('Skill saved');
    } catch (e) {
      onError(e);
    }
  };

  const onToggle = async (s: AdreDeploymentSkill) => {
    try {
      await upsert.mutateAsync({ name: s.name, description: s.description, body: s.body, enabled: !s.enabled });
      onOk(s.enabled ? 'Skill disabled' : 'Skill enabled');
    } catch (e) {
      onError(e);
    }
  };

  const onDelete = async (name: string) => {
    try {
      await del.mutateAsync(name);
      if (selected === name) startNew();
      onOk('Skill deleted');
    } catch (e) {
      onError(e);
    }
  };

  return (
    <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
      <Card variant="outlined" sx={{ flex: 1, minWidth: 280 }}>
        <CardContent>
          <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1 }}>
            <Typography variant="subtitle1">Skills ({data.skills.length})</Typography>
            <Button size="small" variant="outlined" onClick={startNew}>New skill</Button>
          </Stack>
          <Divider sx={{ mb: 1 }} />
          <Stack spacing={0.5}>
            {data.skills.map((s) => (
              <Stack key={s.name} direction="row" alignItems="center" spacing={1}>
                <Button
                  size="small"
                  variant={selected === s.name ? 'contained' : 'text'}
                  onClick={() => setSelected(s.name)}
                  sx={{ justifyContent: 'flex-start', flex: 1, textTransform: 'none' }}
                >
                  {s.name}
                </Button>
                {s.source === 'builtin' && <Chip label="builtin" size="small" />}
                <Switch size="small" checked={s.enabled} onChange={() => onToggle(s)} />
                <Button size="small" color="error" onClick={() => onDelete(s.name)}>Delete</Button>
              </Stack>
            ))}
          </Stack>
        </CardContent>
      </Card>

      <Card variant="outlined" sx={{ flex: 2 }}>
        <CardContent>
          <Stack spacing={2}>
            <TextField label="Skill name" value={draftName} onChange={(e) => setDraftName(e.target.value)} disabled={!!current} helperText="Letters, digits, '-' or '_'. Becomes skills/<name>/SKILL.md." />
            <TextField label="Description" value={draftDesc} onChange={(e) => setDraftDesc(e.target.value)} multiline minRows={2} />
            <TextField label="SKILL.md body" value={draftBody} onChange={(e) => setDraftBody(e.target.value)} multiline minRows={16} slotProps={{ input: { sx: monospace } }} />
            <Box>
              <Button variant="contained" onClick={onSave} disabled={upsert.isPending || !draftName.trim()}>Save skill</Button>
            </Box>
          </Stack>
        </CardContent>
      </Card>
    </Stack>
  );
};

const ProvisioningTab: FC<TabProps> = ({ data, onError, onOk }) => {
  const apply = useApplyAdreDeployment();
  const provision = useProvisionAdreDeployment();
  const savePmmUrl = useUpdateAdreDeploymentPmmUrl();
  const p = data.provisioning;
  const [pmmUrl, setPmmUrl] = useState(p.pmmUrl);

  useEffect(() => setPmmUrl(p.pmmUrl), [p.pmmUrl]);

  const onSavePmmUrl = async () => {
    try {
      await savePmmUrl.mutateAsync(pmmUrl.trim());
      onOk('PMM URL saved');
    } catch (e) {
      onError(e);
    }
  };

  const onApply = async () => {
    try {
      const res = await apply.mutateAsync();
      onOk(res.message ?? 'Applied — restart HolmesGPT to take effect.');
    } catch (e) {
      onError(e);
    }
  };

  const onProvision = async () => {
    try {
      await provision.mutateAsync();
      onOk('Provisioned — service-account token and HOLMES_API_KEY are set.');
    } catch (e) {
      onError(e);
    }
  };

  return (
    <Card variant="outlined">
      <CardContent>
        <Stack spacing={1.5}>
          <Stack direction="row" spacing={2} alignItems="center">
            <Typography sx={{ minWidth: 240 }} color="text.secondary">PMM URL (Holmes → PMM)</Typography>
            <TextField
              size="small"
              value={pmmUrl}
              onChange={(e) => setPmmUrl(e.target.value)}
              placeholder="https://pmm-server:8443"
              helperText="Address Holmes uses to reach PMM (internal/service name on the shared network)."
              sx={{ minWidth: 360 }}
            />
            <Button variant="outlined" onClick={onSavePmmUrl} disabled={savePmmUrl.isPending}>Save URL</Button>
          </Stack>
          <Row label="Config dir (shared with Holmes)" value={p.configDir} mono />
          <Row label="PMM_API_TOKEN" chip={p.tokenConfigured ? 'minted' : 'missing'} ok={p.tokenConfigured} />
          <Row label="HOLMES_API_KEY" chip={p.holmesApiKeyConfigured ? 'generated' : 'missing'} ok={p.holmesApiKeyConfigured} />
          <Row label="Last render" value={p.lastRenderAt ? new Date(p.lastRenderAt).toLocaleString() : 'never'} />
          <Row label="Restart required" chip={p.restartRequired ? 'yes' : 'no'} ok={!p.restartRequired} />
          {p.renderStatus && <Row label="Render status" value={p.renderStatus} />}
        </Stack>
        <Divider sx={{ my: 2 }} />
        <Stack direction="row" spacing={2}>
          <Button variant="outlined" onClick={onProvision} disabled={provision.isPending}>Provision secrets</Button>
          <Button variant="contained" onClick={onApply} disabled={apply.isPending}>Apply (render to disk)</Button>
        </Stack>
        <Alert severity="info" sx={{ mt: 2 }}>
          Apply renders config.yaml, model_list.yaml, .env and skills to the shared directory. Until the Holmes reload API ships, restart the HolmesGPT container to pick up changes.
        </Alert>
      </CardContent>
    </Card>
  );
};

const Row: FC<{ label: string; value?: string; chip?: string; ok?: boolean; mono?: boolean }> = ({ label, value, chip, ok, mono }) => (
  <Stack direction="row" spacing={2} alignItems="center">
    <Typography sx={{ minWidth: 240 }} color="text.secondary">{label}</Typography>
    {chip != null ? (
      <Chip label={chip} size="small" color={ok ? 'success' : 'default'} />
    ) : (
      <Typography sx={mono ? monospace : undefined}>{value}</Typography>
    )}
  </Stack>
);

export default AdreDeploymentPage;
