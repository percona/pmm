import IconButton from '@mui/material/IconButton';
import { ShareIcon, Tooltip } from '@percona/peak-ui';
import { useCreateShortUrl } from 'hooks/api/useShortUrls';
import { enqueueSnackbar } from 'notistack';
import type { FC } from 'react';
import { constructUrl } from 'utils/link.utils';
import { Messages } from './AppHeader.messages';

/** Copies a link to the page as it is; a Grafana page gets a Grafana short URL, which keeps its state. */
export const ShareLinkButton: FC = () => {
  const { mutateAsync: createShortUrl } = useCreateShortUrl();

  const handleCopy = async () => {
    try {
      const path = constructUrl(location).replace(
        /\/pmm-ui\/(next\/)?graph\//,
        ''
      );
      const res = location.pathname.includes('/graph')
        ? await createShortUrl(path)
        : { url: window.location.href };

      if (navigator.clipboard && window.isSecureContext) {
        navigator.clipboard.writeText(res.url);

        enqueueSnackbar(Messages.copied, { variant: 'success' });
      } else {
        enqueueSnackbar(Messages.clipboardUnavailable, { variant: 'error' });
      }
    } catch {
      enqueueSnackbar(Messages.copyFailed, { variant: 'error' });
    }
  };

  return (
    <Tooltip title={Messages.share}>
      <IconButton aria-label={Messages.share} onClick={handleCopy}>
        <ShareIcon />
      </IconButton>
    </Tooltip>
  );
};
