import { FC } from 'react';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Messages } from './QanHeader.messages';
import QanHeaderTabs from './qan-header-tabs/QanHeaderTabs';
import QanHeaderActions from './qan-header-actions/QanHeaderActions';
import Divider from '@mui/material/Divider';
import { useIsRealTimeQan } from 'hooks/utils/useLocation';

const QanHeader: FC = () => {
  const isRealTime = useIsRealTimeQan();

  return (
    <>
      <Stack
        sx={{
          pt: 1,
          px: 2,
          gap: 3,
          flexDirection: 'row',
          justifyContent: 'flex-start',
          alignItems: 'center',
        }}
      >
        <Typography variant="h6">{Messages.title}</Typography>
        <QanHeaderTabs />
        <QanHeaderActions />
      </Stack>
      {isRealTime && <Divider sx={{ mx: 2, borderRadius: 4 }} />}{' '}
    </>
  );
};

export default QanHeader;
