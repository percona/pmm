import ShareOutlinedIcon from '@mui/icons-material/ShareOutlined';
import PsychologyOutlinedIcon from '@mui/icons-material/PsychologyOutlined';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import { FC } from 'react';
import { useNavigate } from 'react-router-dom';
import { useCreateShortUrl } from 'hooks/api/useShortUrls';
import { constructUrl } from 'utils/link.utils';
import { enqueueSnackbar } from 'notistack';
import Tooltip from '@mui/material/Tooltip';
import { PMM_NEW_NAV_PATH } from 'lib/constants';

export const QanHeaderActions: FC = () => {
  const navigate = useNavigate();
  const { mutateAsync: createShortUrl } = useCreateShortUrl();

  const handleCopy = async () => {
    try {
      const path = constructUrl(location).replace(/\/pmm-ui\/(next\/)?graph\//, '');
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
      <Tooltip title="AI Insights" arrow>
        <IconButton
          data-testid="qan-header-actions-ai-insights-button"
          onClick={() => navigate(`${PMM_NEW_NAV_PATH}/qan/ai-insights`)}
          aria-label="AI Insights"
        >
          <PsychologyOutlinedIcon />
        </IconButton>
      </Tooltip>
      <Tooltip title="Share session settings" arrow>
        <IconButton
          data-testid="qan-header-actions-copy-button"
          onClick={handleCopy}
          aria-label="Share session settings"
        >
          <ShareOutlinedIcon />
        </IconButton>
      </Tooltip>
    </Stack>
  );
};

export default QanHeaderActions;
