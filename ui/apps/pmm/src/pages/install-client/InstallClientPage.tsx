import { type ChangeEvent, useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  FormControl,
  FormControlLabel,
  FormHelperText,
  InputLabel,
  MenuItem,
  Select,
  Stack,
  Switch,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import AccessTimeOutlinedIcon from '@mui/icons-material/AccessTimeOutlined';
import { Page } from 'components/page';
import { createNodeInstallToken } from 'api/installToken';
import type { SelectChangeEvent } from '@mui/material/Select';
import {
  buildInstallCommand,
  buildPmmServerURL,
  CredentialsMode,
  formatExpiresIn,
  Technology,
} from './InstallClientPage.utils';

export const InstallClientPage = () => {
  const [technology, setTechnology] = useState<Technology>('mysql');
  const [credentialsMode, setCredentialsMode] = useState<CredentialsMode>('prompt');
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
  const [copied, setCopied] = useState(false);
  const [genLoading, setGenLoading] = useState(false);
  const [genError, setGenError] = useState<string | null>(null);
  const [tokenExpiresAt, setTokenExpiresAt] = useState<Date | null>(null);
  const [now, setNow] = useState(() => Date.now());
  const refreshNow = useCallback(() => setNow(Date.now()), []);

  // Tick once a second while a token is live, so the countdown chip refreshes.
  // Stops as soon as expiresAt is null or the token has expired.
  useEffect(() => {
    if (!tokenExpiresAt || isExpired) return undefined;
    const id = window.setInterval(refreshNow, 1000);
    return () => window.clearInterval(id);
  }, [tokenExpiresAt, isExpired, refreshNow]);

  const secondsLeft = tokenExpiresAt
    ? Math.max(0, Math.floor((tokenExpiresAt.getTime() - now) / 1000))
    : 0;
  const isExpired = !!tokenExpiresAt && secondsLeft <= 0;

  // When the timer hits zero, drop the secret so the rendered command falls
  // back to the placeholder. We deliberately keep `tokenExpiresAt` set so the
  // chip can still show "Expired — regenerate" until the user acts.
  useEffect(() => {
    if (isExpired && token) {
      setToken('');
    }
  }, [isExpired, token]);

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
      }),
    [
    credentialsMode,
    dbAuthDB,
    dbHost,
    dbName,
    dbPassword,
    dbPort,
    dbServiceName,
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
    await navigator.clipboard.writeText(command);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 2000);
  }, [clipboardAvailable, command]);

  const handleGenerateToken = useCallback(async () => {
    setGenError(null);
    setGenLoading(true);
    try {
      const res = await createNodeInstallToken(technology, 0);
      setToken(res.token);
      // installToken.ts always returns expiresAt; the fallback is just defensive
      // belt-and-braces in case of a future refactor.
      const expires = res.expiresAt
        ? new Date(res.expiresAt)
        : new Date(Date.now() + 15 * 60 * 1000);
      setTokenExpiresAt(expires);
      refreshNow();
    } catch (error) {
      setGenError(error instanceof Error ? error.message : 'Failed to create token');
    } finally {
      setGenLoading(false);
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
              <Typography variant="body2" sx={{ mb: 1 }}>
                Choose installation options, then copy and run the generated command on your database
                node. <em>Include env variables</em> and <em>Pass as script flags</em> use the usual{' '}
                <code>curl … | bash</code> form. <em>Prompt on node</em> renders a two-step command
                that downloads the script to <code>/tmp/install-pmm-client.sh</code> first, then runs
                it with <code>sudo -E bash</code> so it can prompt you for the DB user and password on
                the node (or skip prompts if you already exported <code>DB_USER</code> /{' '}
                <code>DB_PASSWORD</code> — <code>-E</code> keeps them visible to the script).
              </Typography>
              <Typography variant="body2">
                <strong>Generated tokens are Grafana Admin–role on the minted install service account and valid for 15 minutes</strong>{' '}
                — treat the URL like a password.
              </Typography>
            </Alert>
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
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
              <FormControl fullWidth>
                <InputLabel id="credentials-mode-label">
                  Credentials mode
                </InputLabel>
                <Select
                  labelId="credentials-mode-label"
                  value={credentialsMode}
                  label="Credentials mode"
                  onChange={handleCredentialsModeChange}
                >
                  <MenuItem value="prompt">
                    Prompt on node (downloads script first, asks for DB user/password)
                  </MenuItem>
                  <MenuItem value="env">Include env variables (recommended for curl | bash)</MenuItem>
                  <MenuItem value="flags">Pass as script flags</MenuItem>
                </Select>
                <FormHelperText>
                  In prompt mode the rendered command is a two-liner: <code>curl -o</code> downloads
                  the script to <code>/tmp/install-pmm-client.sh</code>, then{' '}
                  <code>sudo -E bash</code> runs it on a TTY so it can ask for the DB user and password,
                  or use credentials you already exported (<code>DB_USER</code>, <code>DB_PASSWORD</code>, or
                  per-tech <code>MYSQL_*</code> / …) without prompts.
                </FormHelperText>
              </FormControl>
            </Stack>

            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
              <TextField
                fullWidth
                label="PMM host"
                value={pmmHost}
                onChange={handlePmmHostChange}
                helperText="Hostname or hostname:port for PMM_SERVER_URL (defaults to this page if empty)"
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
                    : 'Used only to render command locally in browser. Generated tokens auto-expire 15 min after creation.'
                }
              />
            </Stack>
            <Stack
              direction={{ xs: 'column', md: 'row' }}
              spacing={2}
              alignItems={{ xs: 'stretch', md: 'center' }}
            >
              <Button
                variant="outlined"
                onClick={handleGenerateToken}
                disabled={genLoading}
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
            <FormHelperText sx={{ mt: -1 }}>
              Tokens are valid for 15 minutes after generation. Run the command on your node before
              then.
            </FormHelperText>

            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
              <TextField
                fullWidth
                label="Node name (optional)"
                value={nodeName}
                onChange={handleNodeNameChange}
              />
              <TextField
                fullWidth
                label="Node address (optional)"
                value={nodeAddress}
                onChange={handleNodeAddressChange}
              />
            </Stack>

            {credentialsMode === 'prompt' ? (
              <Typography variant="body2" color="text.secondary">
                DB user and password will be requested when the script runs on the node.
              </Typography>
            ) : (
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                <TextField
                    fullWidth
                    label="DB user (optional)"
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

            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
              <TextField
                fullWidth
                label="DB host"
                value={dbHost}
                onChange={handleDbHostChange}
              />
              <TextField
                fullWidth
                label="DB port"
                value={dbPort}
                onChange={handleDbPortChange}
              />
              <TextField
                fullWidth
                label="Service name"
                value={dbServiceName}
                onChange={handleDbServiceNameChange}
              />
            </Stack>

            {technology === 'postgresql' && (
              <TextField
                fullWidth
                label="PostgreSQL database (optional)"
                value={dbName}
                onChange={handleDbNameChange}
              />
            )}
            {technology === 'mongodb' && (
              <TextField
                fullWidth
                label="MongoDB auth DB (optional)"
                value={dbAuthDB}
                onChange={handleDbAuthDBChange}
              />
            )}

            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
              <FormControlLabel
                control={
                  <Switch
                    checked={insecureTLS}
                    onChange={handleInsecureTLSChange}
                  />
                }
                label="Use insecure TLS"
              />
              <FormControlLabel
                control={
                  <Switch
                    checked={registerForce}
                    onChange={handleRegisterForceChange}
                  />
                }
                label="Force re-register node"
              />
            </Stack>
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
          </Stack>
        </CardContent>
      </Card>
    </Page>
  );
};
