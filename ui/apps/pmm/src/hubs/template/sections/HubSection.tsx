import ArrowForwardRounded from '@mui/icons-material/ArrowForwardRounded';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import Skeleton from '@mui/material/Skeleton';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import useMediaQuery from '@mui/material/useMediaQuery';
import { Tooltip } from '@percona/peak-ui';
import { PREVIEW_LEVEL_TOOLTIP } from 'components/preview-chip/PreviewChip.constants';
import type { FC, ReactNode } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { Messages } from '../hub.messages';
import type { HubDoor, HubSectionState } from '../hub.types';
import { DoorButton } from './DoorLink';
import { visuallyHidden } from './hub.sx';

export interface HubSectionProps {
  /** Anchor and id prefix: `#clusters`, `clusters-title`. */
  id: string;
  title: string;
  count?: number;
  /** Visually hidden noun after the count, such as "clusters". */
  countLabel?: string;
  /** The section's way out, right of the title. */
  door?: HubDoor;
  children: ReactNode;
}

const SectionDoor: FC<{ door: HubDoor }> = ({ door }) => {
  const theme = useTheme();
  const narrow = useMediaQuery(theme.breakpoints.down('sm'));

  if (!narrow) {
    return <DoorButton door={door} />;
  }

  const name = [
    door.label,
    door.context,
    door.preview && PREVIEW_LEVEL_TOOLTIP[door.preview],
  ]
    .filter(Boolean)
    .join(' ');

  return (
    <Tooltip title={name}>
      <IconButton component={RouterLink} to={door.to} aria-label={name}>
        <ArrowForwardRounded />
      </IconButton>
    </Tooltip>
  );
};

/** A page section: an h2 header row with a count and a door, then its body. */
export const HubSection: FC<HubSectionProps> = ({
  id,
  title,
  count,
  countLabel,
  door,
  children,
}) => (
  <Box
    component="section"
    id={id}
    aria-labelledby={`${id}-title`}
    sx={{ display: 'flex', flexDirection: 'column', gap: 1, minWidth: 0 }}
  >
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, minHeight: 40 }}>
      <Typography
        variant="sectionHeading"
        component="h2"
        id={`${id}-title`}
        tabIndex={-1}
      >
        {title}
      </Typography>
      {count != null && (
        <Typography variant="body2" color="text.secondary">
          {count}
          {countLabel && (
            <Box component="span" sx={visuallyHidden}>
              {` ${countLabel}`}
            </Box>
          )}
        </Typography>
      )}
      {door && (
        <Box sx={{ ml: 'auto', display: 'flex', alignItems: 'center' }}>
          <SectionDoor door={door} />
        </Box>
      )}
    </Box>
    {children}
  </Box>
);

export interface SectionStateProps {
  state: HubSectionState;
  onRetry: () => void;
  /** Rows of skeleton shown while the section first loads. */
  skeletonRows?: number;
  /** Show the warning above the content instead of in its place. */
  keepChildrenOnError?: boolean;
  children: ReactNode;
}

/** A section body that shows a skeleton while loading and a warning when its source failed. */
export const SectionState: FC<SectionStateProps> = ({
  state,
  onRetry,
  skeletonRows = 3,
  keepChildrenOnError = false,
  children,
}) => {
  if (state.status === 'loading') {
    return (
      <Box aria-busy="true">
        {Array.from({ length: skeletonRows }, (_, index) => (
          <Skeleton key={index} variant="text" height={40} />
        ))}
      </Box>
    );
  }

  if (state.status === 'error') {
    const warning = (
      <Alert
        severity="warning"
        action={
          <Button size="small" color="inherit" onClick={onRetry}>
            {Messages.sectionError.retry}
          </Button>
        }
      >
        {state.message ?? Messages.sectionError.fallback}
      </Alert>
    );

    return keepChildrenOnError || state.partial ? (
      <>
        {warning}
        {children}
      </>
    ) : (
      warning
    );
  }

  return <>{children}</>;
};
