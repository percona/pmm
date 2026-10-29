import DashboardOutlined from '@mui/icons-material/DashboardOutlined';
import Box from '@mui/material/Box';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemButton from '@mui/material/ListItemButton';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import ListSubheader from '@mui/material/ListSubheader';
import Paper from '@mui/material/Paper';
import { type FC, useId } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { Messages } from '../hub.messages';
import type {
  HubDashboardLink,
  HubDashboards,
  HubDoor,
  HubSectionState,
} from '../hub.types';
import { HubSection, SectionState } from './HubSection';
import { visuallyHidden } from './hub.sx';
import { formatAge } from './hub.time';

const DashboardGroup: FC<{
  title: string;
  links: HubDashboardLink[];
  empty?: string;
}> = ({ title, links, empty }) => {
  const id = useId();

  return (
    <List
      dense
      disablePadding
      aria-labelledby={id}
      subheader={
        <ListSubheader id={id} disableSticky>
          {title}
        </ListSubheader>
      }
    >
      {links.map((link) => (
        <ListItem key={link.uid} disablePadding>
          <ListItemButton component={RouterLink} to={link.to}>
            <ListItemIcon>
              <DashboardOutlined />
            </ListItemIcon>
            <ListItemText
              primary={
                <>
                  {link.label}
                  {link.context && (
                    <Box component="span" sx={visuallyHidden}>
                      {` ${link.context}`}
                    </Box>
                  )}
                </>
              }
              secondary={link.visitedAt && formatAge(link.visitedAt)}
            />
          </ListItemButton>
        </ListItem>
      ))}
      {links.length === 0 && empty && (
        <ListItem>
          <ListItemText
            secondary={empty}
            slotProps={{ secondary: { component: 'span' } }}
          />
        </ListItem>
      )}
    </List>
  );
};

export interface DashboardsSectionProps {
  dashboards: HubDashboards;
  /** Every app, so the ones without a tile stay a click away. */
  allApps?: HubDoor;
  state: HubSectionState;
  onRetry: () => void;
}

/** Dashboards: recommended for what runs, for developers, recently visited, all; then the way to every app. */
export const DashboardsSection: FC<DashboardsSectionProps> = ({
  dashboards,
  allApps,
  state,
  onRetry,
}) => {
  const { recommended, forDevelopers, recent, all } = dashboards;

  return (
    <HubSection
      id="dashboards"
      title={Messages.sections.dashboards}
      doors={[
        all.count == null
          ? all
          : { ...all, label: `${all.label} ${all.count}` },
        ...(allApps ? [allApps] : []),
      ]}
    >
      <SectionState state={state} onRetry={onRetry}>
        <Paper variant="outlined">
          {recommended.length > 0 && (
            <DashboardGroup
              title={Messages.dashboards.recommended}
              links={recommended}
            />
          )}
          {forDevelopers.length > 0 && (
            <DashboardGroup
              title={Messages.dashboards.forDevelopers}
              links={forDevelopers}
            />
          )}
          <DashboardGroup
            title={Messages.dashboards.recent}
            links={recent}
            empty={Messages.dashboards.recentEmpty}
          />
        </Paper>
      </SectionState>
    </HubSection>
  );
};
