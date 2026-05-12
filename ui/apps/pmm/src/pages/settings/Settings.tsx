import Box from '@mui/material/Box';
import CircularProgress from '@mui/material/CircularProgress';
import Stack from '@mui/material/Stack';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import { FC } from 'react';
import { Page } from 'components/page';
import { useSettings } from 'hooks/api/useSettings';
import { SshKeyForm } from './components/ssh-key/SshKeyForm';
import { MetricsResolutionForm } from './components/metrics-resolution/MetricsResolutionForm';
import { AdvancedSettingsForm } from './components/advanced/AdvancedSettingsForm';
import { Messages } from './Settings.messages';
import { TabValue } from './Settings.types';
import { useNavigate, useParams } from 'react-router-dom';
import { OrgRole } from 'types/user.types';
import { useUser } from 'contexts/user';

export const Settings: FC = () => {
  const { user } = useUser();
  const { tab = 'metrics-resolution' } = useParams<{ tab: TabValue }>();
  const {
    data: settings,
    isLoading,
    isEnabled,
  } = useSettings({
    enabled: !!user && user.isPMMAdmin,
  });
  const navigate = useNavigate();

  if (isLoading || (isEnabled && !settings)) {
    return (
      <Page title={Messages.title}>
        <Stack alignItems="center" py={4}>
          <CircularProgress data-testid="settings-loading" />
        </Stack>
      </Page>
    );
  }

  const setTab = (value: TabValue) => navigate(`/settings/${value}`);

  return (
    <Page
      title={Messages.title}
      fullWidth
      surface="paper"
      roles={[OrgRole.Admin]}
    >
      <Stack gap={3} sx={{ flex: 1 }}>
        <Tabs
          data-testid="settings-tabs"
          value={tab}
          onChange={(_, value: TabValue) => setTab(value)}
          variant="scrollable"
          scrollButtons="auto"
          sx={{ borderBottom: 1, borderColor: 'divider' }}
        >
          <Tab
            data-testid="settings-tab-metrics"
            value="metrics-resolution"
            label={Messages.tabs.metrics}
          />
          <Tab
            data-testid="settings-tab-advanced"
            value="advanced-settings"
            label={Messages.tabs.advanced}
          />
          <Tab
            data-testid="settings-tab-ssh"
            value="ssh-key"
            label={Messages.tabs.ssh}
          />
        </Tabs>

        <Box sx={{ flex: 1 }} data-testid="settings-tab-content">
          {tab === 'metrics-resolution' && (
            <MetricsResolutionForm settings={settings!} />
          )}
          {tab === 'advanced-settings' && (
            <AdvancedSettingsForm settings={settings!} />
          )}
          {tab === 'ssh-key' && <SshKeyForm settings={settings!} />}
        </Box>
      </Stack>
    </Page>
  );
};
