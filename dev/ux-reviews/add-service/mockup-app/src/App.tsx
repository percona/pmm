import { FC, ReactNode, useContext, useEffect } from 'react';
import { FormProvider, useForm } from 'react-hook-form';
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Alert,
  AlertTitle,
  Box,
  Breadcrumbs,
  Button,
  Card,
  CardActionArea,
  CardContent,
  Chip,
  CircularProgress,
  Divider,
  IconButton,
  InputAdornment,
  Link,
  MenuItem,
  Paper,
  Radio,
  Stack,
  Step,
  StepLabel,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TextField,
  Typography,
} from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import SearchIcon from '@mui/icons-material/Search';
import CloudOutlinedIcon from '@mui/icons-material/CloudOutlined';
import CloudQueueIcon from '@mui/icons-material/CloudQueue';
import HubOutlinedIcon from '@mui/icons-material/HubOutlined';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import AddIcon from '@mui/icons-material/Add';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import OpenInNewIcon from '@mui/icons-material/OpenInNew';
import TerminalIcon from '@mui/icons-material/Terminal';
import LockOutlinedIcon from '@mui/icons-material/LockOutlined';
import UploadFileOutlinedIcon from '@mui/icons-material/UploadFileOutlined';
import HourglassEmptyIcon from '@mui/icons-material/HourglassEmpty';
import ConstructionOutlinedIcon from '@mui/icons-material/ConstructionOutlined';
import { visuallyHidden } from '@mui/utils';
import ToggleButton from '@mui/material/ToggleButton';
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup';
import Autocomplete from '@mui/material/Autocomplete';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import ArrowForwardIcon from '@mui/icons-material/ArrowForward';
import {
  CodeCopyBlock,
  ColorModeContext,
  HaproxyIcon,
  MongoIcon,
  MySqlIcon,
  PerconaVaIcon,
  PostgreSqlIcon,
  ProxyIcon,
  SelectInput,
  Stepper,
  SwitchInput,
  TextInput,
  ThemeContextProvider,
  pmmThemeOptions,
} from '@percona/peak-ui';

const params = new URLSearchParams(window.location.search);
const screen = params.get('s') || 'select';
const wantedMode = (params.get('mode') || 'dark') as 'light' | 'dark';

const ModeSync: FC = () => {
  const { colorMode, toggleColorMode } = useContext(ColorModeContext);
  useEffect(() => {
    if (colorMode !== wantedMode) toggleColorMode();
  }, [colorMode, toggleColorMode]);
  return null;
};

/* ---------- shared bits ---------- */

const Shell: FC<{ crumbs: string[]; children: ReactNode; aside?: ReactNode; footer?: ReactNode }> = ({
  crumbs,
  children,
  aside,
  footer,
}) => (
  <Box sx={{ minHeight: '100vh', bgcolor: 'background.paper', display: 'flex', flexDirection: 'column' }}>
    <Box component="main" sx={{ px: { xs: 2, md: 5 }, pt: 3, flex: 1 }}>
      <Breadcrumbs sx={{ mb: 2 }}>
        {crumbs.map((c, i) =>
          i < crumbs.length - 1 ? (
            <Link key={c} underline="always" color="text.primary" href="#">
              {c}
            </Link>
          ) : (
            <Typography key={c} color="text.primary">
              {c}
            </Typography>
          )
        )}
      </Breadcrumbs>
      <Stack direction={{ xs: 'column', lg: 'row' }} gap={4} alignItems={{ xs: 'stretch', lg: 'flex-start' }} sx={{ maxWidth: 1240 }}>
        <Box sx={{ flex: 1, minWidth: 0, maxWidth: 800 }}>{children}</Box>
        {aside && (
          <Box component="aside" aria-label="Setup help" sx={{ width: { xs: '100%', lg: 340 }, maxWidth: 800, flexShrink: 0, position: { lg: 'sticky' }, top: 24 }}>
            {aside}
          </Box>
        )}
      </Stack>
    </Box>
    {footer}
  </Box>
);

const StickyFooter: FC<{ primary: string; secondary?: ReactNode; note?: ReactNode }> = ({ primary, secondary, note }) => (
  <Paper
    square
    elevation={0}
    sx={{
      borderTop: 1,
      borderColor: 'divider',
      px: { xs: 2, md: 5 },
      py: 1.5,
      bgcolor: 'background.paper',
    }}
  >
    <Stack direction="row" gap={1.5} alignItems="center" flexWrap="wrap" sx={{ maxWidth: 800 }}>
      <Box sx={{ flex: '1 1 220px' }}>{note}</Box>
      <Button variant="text">Cancel</Button>
      {secondary}
      <Button variant="contained">{primary}</Button>
    </Stack>
  </Paper>
);

const Section: FC<{ title: string; description?: ReactNode; children: ReactNode; action?: ReactNode }> = ({
  title,
  description,
  children,
  action,
}) => (
  <Card variant="outlined" sx={{ mb: 3 }}>
    <CardContent sx={{ p: 3 }}>
      <Stack direction="row" alignItems="flex-start" sx={{ mb: 2 }}>
        <Box sx={{ flex: 1 }}>
          <Typography variant="h5" component="h2">{title}</Typography>
          {description && (
            <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
              {description}
            </Typography>
          )}
        </Box>
        {action}
      </Stack>
      <Stack gap={2.5}>{children}</Stack>
    </CardContent>
  </Card>
);

const Row: FC<{ children: ReactNode; cols?: string }> = ({ children, cols = '1fr 1fr' }) => (
  <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: cols }, gap: 2, alignItems: 'start' }}>{children}</Box>
);

const PageHeader: FC<{ icon: ReactNode; title: string; subtitle: string }> = ({ icon, title, subtitle }) => (
  <Stack direction="row" gap={2} alignItems="center" sx={{ mb: 3 }}>
    <IconButton size="small" aria-label="Back to service types">
      <ArrowBackIcon />
    </IconButton>
    <Box sx={{ fontSize: 40, display: 'flex' }} aria-hidden>{icon}</Box>
    <Box>
      <Typography variant="h3" component="h1">{title}</Typography>
      <Typography variant="body2" color="text.secondary">
        {subtitle}
      </Typography>
    </Box>
  </Stack>
);

const ChoiceCard: FC<{ selected?: boolean; title: string; description: string; badge?: ReactNode }> = ({
  selected,
  title,
  description,
  badge,
}) => (
  <Card
    component="label"
    variant="outlined"
    sx={{
      borderColor: selected ? 'primary.main' : 'divider',
      borderWidth: selected ? 2 : 1,
      flex: 1,
      cursor: 'pointer',
      '&:focus-within': { outline: 2, outlineColor: 'primary.main', outlineOffset: 2 },
    }}
  >
    <Box sx={{ p: 2, height: '100%' }}>
      <Stack direction="row" gap={1} alignItems="flex-start">
        <Radio checked={!!selected} size="small" sx={{ p: 0, mt: 0.25 }} />
        <Box>
          <Stack direction="row" gap={1} alignItems="center">
            <Typography variant="subtitle2">{title}</Typography>
            {badge}
          </Stack>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
            {description}
          </Typography>
        </Box>
      </Stack>
    </Box>
  </Card>
);

/* Marks an element that the current API can't back yet. */
const ApiNote: FC<{ children: ReactNode }> = ({ children }) => (
  <Chip
    size="small"
    variant="outlined"
    color="warning"
    icon={<ConstructionOutlinedIcon />}
    label={children}
    sx={{ borderStyle: 'dashed', alignSelf: 'flex-start', height: 'auto', py: 0.25, '& .MuiChip-label': { whiteSpace: 'normal' } }}
  />
);

/* ---------- Screen A: service type picker ---------- */

type Tech = { key: string; name: string; desc: string; icon: ReactNode; caps: string[]; disabled?: string };

const groups: { title: string; items: Tech[] }[] = [
  {
    title: 'Databases',
    items: [
      { key: 'mysql', name: 'MySQL', desc: 'Percona Server, MySQL, MariaDB', icon: <MySqlIcon fontSize="inherit" />, caps: ['Metrics', 'QAN'] },
      { key: 'postgresql', name: 'PostgreSQL', desc: 'Community, Percona Distribution', icon: <PostgreSqlIcon fontSize="inherit" />, caps: ['Metrics', 'QAN'] },
      { key: 'mongodb', name: 'MongoDB', desc: 'Replica sets and sharded clusters', icon: <MongoIcon fontSize="inherit" />, caps: ['Metrics', 'QAN'] },
      { key: 'valkey', name: 'Valkey / Redis', desc: 'Valkey and Redis servers', icon: <PerconaVaIcon fontSize="inherit" />, caps: ['Metrics'] },
    ],
  },
  {
    title: 'Proxies and load balancers',
    items: [
      { key: 'proxysql', name: 'ProxySQL', desc: 'Admin interface metrics', icon: <ProxyIcon fontSize="inherit" />, caps: ['Metrics'] },
      { key: 'haproxy', name: 'HAProxy', desc: 'Built-in Prometheus endpoint', icon: <HaproxyIcon fontSize="inherit" />, caps: ['Metrics'] },
    ],
  },
  {
    title: 'Cloud discovery',
    items: [
      { key: 'rds', name: 'Amazon RDS and Aurora', desc: 'Find MySQL and PostgreSQL instances in your AWS account', icon: <CloudOutlinedIcon fontSize="inherit" />, caps: ['Metrics', 'QAN', 'Discovery'] },
      { key: 'azure', name: 'Azure Database', desc: 'MySQL and PostgreSQL flexible servers', icon: <CloudQueueIcon fontSize="inherit" />, caps: ['Metrics', 'Discovery'], disabled: 'Turned off in Settings › Advanced' },
    ],
  },
  {
    title: 'Other',
    items: [
      { key: 'external', name: 'External exporter', desc: 'Scrape any Prometheus-compatible /metrics endpoint', icon: <HubOutlinedIcon fontSize="inherit" />, caps: ['Metrics'] },
    ],
  },
];

const TechCardBody: FC<{ t: Tech }> = ({ t }) => (
  <Stack direction="row" gap={2} alignItems="flex-start" sx={{ width: '100%' }}>
    <Box sx={{ fontSize: 36, display: 'flex', color: 'text.primary' }} aria-hidden>
      {t.icon}
    </Box>
    <Box sx={{ flex: 1 }}>
      <Typography variant="subtitle1" component="span" sx={{ fontWeight: 600, display: 'block' }}>
        {t.name}
      </Typography>
      <Typography variant="body2" color="text.secondary" component="span" sx={{ display: 'block', mb: 1 }}>
        {t.desc}
      </Typography>
      <Stack direction="row" gap={0.5} flexWrap="wrap">
        {t.caps.map((c) => (
          <Chip key={c} size="small" variant="outlined" label={c} />
        ))}
      </Stack>
    </Box>
  </Stack>
);

const TechCard: FC<{ t: Tech }> = ({ t }) =>
  t.disabled ? (
    <Card variant="outlined" sx={{ height: '100%', p: 2, borderStyle: 'dashed' }} aria-disabled>
      <TechCardBody t={t} />
      <Stack direction="row" gap={1} alignItems="center" sx={{ mt: 1.5, pl: 6.5 }}>
        <Typography variant="body2">{t.disabled}.</Typography>
        <Link href="#" variant="body2">
          Turn on
        </Link>
      </Stack>
    </Card>
  ) : (
    <Card variant="outlined" sx={{ height: '100%' }}>
      <CardActionArea sx={{ p: 2, height: '100%', alignItems: 'flex-start', display: 'flex', justifyContent: 'flex-start' }}>
        <TechCardBody t={t} />
      </CardActionArea>
    </Card>
  );

const SelectScreen = () => (
  <Shell crumbs={['Inventory', 'Add service']}>
    <Typography variant="h2" component="h1" sx={{ mb: 1 }}>
      Add a service to monitor
    </Typography>
    <Typography color="text.secondary" sx={{ mb: 3 }}>
      Pick what you want to monitor. PMM connects to it from an agent and starts collecting metrics right away.
    </Typography>

    <Card variant="outlined" sx={{ mb: 3, p: 2.5 }}>
      <Stack direction="row" gap={1.5} alignItems="flex-start">
        <TerminalIcon sx={{ mt: 0.25 }} aria-hidden />
        <Box sx={{ minWidth: 0, flex: 1 }}>
          <Typography variant="subtitle1" component="h2" sx={{ fontWeight: 600 }}>
            Can you install software on the database host?
          </Typography>
          <Typography variant="body2" sx={{ mt: 0.5, mb: 1.5 }}>
            Installing PMM Client there gives you OS metrics, slow-log QAN and lower latency. Use the options below for
            managed databases or hosts you can't install on. Create a service account token first under Users and access.
          </Typography>
          <Stack gap={1}>
            <CodeCopyBlock message="pmm-admin config --server-url=https://service_token:<glsa_token>@pmm.kubine.cc" />
            <CodeCopyBlock message="pmm-admin add mysql --username=pmm --password=<password>" />
          </Stack>
          <Link href="#" sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5, mt: 1 }}>
            PMM Client install guide <OpenInNewIcon fontSize="inherit" />
          </Link>
        </Box>
      </Stack>
    </Card>

    <TextField
      size="small"
      placeholder="Search technologies"
      sx={{ mb: 3, width: 320, maxWidth: '100%' }}
      slotProps={{ htmlInput: { 'aria-label': 'Search technologies' }, input: { startAdornment: <InputAdornment position="start"><SearchIcon /></InputAdornment> } }}
    />

    {groups.map((g) => (
      <Box key={g.title} sx={{ mb: 3 }}>
        <Typography variant="overline" component="h2" color="text.secondary">
          {g.title}
        </Typography>
        <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: 'repeat(2, 1fr)' }, gap: 2, mt: 1 }}>
          {g.items.map((t) => (
            <TechCard key={t.key} t={t} />
          ))}
        </Box>
      </Box>
    ))}
  </Shell>
);

/* ---------- Screen B/C/D: MySQL form ---------- */

const HelpAside: FC<{ tested?: boolean }> = () => (
  <Stack gap={2}>
    <Card variant="outlined">
      <CardContent>
        <Typography variant="h6" component="h2" sx={{ mb: 1 }}>
          Before you start
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 1.5 }}>
          Create a monitoring user on the MySQL server. Replace &lt;agent-host&gt; with the address PMM connects from.
        </Typography>
        <ToggleButtonGroup exclusive value="8" size="small" aria-label="MySQL version" sx={{ mb: 1.5, flexWrap: 'wrap' }}>
          <ToggleButton value="8">MySQL 8.x</ToggleButton>
          <ToggleButton value="57">5.7 / MariaDB</ToggleButton>
          <ToggleButton value="rds">RDS / Aurora</ToggleButton>
        </ToggleButtonGroup>
        <CodeCopyBlock
          message={`CREATE USER 'pmm'@'<agent-host>'
  IDENTIFIED BY '<password>'
  WITH MAX_USER_CONNECTIONS 10;
GRANT SELECT, PROCESS, REPLICATION CLIENT,
  RELOAD, BACKUP_ADMIN
  ON *.* TO 'pmm'@'<agent-host>';`}
        />
        <Link href="#" sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5, mt: 1.5 }} variant="body2">
          MySQL setup guide <OpenInNewIcon fontSize="inherit" />
        </Link>
      </CardContent>
    </Card>
    <Card variant="outlined">
      <CardContent>
        <Typography variant="h6" component="h2" sx={{ mb: 1 }}>
          What PMM will collect
        </Typography>
        <Stack gap={0.75}>
          {['Server metrics (mysqld_exporter)', 'Query Analytics from Performance Schema', 'No OS metrics: the host is remote'].map(
            (l, i) => (
              <Stack key={l} direction="row" gap={1} alignItems="center">
                {i < 2 ? <CheckCircleIcon color="success" fontSize="small" /> : <InfoOutlinedIcon sx={{ color: 'text.secondary' }} fontSize="small" />}
                <Typography variant="body2">{l}</Typography>
              </Stack>
            )
          )}
        </Stack>
      </CardContent>
    </Card>
  </Stack>
);

const LabelsSection: FC<{ filled?: boolean }> = ({ filled }) => (
  <Section
    title="Name and labels"
    description="How the service shows up in dashboards, filters and access roles."
  >
    <TextInput
      name="serviceName"
      label="Service name"
      textFieldProps={{ size: 'small', helperText: 'Leave empty to use the hostname. Must be unique.' }}
    />
    <Row cols="1fr 1fr 1fr">
      <Autocomplete
        freeSolo
        size="small"
        options={['prod', 'staging', 'dev']}
        defaultValue={filled ? 'prod' : undefined}
        renderInput={(params) => <TextField {...params} label="Environment" placeholder="Pick or type" />}
      />
      <TextInput name="cluster" label="Cluster" textFieldProps={{ size: 'small' }} />
      <TextInput name="replicationSet" label="Replication set" textFieldProps={{ size: 'small' }} />
    </Row>
    <Box>
      <Typography variant="body2" sx={{ fontWeight: 500, mb: 1 }}>
        Custom labels
      </Typography>
      <Stack gap={1}>
        {(filled
          ? [
              ['team', 'payments'],
              ['tier', 'gold'],
            ]
          : []
        ).map(([k, v]) => (
          <Row key={k} cols="1fr 1fr auto">
            <TextField size="small" label="Key" value={k} />
            <TextField size="small" label="Value" value={v} />
            <IconButton size="small" aria-label={`Remove label ${k}`}>
              <DeleteOutlineIcon fontSize="small" />
            </IconButton>
          </Row>
        ))}
      </Stack>
      <Button size="small" startIcon={<AddIcon />} sx={{ mt: 1 }}>
        Add label
      </Button>
    </Box>
  </Section>
);

const ConnectionSection: FC<{ state: 'idle' | 'ok' | 'fail' | 'testing'; tls?: boolean }> = ({ state, tls }) => (
  <Section title="Connection" description="Where PMM connects and which user it signs in as.">
    <Row cols="3fr 1fr">
      <TextInput
        name="host"
        label="Hostname or IP"
        isRequired
        textFieldProps={{
          size: 'small',
          placeholder: 'db-prod-1.internal',
        }}
      />
      <TextInput name="port" label="Port" textFieldProps={{ size: 'small' }} />
    </Row>
    <Row>
      <TextInput name="username" label="Username" isRequired textFieldProps={{ size: 'small' }} />
      <TextInput name="password" label="Password" textFieldProps={{ size: 'small', type: 'password' }} />
    </Row>
    <SelectInput
      name="agent"
      label="Connect from"
      helperText="The agent that connects to MySQL and runs the exporter. It must be able to reach the database. If MySQL runs on that agent's own host, use localhost as the hostname and PMM attaches the service to that node."
      selectFieldProps={{ size: 'small' }}
    >
      <MenuItem value="pmm-server">pmm-server · built into PMM Server · connected</MenuItem>
      <MenuItem value="node-01">node-01 · pmm-agent 3.4.0 · connected</MenuItem>
      <MenuItem value="node-02" disabled>
        node-02 · pmm-agent 3.2.1 · disconnected
      </MenuItem>
    </SelectInput>
    <Box>
      <SwitchInput name="tls" label="Use TLS" labelCaption="Encrypt the connection between the agent and MySQL." />
      {tls && (
        <Stack gap={2} sx={{ mt: 2, pl: 2, borderLeft: 2, borderColor: 'divider' }}>
          {['CA certificate', 'Client certificate (optional)', 'Client key (optional)'].map((l) => (
            <Row key={l} cols="1fr auto">
              <TextField size="small" label={l} placeholder="Paste the PEM text or upload a file" multiline minRows={2} slotProps={{ inputLabel: { shrink: true } }} />
              <Button variant="outlined" startIcon={<UploadFileOutlinedIcon />} aria-label={`Upload ${l}`}>
                Upload
              </Button>
            </Row>
          ))}
          <SwitchInput name="tlsSkipVerify" label="Skip certificate and hostname verification" labelCaption="Only for self-signed test setups." />
        </Stack>
      )}
    </Box>
    {state === 'idle' && (
      <Stack direction="row" gap={1.5} alignItems="center" flexWrap="wrap">
        <Button variant="outlined">Test connection</Button>
        <ApiNote>Needs a dry-run check RPC. Without it, the check runs when you press Add service.</ApiNote>
      </Stack>
    )}
    {state === 'testing' && (
      <Stack direction="row" gap={1} alignItems="center">
        <CircularProgress size={18} />
        <Typography variant="body2">Connecting from pmm-server…</Typography>
      </Stack>
    )}
    {state === 'ok' && (
      <>
      <Alert severity="success" action={<Button size="small" color="inherit">Test again</Button>}>
        <AlertTitle>Connected</AlertTitle>
        Percona Server 8.0.36 · performance_schema on
      </Alert>
      <ApiNote>Needs a dry-run check RPC that returns the version and QAN prerequisites. Grant checks would also need new pmm-agent code.</ApiNote>
      </>
    )}
    {state === 'fail' && (
      <Alert severity="error" action={<Button size="small" color="inherit">Retry</Button>}>
        <AlertTitle>Couldn't reach db.invalid.example:3306</AlertTitle>
        The name doesn't resolve from <b>pmm-server</b>, the agent that would run the exporter. Check the hostname, or pick
        an agent on the same network under <b>Connect from</b>.
        <Box sx={{ mt: 1 }}>
          <Link href="#" color="inherit">
            Add it anyway without checking
          </Link>
        </Box>
      </Alert>
    )}
  </Section>
);

const MysqlQanSection: FC = () => (
  <Section title="Query Analytics" description="Collect query statistics so you can find slow and expensive queries.">
    <Stack direction={{ xs: 'column', sm: 'row' }} gap={2}>
      <ChoiceCard selected title="Performance Schema" description="Low overhead. Works on managed MySQL." badge={<Chip size="small" color="primary" label="Recommended" />} />
      <ChoiceCard title="Off" description="Metrics only, no query data." />
    </Stack>
    <SwitchInput
      name="queryExamples"
      label="Collect query examples"
      labelCaption="Stores real queries, which can contain personal data."
    />
    <SwitchInput name="comments" label="Parse query comments" labelCaption="Group queries by tags in SQL comments, for example from an ORM." />
  </Section>
);

const AdvancedSection: FC<{ open?: boolean }> = ({ open }) => (
  <Accordion variant="outlined" defaultExpanded={open} disableGutters sx={{ mb: 3, borderRadius: 1, '&:before': { display: 'none' } }}>
    <AccordionSummary expandIcon={<ExpandMoreIcon />} sx={{ px: 3, py: 1 }}>
      <Box>
        <Typography variant="h5">Advanced</Typography>
        <Typography variant="body2" color="text.secondary">
          Timeout 2s · No region · Table statistics automatic · All collectors on · Connection check on
        </Typography>
      </Box>
    </AccordionSummary>
    <AccordionDetails sx={{ px: 3, pb: 3 }}>
      <Stack gap={2.5}>
        <Row>
          <TextInput name="region" label="Region" textFieldProps={{ size: 'small', helperText: 'Saved on the remote node PMM creates.' }} />
          <TextInput name="az" label="Availability zone" textFieldProps={{ size: 'small' }} />
        </Row>
        <Row>
          <TextInput
            name="timeout"
            label="Connection timeout"
            textFieldProps={{ size: 'small', placeholder: '2', slotProps: { input: { endAdornment: <InputAdornment position="end">s</InputAdornment> } } }}
          />
          <TextInput
            name="maxQueryLength"
            label="Max query length"
            textFieldProps={{ size: 'small', placeholder: 'Default 2048', slotProps: { input: { endAdornment: <InputAdornment position="end">chars</InputAdornment> } } }}
          />
        </Row>
        <SelectInput
          name="tablestats"
          label="Table statistics"
          helperText="Per-table metrics get expensive on servers with many tables."
          selectFieldProps={{ size: 'small' }}
        >
          <MenuItem value="auto">Automatic: stop above 1,000 tables</MenuItem>
          <MenuItem value="custom">Stop above a custom number of tables</MenuItem>
          <MenuItem value="never">Never collect</MenuItem>
        </SelectInput>
        <Autocomplete
          multiple
          freeSolo
          size="small"
          options={['info_schema.tables', 'perf_schema.file_events', 'binlog_size', 'perf_schema.eventsstatements']}
          defaultValue={['info_schema.tables', 'perf_schema.file_events']}
          renderInput={(params) => <TextField {...params} label="Disabled collectors" placeholder="Add a collector" />}
        />
        <SwitchInput
          name="skipCheck"
          label="Skip the connection check"
          labelCaption="Add the service even if PMM can't reach it yet. PMM then can't count tables or check QAN prerequisites."
        />
        <TextInput
          name="dsn"
          label="Extra DSN parameters"
          textFieldProps={{ size: 'small', placeholder: 'allowCleartextPasswords=1' }}
        />
      </Stack>
    </AccordionDetails>
  </Accordion>
);

const MysqlScreen: FC<{ state: 'idle' | 'ok' | 'fail'; tls?: boolean; advanced?: boolean }> = ({ state, tls, advanced }) => {
  const methods = useForm({
    defaultValues: {
      host: state === 'fail' ? 'db.invalid.example' : state === 'ok' ? 'db-prod-1.internal' : '',
      port: '3306',
      username: state === 'idle' ? '' : 'pmm',
      password: state === 'idle' ? '' : 'secret',
      tls: !!tls,
      tlsSkipVerify: false,
      serviceName: '',
      environment: state === 'idle' ? '' : 'prod',
      region: '',
      az: '',
      cluster: '',
      replicationSet: '',
      queryExamples: true,
      comments: false,
      agent: 'pmm-server',
      timeout: '',
      maxQueryLength: '',
      tablestats: 'auto',
      skipCheck: false,
      dsn: '',
    },
  });
  useEffect(() => {
    if (state === 'fail') methods.setError('host', { message: 'Host not found' });
  }, [methods]);
  return (
    <FormProvider {...methods}>
      <Shell
        crumbs={['Inventory', 'Add service', 'MySQL']}
        aside={<HelpAside />}
        footer={
          <StickyFooter
            primary="Add service"
            note={
              state === 'ok' ? (
                <Stack direction="row" gap={0.75} alignItems="center">
                  <CheckCircleIcon color="success" fontSize="small" />
                  <Typography variant="body2">Connection verified</Typography>
                </Stack>
              ) : state === 'fail' ? (
                <Typography variant="body2" color="error.main" role="alert">
                  1 field needs attention: <Link href="#host">Hostname</Link>
                </Typography>
              ) : (
                <Typography variant="body2" color="text.secondary">
                  PMM tests the connection before adding.
                </Typography>
              )
            }
          />
        }
      >
        <PageHeader icon={<MySqlIcon fontSize="inherit" />} title="Add MySQL service" subtitle="Remote monitoring from an existing agent" />
        <ConnectionSection state={state} tls={tls} />
        <LabelsSection filled={state !== 'idle'} />
        <MysqlQanSection />
        <AdvancedSection open={advanced} />
      </Shell>
    </FormProvider>
  );
};

/* ---------- Screen E: PostgreSQL-specific sections ---------- */

const PostgresScreen = () => {
  const methods = useForm({
    defaultValues: {
      host: 'pg-orders.internal',
      port: '5432',
      username: 'pmm',
      password: 'secret',
      database: '',
      tls: false,
      queryExamples: true,
      comments: false,
      autoDiscovery: true,
      autoDiscoveryLimit: '10',
      limitConnections: false,
      maxConnections: '5',
    },
  });
  return (
    <FormProvider {...methods}>
      <Shell crumbs={['Inventory', 'Add service', 'PostgreSQL']} footer={<StickyFooter primary="Add service" />}>
        <PageHeader icon={<PostgreSqlIcon fontSize="inherit" />} title="Add PostgreSQL service" subtitle="Remote monitoring from an existing agent" />
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          Mockup shows the PostgreSQL-specific sections. Connect from, TLS certificates, Name and labels, and Advanced work as on the MySQL page.
        </Typography>
        <Section title="Connection" description="Where PMM connects and which user it signs in as.">
          <Row cols="3fr 1fr">
            <TextInput name="host" label="Hostname or IP" isRequired textFieldProps={{ size: 'small' }} />
            <TextInput name="port" label="Port" textFieldProps={{ size: 'small' }} />
          </Row>
          <Row cols="1fr 1fr 1fr">
            <TextInput name="username" label="Username" isRequired textFieldProps={{ size: 'small' }} />
            <TextInput name="password" label="Password" textFieldProps={{ size: 'small', type: 'password' }} />
            <TextInput name="database" label="Database" textFieldProps={{ size: 'small', placeholder: 'postgres' }} />
          </Row>
          <SwitchInput name="tls" label="Use TLS" labelCaption="Encrypt the connection between the agent and PostgreSQL." />
          <Alert severity="success" action={<Button size="small" color="inherit">Test again</Button>}>
            <AlertTitle>Connected</AlertTitle>
            PostgreSQL 17.2 · pg_stat_monitor 2.1 installed · pg_stat_statements installed · 14 databases
          </Alert>
          <ApiNote>Needs a dry-run check RPC. Today PMM only learns which extensions exist during the add, and reports a fallback as a warning afterwards.</ApiNote>
        </Section>
        <Section title="Query Analytics" description="Choose the extension PMM reads query statistics from.">
          <Stack gap={1.5}>
            <ChoiceCard
              selected
              title="pg_stat_monitor"
              description="Query examples, plans and time buckets. Detected on this server."
              badge={<Chip size="small" color="primary" label="Recommended" />}
            />
            <ChoiceCard title="pg_stat_statements" description="Standard extension, available on most managed PostgreSQL." badge={<Chip size="small" variant="outlined" label="Detected" />} />
            <ChoiceCard title="Off" description="Metrics only, no query data." />
          </Stack>
          <SwitchInput name="queryExamples" label="Collect query examples" labelCaption="Stores real queries, which can contain personal data." />
          <SwitchInput name="comments" label="Parse query comments" labelCaption="Group queries by tags in SQL comments, for example from an ORM." />
        </Section>
        <Section title="Databases" description="PMM finds the databases on this server and collects per-database metrics.">
          <Row cols="2fr 1fr">
            <SwitchInput
              name="autoDiscovery"
              label="Discover databases automatically"
              labelCaption="Stops discovering when the server has more databases than the limit."
            />
            <TextInput name="autoDiscoveryLimit" label="Database limit" textFieldProps={{ size: 'small' }} />
          </Row>
          <Divider />
          <Row cols="1fr">
            <SwitchInput
              name="limitConnections"
              label="Limit exporter connections"
              labelCaption="Cap the connections the exporter opens. Raise it for busy servers."
            />
          </Row>
        </Section>
      </Shell>
    </FormProvider>
  );
};

/* ---------- Screen F: RDS discovery ---------- */

const rdsRows = [
  ['orders-prod', 'Aurora MySQL 3.05', 'eu-west-1', 'eu-west-1a', 'monitored'],
  ['orders-prod-ro', 'Aurora MySQL 3.05', 'eu-west-1', 'eu-west-1b', ''],
  ['billing-pg', 'PostgreSQL 16.4', 'eu-west-1', 'eu-west-1c', ''],
  ['analytics', 'PostgreSQL 15.8', 'us-east-1', 'us-east-1d', ''],
  ['legacy-crm', 'MySQL 8.0.39', 'us-east-1', 'us-east-1a', ''],
];

const RdsScreen = () => (
  <Shell crumbs={['Inventory', 'Add service', 'Amazon RDS and Aurora']}>
    <PageHeader icon={<CloudOutlinedIcon fontSize="inherit" />} title="Add Amazon RDS or Aurora" subtitle="Find instances in your AWS account, then pick one to monitor" />
    <Stepper activeStep={1} sx={{ mb: 3, maxWidth: 620 }}>
      {['Connect to AWS', 'Choose instance', 'Configure'].map((l) => (
        <Step key={l}>
          <StepLabel>{l}</StepLabel>
        </Step>
      ))}
    </Stepper>
    <Card variant="outlined" sx={{ mb: 3, p: 2.5 }}>
      <Stack direction="row" gap={1.5} alignItems="center" flexWrap="wrap">
        <LockOutlinedIcon aria-hidden />
        <Box sx={{ flex: '1 1 300px', minWidth: 0 }}>
          <Typography variant="subtitle2" component="h2">
            Connected to AWS with an IAM role
          </Typography>
          <Typography variant="body2" sx={{ overflowWrap: 'anywhere' }}>
            arn:aws:iam::123456789012:role/PmmRdsMonitoring
          </Typography>
        </Box>
        <Button variant="outlined" size="small">
          Change
        </Button>
      </Stack>
    </Card>
    <Card variant="outlined">
      <CardContent sx={{ p: 3 }}>
        <Stack direction="row" alignItems="center" sx={{ mb: 2 }}>
          <Box sx={{ flex: 1 }}>
            <Typography variant="h5" component="h2">5 instances found</Typography>
            <Typography variant="body2" color="text.secondary">
              In 2 regions · 1 already monitored
            </Typography>
          </Box>
          <TextField size="small" placeholder="Filter by name or engine" slotProps={{ htmlInput: { 'aria-label': 'Filter instances' }, input: { startAdornment: <InputAdornment position="start"><SearchIcon /></InputAdornment> } }} />
        </Stack>
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>Instance</TableCell>
              <TableCell>Engine</TableCell>
              <TableCell>Region / AZ</TableCell>
              <TableCell align="right">
                <Box component="span" sx={visuallyHidden}>
                  Action
                </Box>
              </TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {rdsRows.map(([id, engine, region, az, st]) => (
              <TableRow key={id} hover>
                <TableCell>
                  <Typography variant="body2" sx={{ fontWeight: 500 }}>
                    {id}
                  </Typography>
                </TableCell>
                <TableCell>
                  <Stack direction="row" gap={1} alignItems="center">
                    <Box sx={{ fontSize: 20, display: 'flex' }}>
                      {engine.includes('Postgre') ? <PostgreSqlIcon fontSize="inherit" /> : <MySqlIcon fontSize="inherit" />}
                    </Box>
                    {engine}
                  </Stack>
                </TableCell>
                <TableCell>
                  {region} · {az}
                </TableCell>
                <TableCell align="right">
                  {st ? (
                    <Chip size="small" color="success" variant="outlined" label="Monitored" />
                  ) : (
                    <Button size="small" endIcon={<ArrowForwardIcon />}>
                      Configure
                    </Button>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  </Shell>
);

const RdsAuthScreen = () => (
  <Shell crumbs={['Inventory', 'Add service', 'Amazon RDS and Aurora']}>
    <PageHeader icon={<CloudOutlinedIcon fontSize="inherit" />} title="Add Amazon RDS or Aurora" subtitle="Find instances in your AWS account, then pick one to monitor" />
    <Stepper activeStep={0} sx={{ mb: 3, maxWidth: 620 }}>
      {['Connect to AWS', 'Choose instance', 'Configure'].map((l) => (
        <Step key={l}>
          <StepLabel>{l}</StepLabel>
        </Step>
      ))}
    </Stepper>
    <Section title="Connect to AWS" description="How PMM Server signs in to AWS to list your instances. Use one method.">
      <Stack direction={{ xs: 'column', md: 'row' }} gap={1.5}>
        <ChoiceCard title="PMM Server's identity" description="Instance profile or IRSA when PMM runs on EC2 or EKS." />
        <ChoiceCard selected title="IAM role" description="PMM assumes a role you name." />
        <ChoiceCard title="Access keys" description="A key pair for an IAM user." />
      </Stack>
      <Row cols="1fr auto">
        <TextField size="small" label="Role ARN" defaultValue="arn:aws:iam::123456789012:role/PmmRdsMonitoring" />
        <Button variant="contained">Discover instances</Button>
      </Row>
      <Link href="#" variant="body2" sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5 }}>
        Permissions the role needs <OpenInNewIcon fontSize="inherit" />
      </Link>
    </Section>
  </Shell>
);

/* ---------- Screen G: success ---------- */

const SuccessScreen = () => (
  <Shell crumbs={['Inventory', 'Add service', 'MySQL']}>
    <Card variant="outlined" sx={{ maxWidth: 640 }}>
      <CardContent sx={{ p: 4 }}>
        <CheckCircleIcon color="success" sx={{ fontSize: 48 }} />
        <Typography variant="h3" component="h1" sx={{ mt: 1 }}>
          db-prod-1 is being monitored
        </Typography>
        <Typography color="text.secondary" sx={{ mt: 1, mb: 3 }}>
          MySQL service added to the inventory with the labels environment=prod, team=payments.
        </Typography>
        <Stack gap={1.25} sx={{ mb: 3 }}>
          {[
            ['ok', 'MySQL exporter: running on pmm-server'],
            ['ok', 'Query Analytics agent (Performance Schema): running'],
            ['wait', 'Dashboards and QAN usually show data within a minute or two'],
          ].map(([s, l]) => (
            <Stack key={l} direction="row" gap={1} alignItems="center">
              {s === 'ok' ? <CheckCircleIcon color="success" fontSize="small" /> : <HourglassEmptyIcon color="action" fontSize="small" />}
              <Typography variant="body2">{l}</Typography>
            </Stack>
          ))}
        </Stack>
        <Stack direction="row" gap={1.5} flexWrap="wrap">
          <Button variant="contained" endIcon={<ArrowForwardIcon />}>
            Open MySQL dashboard
          </Button>
          <Button variant="outlined">Query Analytics</Button>
          <Button variant="text">Add another service</Button>
        </Stack>
      </CardContent>
    </Card>
  </Shell>
);

const screens: Record<string, ReactNode> = {
  select: <SelectScreen />,
  mysql: <MysqlScreen state="idle" />,
  'mysql-ok': <MysqlScreen state="ok" tls advanced />,
  'mysql-fail': <MysqlScreen state="fail" />,
  postgresql: <PostgresScreen />,
  'rds-auth': <RdsAuthScreen />,
  rds: <RdsScreen />,
  success: <SuccessScreen />,
};

const App = () => (
  <ThemeContextProvider themeOptions={pmmThemeOptions}>
    <ModeSync />
    {screens[screen]}
  </ThemeContextProvider>
);

export default App;
