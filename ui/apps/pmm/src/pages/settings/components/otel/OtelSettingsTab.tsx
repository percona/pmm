import { Card, CardContent, Stack } from '@mui/material';
import { FC } from 'react';
import type { Settings } from 'types/settings.types';
import { ServerOtelSection } from './ServerOtelSection';
import { LogParserPresetsSection } from './LogParserPresetsSection';
import { LogCollectorsSection } from './LogCollectorsSection';

export const OtelSettingsTab: FC<{ settings: Settings }> = ({ settings }) => (
  <Stack gap={3}>
    <ServerOtelSection settings={settings} />
    <Card variant="outlined">
      <CardContent>
        <LogParserPresetsSection />
      </CardContent>
    </Card>
    <Card variant="outlined">
      <CardContent>
        <LogCollectorsSection />
      </CardContent>
    </Card>
  </Stack>
);
