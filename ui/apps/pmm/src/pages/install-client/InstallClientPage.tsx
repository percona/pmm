import axios from 'axios';
import { useMemo, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  FormControl,
  FormControlLabel,
  InputLabel,
  MenuItem,
  Select,
  Stack,
  Switch,
  TextField,
  Typography,
} from '@mui/material';
import { Page } from 'components/page';
import { createNodeInstallToken } from 'api/installToken';
import {
  buildInstallCommand,
  buildPmmServerURL,
  CredentialsMode,
  Technology,
} from './InstallClientPage.utils';

export const InstallClientPage = () => {
  const [technology, setTechnology] = useState<Technology>('mysql');
  const [credentialsMode, setCredentialsMode] = useState<CredentialsMode>('prompt');
  const [token, setToken] = useState('');
  const [pmmHost, setPmmHost] = useState(() => window.location.host);
  const [insecureTLS, setInsecureTLS] = useState(false);
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
              Choose installation options, then copy and run the generated command
              on your database node.
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
                  <MenuItem value="prompt">Prompt on node (recommended)</MenuItem>
                  <MenuItem value="env">Include env variables</MenuItem>
                  <MenuItem value="flags">Pass as script flags</MenuItem>
                </Select>
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
                onChange={(e) => setToken(e.target.value)}
                helperText="Used only to render command locally in browser"
              />
            </Stack>
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} alignItems="flex-start">
              <Button
                variant="outlined"
                onClick={handleGenerateToken}
                disabled={genLoading}
              >
                {genLoading ? 'Generating…' : 'Generate short-lived token'}
              </Button>
              {genError && (
                <Alert severity="error" sx={{ flex: 1 }}>
                  {genError}
                </Alert>
              )}
            </Stack>

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
                label="DB user (optional in prompt mode)"
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
            <Alert severity="warning">
              Passwords in command-line args or env vars may be visible in shell
              history. Use prompt mode when possible.
            </Alert>
          </Stack>
        </CardContent>
      </Card>
    </Page>
  );
};
