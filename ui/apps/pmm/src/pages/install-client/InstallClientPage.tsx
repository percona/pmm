import {
  type ChangeEvent,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { isAxiosError } from 'axios';
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  Collapse,
  FormControl,
  FormControlLabel,
  FormHelperText,
  InputLabel,
  Link,
  MenuItem,
  Select,
  Stack,
  Switch,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import AccessTimeOutlinedIcon from '@mui/icons-material/AccessTimeOutlined';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { Page } from 'components/page';
import { useUser } from 'contexts/user';
import {
  createNodeInstallToken,
  DEFAULT_TTL_MINUTES,
  DEFAULT_TTL_SECONDS,
} from 'api/installToken';
import type { SelectChangeEvent } from '@mui/material/Select';
import {
  buildInstallCommand,
  buildPmmServerURL,
  CredentialsMode,
  formatExpiresIn,
  suggestDbServiceName,
  Technology,
  MYSQL_QUERY_SOURCES,
  type MySQLQuerySource,
} from './InstallClientPage.utils';

const INSTALL_DOCS_URL =
  'https://docs.percona.com/percona-monitoring-and-management/3/install-pmm/install-pmm-client/one-step-ui-install.html';

// Turn a token-generation failure into an actionable message. The Grafana
// service-account endpoints require org Admin, so a 403 is the common case; prefer
// Grafana's body message over the generic axios "Request failed with status code N".
const describeTokenError = (error: unknown): string => {
  if (isAxiosError(error)) {
    if (error.response?.status === 403) {
      return 'Generating an install command requires PMM Admin privileges.';
    }
    const apiMessage = (error.response?.data as { message?: string } | undefined)
      ?.message;
    return apiMessage || error.message;
  }
  return error instanceof Error ? error.message : 'Failed to create token';
};

export const InstallClientPage = () => {
  const [technology, setTechnology] = useState<Technology>('mysql');
  const [credentialsMode, setCredentialsMode] = useState<CredentialsMode>('prompt');
  const [automationMode, setAutomationMode] = useState(false);
  const [learnMoreOpen, setLearnMoreOpen] = useState(false);
  const [token, setToken] = useState('');
  const [pmmHost, setPmmHost] = useState(() => window.location.host);
  const [insecureTLS, setInsecureTLS] = useState(true);
  const [registerForce, setRegisterForce] = useState(false);
  const [nodeName, setNodeName] = useState('');
  const [nodeAddress, setNodeAddress] = useState('');
  const [dbUser, setDbUser] = useState('');
  const [dbPassword, setDbPassword] = useState('');
  const [dbHost, setDbHost] = useState('');
  const [dbPort, setDbPort] = useState('');
  const [dbName, setDbName] = useState('');
  const [dbAuthDB, setDbAuthDB] = useState('');
  const [dbServiceName, setDbServiceName] = useState('');
  const [mysqlQuerySource, setMysqlQuerySource] = useState<MySQLQuerySource>('');
  const [copied, setCopied] = useState(false);
  const [copyFailed, setCopyFailed] = useState(false);
  const [genLoading, setGenLoading] = useState(false);
  const [genError, setGenError] = useState<string | null>(null);
  const [tokenExpiresAt, setTokenExpiresAt] = useState<Date | null>(null);
  const [now, setNow] = useState(() => Date.now());
  const refreshNow = useCallback(() => setNow(Date.now()), []);

  const { user, isLoading: isUserLoading } = useUser();
  // Allow while the user is still loading (so admins never see a flash of the
  // "admin required" notice), but once resolved without an admin user — including an
  // errored/unauthenticated session — deny. The friendly 403 in handleGenerateToken is
  // the backstop.
  const isPmmAdmin = user ? user.isPMMAdmin : isUserLoading;

  // Mount flag + monotonic request id so an in-flight token generation that resolves
  // after unmount or after a newer generation superseded it does not write stale state.
  const isMountedRef = useRef(true);
  const generationRef = useRef(0);
  const copyTimeoutRef = useRef<number>();
  useEffect(() => {
    // Re-arm on mount: StrictMode (and any remount) runs the cleanup, which would
    // otherwise leave isMountedRef stuck false and make every generation a no-op.
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
      if (copyTimeoutRef.current) {
        window.clearTimeout(copyTimeoutRef.current);
      }
    };
  }, []);

  // ceil (not floor) so a freshly minted token shows the full "15:00" rather than
  // "14:59", and isExpired flips exactly at expiry instead of one second early.
  const secondsLeft = tokenExpiresAt
    ? Math.max(0, Math.ceil((tokenExpiresAt.getTime() - now) / 1000))
    : 0;
  const isExpired = !!tokenExpiresAt && secondsLeft <= 0;

  // In env/flags mode the generated command is non-interactive, so missing DB
  // credentials would make the install fail on the node with no chance to prompt.
  const automationCredsMissing =
    automationMode &&
    credentialsMode !== 'prompt' &&
    (technology === 'valkey'
      ? !dbPassword.trim()
      : !dbUser.trim() || !dbPassword.trim());

  useEffect(() => {
    if (!tokenExpiresAt || isExpired) return undefined;
    const id = window.setInterval(refreshNow, 1000);
    return () => window.clearInterval(id);
  }, [tokenExpiresAt, isExpired, refreshNow]);

  // On expiry, drop both the token and its expiry together so the form stays coherent
  // (empty field + "Generate", not an empty field still flagged "Expired") and the
  // rendered command reverts to the <TOKEN> placeholder, forcing a regenerate.
  useEffect(() => {
    if (isExpired) {
      setToken('');
      setTokenExpiresAt(null);
    }
  }, [isExpired]);

  // setInterval is throttled/paused in background tabs, so the countdown and isExpired
  // can lag the real expiry. Re-sync `now` whenever the tab regains focus/visibility.
  useEffect(() => {
    const resync = () => refreshNow();
    window.addEventListener('focus', resync);
    document.addEventListener('visibilitychange', resync);
    return () => {
      window.removeEventListener('focus', resync);
      document.removeEventListener('visibilitychange', resync);
    };
  }, [refreshNow]);

  const suggestedServiceName = useMemo(
    () => suggestDbServiceName(technology, dbPort, nodeName),
    [technology, dbPort, nodeName]
  );

  const serviceNameHelperText = dbServiceName.trim()
    ? 'Passed to the script as --db-service-name.'
    : `Leave empty to use the script default (${suggestedServiceName} on the node).`;

  const installerUrl = useMemo(
    () => `${window.location.origin}/pmm-static/install-pmm-client.sh`,
    []
  );

  const serverURL = useMemo(() => buildPmmServerURL(pmmHost, token), [pmmHost, token]);
  const clipboardAvailable = useMemo(
    () =>
      typeof window !== 'undefined' &&
      window.isSecureContext &&
      typeof navigator.clipboard?.writeText === 'function',
    []
  );

  const command = useMemo(
    () =>
      buildInstallCommand({
        installerUrl,
        technology,
        credentialsMode,
        serverURL,
        insecureTLS,
        registerForce,
        nodeName,
        nodeAddress,
        dbUser,
        dbPassword,
        dbHost,
        dbPort,
        dbName,
        dbAuthDB,
        dbServiceName,
        dbQuerySource: mysqlQuerySource,
      }),
    [
      credentialsMode,
      dbAuthDB,
      dbHost,
      dbName,
      dbPassword,
      dbPort,
      dbServiceName,
      mysqlQuerySource,
      dbUser,
      insecureTLS,
      installerUrl,
      nodeAddress,
      nodeName,
      registerForce,
      serverURL,
      technology,
    ]
  );

  const handleCopy = useCallback(async () => {
    if (!clipboardAvailable) return;
    try {
      await navigator.clipboard.writeText(command);
      setCopyFailed(false);
      setCopied(true);
      if (copyTimeoutRef.current) {
        window.clearTimeout(copyTimeoutRef.current);
      }
      copyTimeoutRef.current = window.setTimeout(() => setCopied(false), 2000);
    } catch {
      // writeText can reject even when the API exists (document not focused, transient
      // permission denial); surface it instead of leaving an unhandled rejection.
      setCopied(false);
      setCopyFailed(true);
    }
  }, [clipboardAvailable, command]);

  const handleGenerateToken = useCallback(async () => {
    setGenError(null);
    setGenLoading(true);
    const requestId = (generationRef.current += 1);
    // Stale = the component unmounted, or a newer generation superseded this one. We
    // deliberately do NOT key on the selected technology: the minted token is an Admin
    // token that works for any technology, and keying on it could leave genLoading stuck
    // true (and the button disabled) when the user changes the dropdown mid-flight. The
    // latest request always owns the loading flag.
    const isStale = () =>
      !isMountedRef.current || requestId !== generationRef.current;
    try {
      const res = await createNodeInstallToken(technology);
      if (isStale()) return;
      setToken(res.token);
      const expires = res.expiresAt
        ? new Date(res.expiresAt)
        : new Date(Date.now() + DEFAULT_TTL_SECONDS * 1000);
      setTokenExpiresAt(expires);
      refreshNow();
    } catch (error) {
      if (isStale()) return;
      setGenError(describeTokenError(error));
    } finally {
      if (!isStale()) {
        setGenLoading(false);
      }
    }
  }, [refreshNow, technology]);

  const handleTechnologyChange = useCallback(
    (e: SelectChangeEvent<Technology>) => setTechnology(e.target.value as Technology),
    []
  );

  const handleCredentialsModeChange = useCallback(
    (e: SelectChangeEvent<CredentialsMode>) =>
      setCredentialsMode(e.target.value as CredentialsMode),
    []
  );

  const handleAutomationModeChange = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => {
      const enabled = e.target.checked;
      setAutomationMode(enabled);
      if (enabled) {
        setCredentialsMode((mode) => (mode === 'prompt' ? 'env' : mode));
      } else {
        setCredentialsMode('prompt');
        // Prompt mode collects credentials on the node, so drop any creds typed in
        // env/flags mode — otherwise they'd silently re-enter the command if the user
        // re-enables automation later.
        setDbUser('');
        setDbPassword('');
      }
    },
    []
  );

  const handlePmmHostChange = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => setPmmHost(e.target.value),
    []
  );

  const handleTokenChange = useCallback((e: ChangeEvent<HTMLInputElement>) => {
    setToken(e.target.value);
    setTokenExpiresAt(null);
  }, []);

  const handleNodeNameChange = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => setNodeName(e.target.value),
    []
  );

  const handleNodeAddressChange = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => setNodeAddress(e.target.value),
    []
  );

  const handleDbUserChange = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => setDbUser(e.target.value),
    []
  );

  const handleDbPasswordChange = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => setDbPassword(e.target.value),
    []
  );

  const handleDbHostChange = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => setDbHost(e.target.value),
    []
  );

  const handleDbPortChange = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => setDbPort(e.target.value),
    []
  );

  const handleDbServiceNameChange = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => setDbServiceName(e.target.value),
    []
  );

  const handleMysqlQuerySourceChange = useCallback(
    (e: SelectChangeEvent<MySQLQuerySource>) =>
      setMysqlQuerySource(e.target.value as MySQLQuerySource),
    []
  );

  const handleDbNameChange = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => setDbName(e.target.value),
    []
  );

  const handleDbAuthDBChange = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => setDbAuthDB(e.target.value),
    []
  );

  const handleInsecureTLSChange = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => setInsecureTLS(e.target.checked),
    []
  );

  const handleRegisterForceChange = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => setRegisterForce(e.target.checked),
    []
  );

  return (
    <Page title="Install PMM Client (One-step)">
      <Card variant="outlined">
        <CardContent>
          <Stack spacing={2}>
            <Alert severity="info">
              <Typography variant="body2">
                Pick your database type, generate a short-lived token, then copy and run the
                command on your database server with <code>sudo</code>. The script installs the
                PMM client and adds one monitored service.
              </Typography>
              <Typography variant="body2" sx={{ mt: 1 }}>
                DB credentials are requested on the server by default — they are not embedded in
                the command.{' '}
                <Link
                  component="button"
                  variant="body2"
                  onClick={() => setLearnMoreOpen((open) => !open)}
                  sx={{ verticalAlign: 'baseline' }}
                >
                  {learnMoreOpen ? 'Show less' : 'Learn more'}
                </Link>
              </Typography>
              <Collapse in={learnMoreOpen}>
                <Stack spacing={1} sx={{ mt: 1.5 }}>
                  <Typography variant="body2">
                    Generated tokens expire in <strong>{DEFAULT_TTL_MINUTES} minutes</strong> and grant
                    Admin-level access — treat the command like a password. Run the command before it
                    expires; otherwise click <strong>Regenerate</strong> for a fresh one.
                  </Typography>
                  <Typography variant="body2">
                    <strong>Multiple instances on one node:</strong> run the command again with a
                    different port. The script skips node registration after the first run, so your
                    other monitored services stay intact.
                  </Typography>
                  <Typography variant="body2">
                    <strong>Re-adding a service later?</strong> If the command now fails with an
                    authentication error, the earlier token has expired — regenerate the command here
                    and re-run it. The script refreshes the token without removing existing services
                    (do not use <code>--force</code>, which removes the node and all its services).
                  </Typography>
                  <Typography variant="body2">
                    Enable <strong>Running in CI/automation?</strong> below to embed credentials in
                    the command (env or flags). For interactive installs, leave it off.
                  </Typography>
                  <Link href={INSTALL_DOCS_URL} target="_blank" rel="noopener noreferrer" variant="body2">
                    Full documentation
                  </Link>
                </Stack>
              </Collapse>
            </Alert>

            <FormControl fullWidth>
              <InputLabel id="technology-label">Technology</InputLabel>
              <Select
                labelId="technology-label"
                value={technology}
                label="Technology"
                onChange={handleTechnologyChange}
              >
                <MenuItem value="mysql">MySQL</MenuItem>
                <MenuItem value="postgresql">PostgreSQL</MenuItem>
                <MenuItem value="mongodb">MongoDB</MenuItem>
                <MenuItem value="valkey">Valkey</MenuItem>
              </Select>
            </FormControl>

            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
              <TextField
                fullWidth
                label="PMM host"
                value={pmmHost}
                onChange={handlePmmHostChange}
                helperText="Usually leave as-is (this page's hostname). Do not include the protocol (http/https), paths or query parameters."
              />
              <TextField
                fullWidth
                type="password"
                label="Service token"
                value={token}
                onChange={handleTokenChange}
                error={isExpired}
                helperText={
                  isExpired
                    ? 'Token expired. Click Regenerate to mint a new one.'
                    : 'Click Generate below — token is filled in automatically.'
                }
              />
            </Stack>

            {user && !isPmmAdmin && (
              <Alert severity="warning">
                Generating an install command requires PMM Admin privileges. Ask an
                administrator to generate the command for this node.
              </Alert>
            )}

            <Stack
              direction={{ xs: 'column', md: 'row' }}
              spacing={2}
              alignItems={{ xs: 'stretch', md: 'center' }}
            >
              <Button
                variant="outlined"
                onClick={handleGenerateToken}
                disabled={genLoading || !isPmmAdmin}
              >
                {genLoading
                  ? 'Generating…'
                  : tokenExpiresAt
                    ? 'Regenerate token'
                    : 'Generate short-lived token'}
              </Button>
              {tokenExpiresAt && !genLoading && (
                <Chip
                  icon={<AccessTimeOutlinedIcon />}
                  label={
                    isExpired
                      ? 'Expired — regenerate'
                      : `Expires in ${formatExpiresIn(secondsLeft)}`
                  }
                  color={isExpired ? 'error' : 'success'}
                  variant="outlined"
                  size="medium"
                />
              )}
              {genError && (
                <Alert severity="error" sx={{ flex: 1 }}>
                  {genError}
                </Alert>
              )}
            </Stack>

            {!automationMode && (
              <Typography variant="body2" color="text.secondary">
                When you run the command on the node, the script will prompt for the DB user and
                password.
              </Typography>
            )}

            <FormControlLabel
              control={
                <Switch checked={automationMode} onChange={handleAutomationModeChange} />
              }
              label="Running in CI/automation?"
            />

            {automationMode && (
              <Stack spacing={2}>
                <FormControl fullWidth>
                  <InputLabel id="credentials-mode-label">Credentials mode</InputLabel>
                  <Select
                    labelId="credentials-mode-label"
                    value={credentialsMode}
                    label="Credentials mode"
                    onChange={handleCredentialsModeChange}
                  >
                    <MenuItem value="env">
                      Include env variables (recommended for curl | bash)
                    </MenuItem>
                    <MenuItem value="flags">Pass as script flags</MenuItem>
                  </Select>
                  <FormHelperText>
                    Both modes embed credentials in the command — use only in trusted
                    automation. (To be prompted on the node instead, turn off automation.)
                  </FormHelperText>
                </FormControl>
                {credentialsMode !== 'prompt' && (
                  <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                    <TextField
                      fullWidth
                      label="DB user"
                      value={dbUser}
                      onChange={handleDbUserChange}
                    />
                    <TextField
                      fullWidth
                      type="password"
                      label="DB password"
                      value={dbPassword}
                      onChange={handleDbPasswordChange}
                    />
                  </Stack>
                )}
                {automationCredsMissing && (
                  <Alert severity="warning">
                    Enter the DB{' '}
                    {technology === 'valkey' ? 'password' : 'user and password'} above —
                    this command is non-interactive, so without credentials the install
                    will fail on the node.
                  </Alert>
                )}
              </Stack>
            )}

            <Accordion disableGutters elevation={0} sx={{ '&:before': { display: 'none' } }}>
              <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                <Typography variant="subtitle1">Advanced options</Typography>
              </AccordionSummary>
              <AccordionDetails>
                <Stack spacing={2}>
                  <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                    <TextField
                      fullWidth
                      label="Node name"
                      value={nodeName}
                      onChange={handleNodeNameChange}
                      helperText="Optional. Defaults to the server hostname."
                    />
                    <TextField
                      fullWidth
                      label="Node address"
                      value={nodeAddress}
                      onChange={handleNodeAddressChange}
                      helperText="Optional. Defaults to autodetected IP."
                    />
                  </Stack>

                  <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                    <TextField
                      fullWidth
                      label="DB host"
                      value={dbHost}
                      onChange={handleDbHostChange}
                      helperText="Optional. Defaults to 127.0.0.1 on the node."
                    />
                    <TextField
                      fullWidth
                      label="DB port"
                      value={dbPort}
                      onChange={handleDbPortChange}
                      helperText={
                        dbPort.trim()
                          ? 'Adds a -<port> suffix to the default service name.'
                          : 'Optional. Technology default if empty (e.g. 3306).'
                      }
                    />
                    <TextField
                      fullWidth
                      label="Service name"
                      value={dbServiceName}
                      onChange={handleDbServiceNameChange}
                      placeholder={suggestedServiceName}
                      helperText={serviceNameHelperText}
                    />
                  </Stack>

                  {technology === 'mysql' && (
                    <FormControl fullWidth>
                      <InputLabel id="mysql-query-source-label">MySQL query source (QAN)</InputLabel>
                      <Select
                        labelId="mysql-query-source-label"
                        value={mysqlQuerySource}
                        label="MySQL query source (QAN)"
                        onChange={handleMysqlQuerySourceChange}
                      >
                        {MYSQL_QUERY_SOURCES.map((item) => (
                          <MenuItem key={item.value || 'default'} value={item.value}>
                            {item.label}
                          </MenuItem>
                        ))}
                      </Select>
                      <FormHelperText>
                        Slow log vs Performance Schema — see MySQL connect docs for required
                        grants.
                      </FormHelperText>
                    </FormControl>
                  )}

                  {technology === 'postgresql' && (
                    <TextField
                      fullWidth
                      label="PostgreSQL database"
                      value={dbName}
                      onChange={handleDbNameChange}
                    />
                  )}
                  {technology === 'mongodb' && (
                    <TextField
                      fullWidth
                      label="MongoDB auth DB"
                      value={dbAuthDB}
                      onChange={handleDbAuthDBChange}
                    />
                  )}

                  <FormControlLabel
                    control={
                      <Switch checked={insecureTLS} onChange={handleInsecureTLSChange} />
                    }
                    label="Use insecure TLS"
                  />

                  <Box
                    sx={{
                      border: 1,
                      borderColor: 'warning.main',
                      borderRadius: 1,
                      p: 1.5,
                    }}
                  >
                    <FormControlLabel
                      sx={{ m: 0, alignItems: 'flex-start' }}
                      control={
                        <Switch
                          checked={registerForce}
                          onChange={handleRegisterForceChange}
                          color="warning"
                        />
                      }
                      label={
                        <Box>
                          <Typography variant="body2" color="text.primary">
                            Force re-register node
                          </Typography>
                          <Typography
                            variant="body2"
                            color="text.secondary"
                            sx={{ mt: 0.5 }}
                          >
                            Removes the existing node and <strong>all its services</strong> on PMM
                            Server, then registers again. Use only to recover from a failed first
                            install — not when adding another database instance.
                          </Typography>
                        </Box>
                      }
                    />
                  </Box>
                </Stack>
              </AccordionDetails>
            </Accordion>
          </Stack>
        </CardContent>
      </Card>

      <Card variant="outlined">
        <CardContent>
          <Stack spacing={2}>
            <Typography variant="h4">Generated command</Typography>
            <TextField
              value={command}
              fullWidth
              multiline
              minRows={8}
              InputProps={{ readOnly: true }}
            />
            <Box>
              <Tooltip
                title={
                  clipboardAvailable
                    ? ''
                    : 'Copy is unavailable because clipboard access requires HTTPS or localhost.'
                }
              >
                <span>
                  <Button
                    variant="contained"
                    onClick={handleCopy}
                    disabled={!clipboardAvailable}
                  >
                    Copy command
                  </Button>
                </span>
              </Tooltip>
            </Box>
            {copied && <Alert severity="success">Command copied.</Alert>}
            {copyFailed && (
              <Alert severity="error">
                Couldn&apos;t copy to the clipboard. Select the command above and copy it
                manually.
              </Alert>
            )}
          </Stack>
        </CardContent>
      </Card>
    </Page>
  );
};
