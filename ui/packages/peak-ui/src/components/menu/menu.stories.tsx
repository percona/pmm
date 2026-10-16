import { useEffect, useId, useRef, useState, type ReactNode } from 'react';
import { type Meta, type StoryObj } from '@storybook/react';
import * as DocBlock from '@storybook/addon-docs/blocks';
import Button from '@mui/material/Button';
import Divider from '@mui/material/Divider';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import ListSubheader from '@mui/material/ListSubheader';
import Menu, { type MenuProps } from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import ArrowDropDown from '@mui/icons-material/ArrowDropDown';
import ContentCopyOutlined from '@mui/icons-material/ContentCopyOutlined';
import DeleteOutlined from '@mui/icons-material/DeleteOutlined';
import DownloadOutlined from '@mui/icons-material/DownloadOutlined';
import EditOutlined from '@mui/icons-material/EditOutlined';

interface MenuDemoProps {
  label: string;
  children: (close: () => void) => ReactNode;
  menuProps?: Partial<MenuProps>;
}

// Shown open on load without taking focus, so the docs page displays the menu;
// once closed, it opens from its button like any menu.
const MenuDemo = ({ label, children, menuProps }: MenuDemoProps) => {
  const id = useId();
  const buttonRef = useRef<HTMLButtonElement>(null);
  const [anchor, setAnchor] = useState<HTMLElement | null>(null);
  const [shownOnLoad, setShownOnLoad] = useState(true);

  useEffect(() => {
    setAnchor(buttonRef.current);
  }, []);

  const close = () => {
    setAnchor(null);
    setShownOnLoad(false);
  };

  return (
    <>
      <Button
        ref={buttonRef}
        variant="outlined"
        endIcon={<ArrowDropDown />}
        aria-haspopup="menu"
        aria-expanded={!!anchor}
        aria-controls={anchor ? id : undefined}
        onClick={(event) => setAnchor(event.currentTarget)}
      >
        {label}
      </Button>
      <Menu
        id={id}
        anchorEl={anchor}
        open={!!anchor}
        onClose={close}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
        slotProps={{ paper: { sx: { mt: 0.5 } } }}
        autoFocus={!shownOnLoad}
        disableAutoFocus={shownOnLoad}
        disableEnforceFocus={shownOnLoad}
        {...menuProps}
      >
        {children(close)}
      </Menu>
    </>
  );
};

const INTERVALS = ['Off', '5 seconds', '10 seconds', '30 seconds', '1 minute', '5 minutes'];

const IntervalMenu = () => {
  const [value, setValue] = useState('30 seconds');

  return (
    <MenuDemo label={`Refresh: ${value}`}>
      {(close) =>
        INTERVALS.map((option) => (
          <MenuItem
            key={option}
            selected={option === value}
            onClick={() => {
              setValue(option);
              close();
            }}
          >
            {option}
          </MenuItem>
        ))
      }
    </MenuDemo>
  );
};

const TIME_ZONES = [
  'UTC',
  'Europe/Lisbon',
  'Europe/London',
  'Europe/Madrid',
  'Europe/Berlin',
  'Europe/Kyiv',
  'America/New_York',
  'America/Chicago',
  'America/Denver',
  'America/Los_Angeles',
  'Asia/Kolkata',
  'Asia/Tokyo',
  'Australia/Sydney',
];

const meta: Meta<typeof Menu> = {
  title: 'Navigation/Menu',
  component: Menu,
  tags: ['autodocs', 'stable'],
  parameters: {
    layout: 'padded',
    docs: {
      // Each story in a frame of its own, so the menus it opens stay inside it.
      story: { inline: false },
      description: {
        component: [
          'MUI `Menu` as the Peak UI theme styles it, matching the Menu in the Figma Peak UI Kit: a list of actions or choices on a temporary surface, opened from a button or another control. Use MUI `Menu` and `MenuItem` directly; `MenuButton` pairs them with a contained button.',
          '',
          'The surface has 8 px corners, a 1 px border in the divider color and the Figma `elevation8` shadow, with 8 px above and below the rows. Rows are 40 px, with body1 labels 16 px in; `ListSubheader` (an overline label) and `Divider` group them. Every MUI popover shares the surface, `Select` menus included. A menu taller than the screen scrolls inside, and a `maxHeight` on the paper caps it sooner.',
          '',
          '**Note:** A menu is for actions and choices. For links between pages, use `NavItem` inside a navigation landmark: `role="menu"` makes screen readers announce links as menu items.',
        ].join('\n'),
      },
      page: () => (
        <>
          <DocBlock.Title />
          <DocBlock.Subtitle />
          <DocBlock.Description />
          <DocBlock.Primary />
          <DocBlock.Stories />
        </>
      ),
    },
  },
};

export default meta;
type Story = StoryObj<typeof Menu>;

export const Basic: Story = {
  render: () => (
    <MenuDemo label="Actions">
      {(close) =>
        ['Edit', 'Duplicate', 'Export', 'Delete'].map((action) => (
          <MenuItem key={action} onClick={close}>
            {action}
          </MenuItem>
        ))
      }
    </MenuDemo>
  ),
  parameters: { docs: { story: { iframeHeight: '280px' } } },
};

export const WithIcons: Story = {
  render: () => (
    <MenuDemo label="Actions">
      {(close) =>
        [
          { label: 'Edit', Icon: EditOutlined },
          { label: 'Duplicate', Icon: ContentCopyOutlined },
          { label: 'Export', Icon: DownloadOutlined },
          { label: 'Delete', Icon: DeleteOutlined },
        ].map(({ label, Icon }) => (
          <MenuItem key={label} onClick={close}>
            <ListItemIcon>
              <Icon />
            </ListItemIcon>
            <ListItemText>{label}</ListItemText>
          </MenuItem>
        ))
      }
    </MenuDemo>
  ),
  parameters: { docs: { story: { iframeHeight: '280px' } } },
};

export const WithSections: Story = {
  render: () => (
    <MenuDemo label="View">
      {(close) => [
        <ListSubheader key="sort">Sort by</ListSubheader>,
        <MenuItem key="name" onClick={close}>
          Name
        </MenuItem>,
        <MenuItem key="seen" selected onClick={close}>
          Last seen
        </MenuItem>,
        <MenuItem key="status" onClick={close}>
          Status
        </MenuItem>,
        <Divider key="divider" />,
        <ListSubheader key="show">Show</ListSubheader>,
        <MenuItem key="nodes" onClick={close}>
          Nodes
        </MenuItem>,
        <MenuItem key="services" onClick={close}>
          Services
        </MenuItem>,
      ]}
    </MenuDemo>
  ),
  parameters: { docs: { story: { iframeHeight: '420px' } } },
};

export const Selection: Story = {
  render: () => <IntervalMenu />,
  parameters: {
    docs: {
      description: {
        story:
          'A menu of choices marks the current one with `selected`, and opens with focus on it.',
      },
      story: { iframeHeight: '360px' },
    },
  },
};

export const MaxHeight: Story = {
  render: () => (
    <MenuDemo
      label="Time zone"
      menuProps={{ slotProps: { paper: { sx: { mt: 0.5, maxHeight: 232 } } } }}
    >
      {(close) =>
        TIME_ZONES.map((zone) => (
          <MenuItem key={zone} onClick={close}>
            {zone}
          </MenuItem>
        ))
      }
    </MenuDemo>
  ),
  parameters: {
    docs: {
      description: {
        story:
          'Capped at 232 px, the menu scrolls inside, and the half-shown last row signals there is more.',
      },
      story: { iframeHeight: '330px' },
    },
  },
};
