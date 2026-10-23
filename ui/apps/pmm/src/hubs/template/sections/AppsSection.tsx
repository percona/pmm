import Box from '@mui/material/Box';
import Card from '@mui/material/Card';
import CardActions from '@mui/material/CardActions';
import CardContent from '@mui/material/CardContent';
import Typography from '@mui/material/Typography';
import type { FC, ReactNode } from 'react';
import { PreviewChip } from 'components/preview-chip';
import { Messages } from '../hub.messages';
import type { HubApp, HubDoor, HubSectionState } from '../hub.types';
import { DoorButton, DoorLink } from './DoorLink';
import { HubSection, SectionState } from './HubSection';

// Turns each door's label inside the summary into that door, in order.
const withDoors = (text: string, doors: HubDoor[] = []): ReactNode[] => {
  const parts: ReactNode[] = [];
  let rest = text;

  doors.forEach((door) => {
    const at = rest.indexOf(door.label);

    if (at < 0) {
      return;
    }

    parts.push(
      rest.slice(0, at),
      <DoorLink
        key={`${door.label}:${door.to}`}
        door={door}
        underline="always"
      />
    );
    rest = rest.slice(at + door.label.length);
  });
  parts.push(rest);

  return parts;
};

const AppCard: FC<{ app: HubApp }> = ({ app }) => {
  const Icon = app.icon;

  return (
    <Card variant="outlined" data-testid={`hub-app-${app.id}`}>
      <CardContent
        sx={{
          display: 'flex',
          flexDirection: 'column',
          gap: 1,
          px: 2,
          pb: 1,
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Icon />
          <Typography variant="h6" component="h3">
            {app.name}
          </Typography>
          {app.preview && (
            <Box component="span" sx={{ display: 'inline-flex' }}>
              <PreviewChip level={app.preview} />
            </Box>
          )}
        </Box>
        {app.summary && (
          <Typography variant="body2" color="text.secondary">
            {withDoors(app.summary.text, app.summary.doors)}
          </Typography>
        )}
      </CardContent>
      {app.pages.length > 0 && (
        <CardActions
          disableSpacing
          sx={{ px: 1, pt: 0, flexWrap: 'wrap', gap: 1 }}
        >
          {app.pages.map((page) => (
            <DoorButton key={`${page.label}:${page.to}`} door={page} />
          ))}
        </CardActions>
      )}
    </Card>
  );
};

export interface AppsSectionProps {
  apps: HubApp[];
  door?: HubDoor;
  state: HubSectionState;
  onRetry: () => void;
}

/** Apps: a card per app that works on this technology, linking into its pages. */
export const AppsSection: FC<AppsSectionProps> = ({
  apps,
  door,
  state,
  onRetry,
}) => (
  <HubSection id="apps" title={Messages.sections.apps} door={door}>
    <SectionState state={state} onRetry={onRetry} keepChildrenOnError>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
        {apps.map((app) => (
          <AppCard key={app.id} app={app} />
        ))}
      </Box>
    </SectionState>
  </HubSection>
);
