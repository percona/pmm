import { Divider, Stack } from '@mui/material';
import { useChangelogs } from 'hooks/api/useUpdates';
import { FC } from 'react';
import { ReleaseNotes } from './release-notes';

export const Changelog: FC = () => {
  const { data, isLoading } = useChangelogs();
  const changelogs = data?.updates || [];

  if (isLoading || !changelogs.length) {
    return null;
  }

  return (
    <Stack>
      <Divider sx={{ mx: 2, my: 4 }} />
      {changelogs.map((changelog, idx) => (
        <Stack key={changelog.version} sx={{ px: 2 }}>
          <ReleaseNotes content={changelog.releaseNotesText} />
          {idx !== changelogs.length - 1 && <Divider sx={{ my: 4 }} />}
        </Stack>
      ))}
    </Stack>
  );
};
