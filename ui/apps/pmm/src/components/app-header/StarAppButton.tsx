import Box from '@mui/material/Box';
import IconButton from '@mui/material/IconButton';
import { StarFillIcon, StarIcon, Tooltip } from '@percona/peak-ui';
import { visuallyHidden } from 'components/preview-chip/PreviewChip.styles';
import { useNavigation } from 'contexts/navigation';
import { type FC, useState } from 'react';
import { Messages } from './AppHeader.messages';

export interface StarAppButtonProps {
  /** The app's id in the Apps catalog. */
  id: string;
}

/** Stars an app, listing it under Starred in the sidebar, or takes it out again. */
export const StarAppButton: FC<StarAppButtonProps> = ({ id }) => {
  const { starredApps, toggleStarredApp } = useNavigation();
  const [announcement, setAnnouncement] = useState('');
  const starred = starredApps.includes(id);
  const label = starred ? Messages.unstar : Messages.star;

  return (
    <>
      <Tooltip title={label}>
        <IconButton
          aria-label={label}
          onClick={() => {
            toggleStarredApp(id);
            setAnnouncement(starred ? Messages.unstarred : Messages.starred);
          }}
        >
          {starred ? <StarFillIcon /> : <StarIcon />}
        </IconButton>
      </Tooltip>
      <Box aria-live="polite" sx={visuallyHidden}>
        {announcement}
      </Box>
    </>
  );
};
