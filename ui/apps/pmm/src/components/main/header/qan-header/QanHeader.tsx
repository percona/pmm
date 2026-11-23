import { FC } from 'react';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Messages } from './QanHeader.messages';
import QanHeaderActions from './qan-header-actions/QanHeaderActions';
import { useIsRealtimeQan } from 'hooks/utils/useLocation';

const QanHeader: FC = () => {
  const isRealtime = useIsRealtimeQan();

  return (
    <Stack
      sx={{
        mx: 2,
        pt: 1,
        gap: 1,
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        ...(isRealtime && { borderBottom: 1, borderColor: 'divider' }),
      }}
    >
      <Typography variant="h6" sx={{ my: 1 }}>
        {isRealtime ? Messages.titleRealtime : Messages.title}
      </Typography>
      <QanHeaderActions />
    </Stack>
  );
};

export default QanHeader;
