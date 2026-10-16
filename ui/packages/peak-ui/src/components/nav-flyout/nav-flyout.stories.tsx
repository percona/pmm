import { useEffect, useRef, useState, type ReactNode } from 'react';
import { type Meta, type StoryObj } from '@storybook/react';
import * as DocBlock from '@storybook/addon-docs/blocks';
import Box from '@mui/material/Box';
import ListItem from '@mui/material/ListItem';
import { listItemTextClasses } from '@mui/material/ListItemText';
import AccountTreeOutlined from '@mui/icons-material/AccountTreeOutlined';
import NotificationsOutlined from '@mui/icons-material/NotificationsOutlined';
import SettingsOutlined from '@mui/icons-material/SettingsOutlined';
import NavItem from '../nav-item';
import NavFlyout from './nav-flyout';

interface Group {
  id: string;
  label: string;
  icon: ReactNode;
  entries: string[];
  current?: string;
}

const GROUPS: Group[] = [
  {
    id: 'alerts',
    label: 'Alerts',
    icon: <NotificationsOutlined />,
    entries: ['Fired alerts', 'Alert rules', 'Contact points', 'Silences'],
  },
  {
    id: 'inventory',
    label: 'Inventory',
    icon: <AccountTreeOutlined />,
    entries: ['Services', 'Nodes'],
    current: 'Nodes',
  },
  {
    id: 'configuration',
    label: 'Configuration',
    icon: <SettingsOutlined />,
    entries: [
      'Settings',
      'Updates',
      'Organizations',
      'Stats and licenses',
      'Default preferences',
      'Users',
      'Teams',
      'Service accounts',
      'Access roles',
    ],
  },
];

// A collapsed sidebar: each group's icon opens its entries beside it. One group is shown open on
// load without taking focus, so the docs page displays it; once closed, panels open on click.
const CollapsedRail = ({ openOnLoad }: { openOnLoad: string }) => {
  const anchors = useRef<Record<string, HTMLDivElement | null>>({});
  const [open, setOpen] = useState<{ id: string; anchor: HTMLElement } | null>(null);
  const [shownOnLoad, setShownOnLoad] = useState(true);

  useEffect(() => {
    const anchor = anchors.current[openOnLoad];

    if (anchor) {
      setOpen({ id: openOnLoad, anchor });
    }
  }, [openOnLoad]);

  const close = () => {
    setOpen(null);
    setShownOnLoad(false);
  };

  return (
    <Box
      component="nav"
      aria-label="Main"
      sx={{ width: 48, py: 1, borderRight: 1, borderColor: 'divider' }}
    >
      {GROUPS.map((group) => {
        const panelId = `${group.id}-panel`;
        const isOpen = open?.id === group.id;

        return (
          <Box key={group.id}>
            <NavItem
              ref={(node) => {
                anchors.current[group.id] = node;
              }}
              text={group.label}
              icon={group.icon}
              selected={!!group.current}
              aria-current={group.current ? 'true' : undefined}
              aria-expanded={isOpen}
              aria-controls={isOpen ? panelId : undefined}
              onClick={(event) => setOpen({ id: group.id, anchor: event.currentTarget })}
              sx={{ px: 1.5, borderRadius: 0, [`.${listItemTextClasses.root}`]: { opacity: 0 } }}
            />
            <NavFlyout
              id={panelId}
              label={group.label}
              anchorEl={isOpen ? open.anchor : null}
              onClose={close}
              autoFocus={!shownOnLoad}
            >
              {group.entries.map((entry) => (
                <ListItem key={entry} disablePadding>
                  <NavItem
                    text={entry}
                    href={`#${entry.toLowerCase().replaceAll(' ', '-')}`}
                    selected={entry === group.current}
                    onClick={close}
                  />
                </ListItem>
              ))}
            </NavFlyout>
          </Box>
        );
      })}
    </Box>
  );
};

const meta: Meta<typeof NavFlyout> = {
  title: 'Navigation/Navigation Flyout',
  component: NavFlyout,
  tags: ['autodocs', 'experimental'],
  parameters: {
    layout: 'padded',
    docs: {
      // Each story in a frame of its own, so the panel it opens stays inside it.
      story: { inline: false },
      description: {
        component: [
          "A navigation group's entries in a panel beside its item, for a sidebar collapsed to a rail of icons where the group cannot open in place. The panel is a navigation landmark of links, not a menu: screen readers keep announcing its entries as links.",
          '',
          "It sits on the theme's popover surface, the Figma Menu's, and stays inside the viewport: it moves up when the group sits low on the screen and scrolls when the entries outgrow it. The group's name heads it in `sectionHeading` above a divider and stays in view as it scrolls. `NavItem` entries run edge to edge, with labels, or the icons of entries that have one, 16 px in.",
          '',
          'Opening moves focus to the current page, or else the first entry. Up and Down move between entries and wrap round, Home and End go to the ends, Tab reaches any actions, and Escape closes the panel and returns focus to the trigger.',
          '',
          "**Trigger:** use a `NavItem` with `aria-expanded` and `aria-controls`; it keeps its hover look while the panel is open. Wrap each entry in a `ListItem`, since the panel's list is a `ul`.",
        ].join('\n'),
      },
      page: () => (
        <>
          <DocBlock.Title />
          <DocBlock.Subtitle />
          <DocBlock.Description />
          <DocBlock.Primary />
          <DocBlock.Controls />
          <DocBlock.Stories />
        </>
      ),
    },
  },
};

export default meta;
type Story = StoryObj<typeof NavFlyout>;

export const InCollapsedRail: Story = {
  render: () => <CollapsedRail openOnLoad="inventory" />,
  parameters: { docs: { story: { iframeHeight: '240px' } } },
};

export const LongGroup: Story = {
  render: () => <CollapsedRail openOnLoad="configuration" />,
  parameters: {
    docs: {
      description: {
        story:
          'A group taller than the screen scrolls inside the panel, and its heading stays in view.',
      },
      story: { iframeHeight: '320px' },
    },
  },
};
