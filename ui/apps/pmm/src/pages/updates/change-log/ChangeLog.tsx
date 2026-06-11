import { Divider, Stack } from '@mui/material';
import { useChangeLogs } from 'hooks/api/useUpdates';
import { FC } from 'react';
import { ReleaseNotes } from './release-notes';

export const ChangeLog: FC = () => {
  const { data, isLoading } = useChangeLogs();
  const changeLogs = data?.updates || [];

  if (isLoading || !changeLogs.length) {
    return null;
  }

  return (
    <Stack>
      <Divider sx={{ mx: 2, my: 4 }} />
      {changeLogs.map((changeLog, idx) => (
        <Stack key={changeLog.version} sx={{ px: 2 }}>
          <ReleaseNotes content={changeLog.releaseNotesText} />
          {idx !== changeLogs.length - 1 && <Divider sx={{ my: 4 }} />}
        </Stack>
      ))}
    </Stack>
  );
};
