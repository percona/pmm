import ShareOutlinedIcon from '@mui/icons-material/ShareOutlined';
import ElectricBoltOutlinedIcon from '@mui/icons-material/ElectricBoltOutlined';
import Badge from '@mui/material/Badge';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import { FC, useState } from 'react';
import { useCreateShortUrl } from 'hooks/api/useShortUrls';
import { constructUrl } from 'utils/link.utils';
import { enqueueSnackbar } from 'notistack';
import { RunningAgentsModal } from 'components/modal-rt-running-agents';

export const QanHeaderActions: FC = () => {
  const { mutateAsync: createShortUrl } = useCreateShortUrl();
  const [openRunningAgentsModal, setOpenRunningAgentsModal] = useState(false);

  const handleCopy = async () => {
    const path = constructUrl(location).replace('/pmm-ui/next/graph/', '');
    const res = location.pathname.includes('/graph')
      ? await createShortUrl(path)
      : { url: window.location.href };

    if (navigator.clipboard && window.isSecureContext) {
      navigator.clipboard.writeText(res.url);

      enqueueSnackbar('Link copied to clipboard', { variant: 'success' });
    } else {
      enqueueSnackbar('Clipboard is not available', { variant: 'error' });
    }
  };

  const handleOpenRunningAgentsModal = () => {
    setOpenRunningAgentsModal(true);
  };

  const handleCloseRunningAgentsModal = () => {
    setOpenRunningAgentsModal(false);
  };

  return (
    <>
      <Stack gap={1} flex={1} flexDirection="row" justifyContent="flex-end">
        <IconButton
          data-testid="qan-header-actions-running-agents-button"
          onClick={handleOpenRunningAgentsModal}
        >
          <Badge color="warning" badgeContent={3}>
            <ElectricBoltOutlinedIcon />
          </Badge>
        </IconButton>
        <IconButton
          data-testid="qan-header-actions-copy-button"
          onClick={handleCopy}
        >
          <ShareOutlinedIcon />
        </IconButton>
      </Stack>
      <RunningAgentsModal
        open={openRunningAgentsModal}
        onClose={handleCloseRunningAgentsModal}
      />
    </>
  );
};

export default QanHeaderActions;
