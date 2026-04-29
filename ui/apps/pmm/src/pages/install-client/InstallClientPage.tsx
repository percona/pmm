import axios from 'axios';
import { useEffect, useMemo, useState } from 'react';
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
  Typography,
} from '@mui/material';
import AccessTimeOutlinedIcon from '@mui/icons-material/AccessTimeOutlined';
import { Page } from 'components/page';
import { createNodeInstallToken } from 'api/installToken';
import {
  buildInstallCommand,
  buildPmmServerURL,
  CredentialsMode,
  formatExpiresIn,
  Technology,
} from './InstallClientPage.utils';

export const InstallClientPage = () => {
  const [technology, setTechnology] = useState<Technology>('mysql');
  const [credentialsMode, setCredentialsMode] = useState<CredentialsMode>('env');
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

  // Tick once a second while a token is live, so the countdown chip refreshes.
  // Stops as soon as expiresAt is null (e.g. user cleared the field manually).
  useEffect(() => {
    if (!tokenExpiresAt) return undefined;
    const id = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(id);
  }, [tokenExpiresAt]);

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

  const handleCopy = async () => {
    await navigator.clipboard.writeText(command);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 2000);
  };

  const handleGenerateToken = async () => {
    setGenError(null);
    setGenLoading(true);
    try {
      const res = await createNodeInstallToken(technology, 0);
      setToken(res.token);
      // Server enforces a 15-minute hard cap (see install_token.go). If the
      // response unexpectedly omits expiresAt, fall back to "now + 15 min" so
      // the countdown still works and we don't silently lose the safety net.
      const expires = res.expiresAt
        ? new Date(res.expiresAt)
        : new Date(Date.now() + 15 * 60 * 1000);
      setTokenExpiresAt(expires);
      setNow(Date.now());
    } catch (e: unknown) {
      let msg = 'Failed to create token';
      if (axios.isAxiosError(e)) {
        const data = e.response?.data as { message?: string } | undefined;
        msg = data?.message ?? e.message;
      }
      setGenError(msg);
    } finally {
      setGenLoading(false);
    }
  };

  return (
    <Page title="Install PMM Client (One-step)">
      <Card variant="outlined">
        <CardContent>
          <Stack spacing={2}>
            <Alert severity="info">
              <Typography variant="body2" sx={{ mb: 1 }}>
                Choose installation options, then copy and run the generated command on your database
                node. The usual <code>curl … | bash</code> form has no interactive terminal on stdin,
                so use env variables or flags for database credentials unless you save the script and
                run it from a real shell.
              </Typography>
              <Typography variant="body2">
                <strong>Generated tokens are admin-role and valid for 15 minutes</strong> — treat the
                URL like a password.
              </Typography>
            </Alert>
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
              <FormControl fullWidth>
                <InputLabel id="technology-label">Technology</InputLabel>
                <Select
                  labelId="technology-label"
                  value={technology}
                  label="Technology"
                  onChange={(e) => setTechnology(e.target.value as Technology)}
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
                  onChange={(e) =>
                    setCredentialsMode(e.target.value as CredentialsMode)
                  }
                >
                  <MenuItem value="env">Include env variables (recommended for curl | bash)</MenuItem>
                  <MenuItem value="flags">Pass as script flags</MenuItem>
                  <MenuItem value="prompt">
                    Prompt on node (TTY only — save script and run in a terminal, not curl | bash)
                  </MenuItem>
                </Select>
                <FormHelperText>
                  Piping from curl gives the script stdin, not your keyboard; prompts only work when
                  stdin is a terminal (e.g. download the script, then{' '}
                  <code>sudo bash ./install-pmm-client.sh …</code>).
                </FormHelperText>
              </FormControl>
            </Stack>

            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
              <TextField
                fullWidth
                label="PMM host"
                value={pmmHost}
                onChange={(e) => setPmmHost(e.target.value)}
                helperText="Hostname or hostname:port for PMM_SERVER_URL (defaults to this page if empty)"
              />
              <TextField
                fullWidth
                type="password"
                label="Service token"
                value={token}
                onChange={(e) => {
                  setToken(e.target.value);
                  // User edited the field manually — drop the expiry so we
                  // stop ticking against a token they overrode.
                  setTokenExpiresAt(null);
                }}
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
                onChange={(e) => setNodeName(e.target.value)}
              />
              <TextField
                fullWidth
                label="Node address (optional)"
                value={nodeAddress}
                onChange={(e) => setNodeAddress(e.target.value)}
              />
            </Stack>

            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
              <TextField
                fullWidth
                label="DB user (optional)"
                value={dbUser}
                onChange={(e) => setDbUser(e.target.value)}
              />
              <TextField
                fullWidth
                type="password"
                label="DB password (optional in prompt mode)"
                value={dbPassword}
                onChange={(e) => setDbPassword(e.target.value)}
              />
            </Stack>

            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
              <TextField
                fullWidth
                label="DB host"
                value={dbHost}
                onChange={(e) => setDbHost(e.target.value)}
              />
              <TextField
                fullWidth
                label="DB port"
                value={dbPort}
                onChange={(e) => setDbPort(e.target.value)}
              />
              <TextField
                fullWidth
                label="Service name"
                value={dbServiceName}
                onChange={(e) => setDbServiceName(e.target.value)}
              />
            </Stack>

            {technology === 'postgresql' && (
              <TextField
                fullWidth
                label="PostgreSQL database (optional)"
                value={dbName}
                onChange={(e) => setDbName(e.target.value)}
              />
            )}
            {technology === 'mongodb' && (
              <TextField
                fullWidth
                label="MongoDB auth DB (optional)"
                value={dbAuthDB}
                onChange={(e) => setDbAuthDB(e.target.value)}
              />
            )}

            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
              <FormControlLabel
                control={
                  <Switch
                    checked={insecureTLS}
                    onChange={(e) => setInsecureTLS(e.target.checked)}
                  />
                }
                label="Use insecure TLS"
              />
              <FormControlLabel
                control={
                  <Switch
                    checked={registerForce}
                    onChange={(e) => setRegisterForce(e.target.checked)}
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
              <Button variant="contained" onClick={handleCopy}>
                Copy command
              </Button>
            </Box>
            {copied && <Alert severity="success">Command copied.</Alert>}
          </Stack>
        </CardContent>
      </Card>
    </Page>
  );
};
