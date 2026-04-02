import Box from '@mui/material/Box';
import CircularProgress from '@mui/material/CircularProgress';
import Stack from '@mui/material/Stack';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import { FC, useState } from 'react';
import { Page } from 'components/page';
import { useSettings } from 'hooks/api/useSettings';
import { SshKeyForm } from './components/ssh-key/SshKeyForm';
import { MetricsResolutionForm } from './components/metrics-resolution/MetricsResolutionForm';
import { AdvancedSettingsForm } from './components/advanced/AdvancedSettingsForm';
import { Messages } from './Settings.messages';
import { TabValue } from './Settings.types';

export const Settings: FC = () => {
  const [tab, setTab] = useState<TabValue>('advanced');
  const { data: settings, isLoading } = useSettings();

  if (isLoading || !settings) {
    return (
      <Page title={Messages.title}>
        <Stack alignItems="center" py={4}>
          <CircularProgress data-testid="settings-loading" />
        </Stack>
      </Page>
    );
  }

  return (
    <Page title={Messages.title} fullWidth>
      <Stack gap={3}>
        <Tabs
          value={tab}
          onChange={(_, value: TabValue) => setTab(value)}
          sx={{ borderBottom: 1, borderColor: 'divider' }}
        >
          <Tab
            data-testid="settings-tab-advanced"
            value="advanced"
            label={Messages.tabs.advanced}
          />
          <Tab
            data-testid="settings-tab-ssh"
            value="ssh"
            label={Messages.tabs.ssh}
          />
          <Tab
            data-testid="settings-tab-metrics"
            value="metrics"
            label={Messages.tabs.metrics}
          />
        </Tabs>

        <Box sx={{ py: 2 }}>
          {tab === 'advanced' && <AdvancedSettingsForm settings={settings} />}
          {tab === 'ssh' && <SshKeyForm settings={settings} />}
          {tab === 'metrics' && <MetricsResolutionForm settings={settings} />}
        </Box>
      </Stack>
    </Page>
  );
};
