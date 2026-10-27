import Box from '@mui/material/Box';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import { type FC, useState } from 'react';
import { DumpStatus } from 'types/dump.types';
import { CreateDatasetMockup } from './CreateDatasetMockup';
import { type ListDialog, DumpListMockup } from './DumpListMockup';
import { MOCK_DUMPS } from './mockups.data';

const SCREENS = [
  { id: 'list', label: 'List' },
  { id: 'empty', label: 'Empty list' },
  { id: 'bulk', label: 'Bulk selection' },
  { id: 'support', label: 'Send to Support' },
  { id: 'logs', label: 'Logs' },
  { id: 'delete', label: 'Delete' },
  { id: 'create', label: 'Create' },
  { id: 'create-filled', label: 'Create (filled)' },
] as const;

type ScreenId = (typeof SCREENS)[number]['id'];

const READY_IDS = MOCK_DUMPS.filter(
  ({ status }) => status === DumpStatus.Success
).map(({ dumpId }) => dumpId);

const LIST_DIALOGS: Partial<Record<ScreenId, ListDialog>> = {
  support: 'support',
  logs: 'logs',
  delete: 'delete',
};

const getInitialScreen = (): ScreenId => {
  const screen = new URLSearchParams(window.location.search).get('screen');
  return SCREENS.find(({ id }) => id === screen)?.id ?? 'list';
};

export const PmmDumpMockups: FC = () => {
  const [screen, setScreen] = useState<ScreenId>(getInitialScreen);

  const go = (next: ScreenId) => {
    setScreen(next);
    window.history.replaceState(null, '', `?screen=${next}`);
  };

  const renderScreen = () => {
    switch (screen) {
      case 'create':
      case 'create-filled':
        return (
          <CreateDatasetMockup
            variant={screen === 'create-filled' ? 'filled' : 'default'}
            onCancel={() => go('list')}
          />
        );
      case 'empty':
        return <DumpListMockup dumps={[]} onCreate={() => go('create')} />;
      default:
        return (
          <DumpListMockup
            dumps={MOCK_DUMPS}
            initialSelection={
              ['bulk', 'support', 'delete'].includes(screen) ? READY_IDS : []
            }
            initialDialog={LIST_DIALOGS[screen]}
            onCreate={() => go('create')}
          />
        );
    }
  };

  return (
    <Box width="100%" display="flex" flexDirection="column">
      <Box
        component="nav"
        aria-label="Mockup screens"
        sx={{ borderBottom: 1, borderColor: 'divider', px: 2 }}
        data-testid="mockup-nav"
      >
        <Tabs
          value={screen}
          onChange={(_, value: ScreenId) => go(value)}
          variant="scrollable"
        >
          {SCREENS.map(({ id, label }) => (
            <Tab key={id} value={id} label={label} />
          ))}
        </Tabs>
      </Box>
      <Box key={screen} flex={1} display="flex">
        {renderScreen()}
      </Box>
    </Box>
  );
};
