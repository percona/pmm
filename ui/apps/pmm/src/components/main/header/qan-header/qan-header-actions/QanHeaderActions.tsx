import ShareOutlinedIcon from '@mui/icons-material/ShareOutlined';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import { FC } from 'react';
import { useCreateShortUrl } from 'hooks/api/useShortUrls';
import { constructUrl } from 'utils/link.utils';
import { enqueueSnackbar } from 'notistack';

export const QanHeaderActions: FC = () => {
  const { mutateAsync: createShortUrl } = useCreateShortUrl();

  const handleCopy = async () => {
    try {
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
    } catch (error) {
      enqueueSnackbar('Failed to copy link to clipboard', { variant: 'error' });
    }
  };

  return (
    <Stack gap={1} flex={1} flexDirection="row" justifyContent="flex-end">
      <IconButton
        data-testid="qan-header-actions-copy-button"
        onClick={handleCopy}
      >
        <ShareOutlinedIcon />
      </IconButton>
    </Stack>
  );
};

export default QanHeaderActions;
