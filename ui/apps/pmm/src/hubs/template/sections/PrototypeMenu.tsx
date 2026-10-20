import ArrowDropDown from '@mui/icons-material/ArrowDropDown';
import Check from '@mui/icons-material/Check';
import ScienceOutlined from '@mui/icons-material/ScienceOutlined';
import Button from '@mui/material/Button';
import Divider from '@mui/material/Divider';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import ListSubheader from '@mui/material/ListSubheader';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import { type FC, type ReactNode, useId, useState } from 'react';
import { Messages } from '../hub.messages';
import type {
  HubConnectionMode,
  HubOpsMode,
  HubParams,
  HubRole,
  HubRoleMode,
  HubScenario,
} from '../hub.types';

const SCENARIOS: HubScenario[] = ['healthy', 'wrong', 'empty', 'live'];
const OPS: HubOpsMode[] = ['live', 'on', 'off'];
const ROLES: HubRoleMode[] = ['live', 'admin', 'viewer'];
const CONNECTIONS: HubConnectionMode[] = ['live', 'lost'];

export interface PrototypeMenuProps {
  params: HubParams;
  setParams: (next: Partial<HubParams>) => void;
  /** The role and Operations the page shows, after the switches. */
  role: HubRole;
  operations: boolean;
  /** What the signed-in user has; on live data the switches can lower it, never raise it. */
  access: { admin: boolean; operations: boolean };
}

/** Prototype only: switches the hub between live data and sample states. */
export const PrototypeMenu: FC<PrototypeMenuProps> = ({
  params,
  setParams,
  role,
  operations,
  access,
}) => {
  const buttonId = useId();
  const menuId = useId();
  const groupId = useId();
  const [anchor, setAnchor] = useState<HTMLElement | null>(null);
  const { prototype } = Messages;
  const live = params.scenario === 'live';

  // Why a value cannot apply right now, if it cannot.
  const roleBlocked = (value: HubRoleMode) =>
    value === 'admin' && live && !access.admin
      ? prototype.hints.sampleOnly
      : undefined;
  const opsBlocked = (value: HubOpsMode) => {
    if (value !== 'on') {
      return undefined;
    }
    if (role !== 'admin') {
      return prototype.hints.adminOnly;
    }

    return live && !access.operations ? prototype.hints.sampleOnly : undefined;
  };

  // A value the page ignores reads as following the real user, as the page does.
  const shownRole: HubRoleMode =
    params.role === 'admin' && role !== 'admin' ? 'live' : params.role;
  const shownOps: HubOpsMode =
    params.ops === 'on' && !operations ? 'live' : params.ops;
  // Live data has a real connection, so only sample data can lose one.
  const shownConnection: HubConnectionMode = live ? 'live' : params.connection;
  const connectionBlocked = (value: HubConnectionMode) =>
    value === 'lost' && live ? prototype.hints.sampleOnly : undefined;

  const summary = [
    prototype.scenario[params.scenario],
    // Operations is on by default, so only Off is news.
    shownOps === 'off' && prototype.opsSummary.off,
    shownRole !== 'live' && prototype.role[shownRole],
    shownConnection === 'lost' && prototype.connectionSummary,
  ].filter((part): part is string => !!part);

  const choose = (next: Partial<HubParams>) => {
    setParams(next);
    setAnchor(null);
  };

  // Separators bound each radio set, and each item names its set, since MUI's
  // menu keyboard needs the items as direct children, so no role="group".
  const group = <T extends string>(
    key: string,
    title: string,
    values: T[],
    labels: Record<T, string>,
    selected: T,
    toParams: (value: T) => Partial<HubParams>,
    blocked: (value: T) => string | undefined = () => undefined
  ): ReactNode[] => {
    const titleId = `${groupId}-${key}`;

    return [
      ...(key === 'data'
        ? []
        : [<Divider key={`${key}-divider`} component="li" />]),
      <ListSubheader key={`${key}-title`} id={titleId} role="none">
        {title}
      </ListSubheader>,
      ...values.map((value, index) => {
        const hint = blocked(value);

        return (
          <MenuItem
            key={`${key}-${value}`}
            role="menuitemradio"
            aria-checked={value === selected}
            aria-describedby={titleId}
            aria-posinset={index + 1}
            aria-setsize={values.length}
            selected={value === selected}
            disabled={!!hint}
            onClick={() => choose(toParams(value))}
          >
            {value === selected && (
              <ListItemIcon>
                <Check />
              </ListItemIcon>
            )}
            <ListItemText inset={value !== selected} secondary={hint}>
              {labels[value]}
            </ListItemText>
          </MenuItem>
        );
      }),
    ];
  };

  return (
    <>
      <Button
        id={buttonId}
        variant="text"
        size="small"
        startIcon={<ScienceOutlined />}
        endIcon={<ArrowDropDown />}
        aria-haspopup="menu"
        aria-controls={anchor ? menuId : undefined}
        aria-expanded={anchor ? true : undefined}
        onClick={(event) => setAnchor(event.currentTarget)}
      >
        {prototype.trigger(summary)}
      </Button>
      <Menu
        id={menuId}
        anchorEl={anchor}
        open={!!anchor}
        onClose={() => setAnchor(null)}
        // Focus opens on the first choice, Data, not on the last checked item.
        variant="menu"
        slotProps={{ list: { 'aria-labelledby': buttonId } }}
      >
        {group(
          'data',
          prototype.groups.data,
          SCENARIOS,
          prototype.scenario,
          params.scenario,
          (scenario) => ({ scenario })
        )}
        {group(
          'ops',
          prototype.groups.operations,
          OPS,
          prototype.ops,
          shownOps,
          (ops) => ({ ops }),
          opsBlocked
        )}
        {group(
          'role',
          prototype.groups.role,
          ROLES,
          prototype.role,
          shownRole,
          (next) => ({ role: next }),
          roleBlocked
        )}
        {group(
          'connection',
          prototype.groups.connection,
          CONNECTIONS,
          prototype.connection,
          shownConnection,
          (connection) => ({ connection }),
          connectionBlocked
        )}
      </Menu>
    </>
  );
};
