import { FC } from 'react';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Messages } from './QanHeader.messages';
import QanHeaderTabs from './qan-header-tabs/QanHeaderTabs';
import QanHeaderActions from './qan-header-actions/QanHeaderActions';
import { useIsRealtimeQan } from 'hooks/utils/useLocation';

const QanHeader: FC = () => {
  const isRealtime = useIsRealtimeQan();

  return (
    <Stack
      sx={{
        mx: 2,
        pt: 1,
        columnGap: 3,
        flexDirection: 'row',
        flexWrap: 'wrap',
        alignItems: 'center',
        // The border lives on the container so the tabs always sit flush
        // against it, whether they share the title's row or wrap below it
        ...(isRealtime && { borderBottom: 1, borderColor: 'divider' }),
      }}
    >
      <Typography variant="h6" sx={{ my: 1 }}>
        {Messages.title}
      </Typography>
      <Stack
        sx={{
          // Wraps under the title as a unit when narrower than the basis,
          // then shrinks freely and lets the tabs scroll internally
          flex: '1 1 320px',
          minWidth: 0,
          gap: 1,
          flexDirection: 'row',
          alignItems: 'center',
        }}
      >
        <QanHeaderTabs />
        <QanHeaderActions />
      </Stack>
    </Stack>
  );
};

export default QanHeader;
