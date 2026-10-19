import ArrowDropDown from '@mui/icons-material/ArrowDropDown';
import Check from '@mui/icons-material/Check';
import ScienceOutlined from '@mui/icons-material/ScienceOutlined';
import Button from '@mui/material/Button';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import ListSubheader from '@mui/material/ListSubheader';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import { type FC, type ReactNode, useId, useState } from 'react';
import { Messages } from '../hub.messages';
import type {
  HubOpsMode,
  HubParams,
  HubRoleMode,
  HubScenario,
} from '../hub.types';

const SCENARIOS: HubScenario[] = ['live', 'empty', 'healthy', 'wrong'];
const OPS: HubOpsMode[] = ['live', 'on', 'off'];
const ROLES: HubRoleMode[] = ['live', 'admin', 'viewer'];

export interface PrototypeMenuProps {
  params: HubParams;
  setParams: (next: Partial<HubParams>) => void;
}

/** Prototype only: switches the hub between live data and sample states. */
export const PrototypeMenu: FC<PrototypeMenuProps> = ({
  params,
  setParams,
}) => {
  const buttonId = useId();
  const menuId = useId();
  const [anchor, setAnchor] = useState<HTMLElement | null>(null);
  const { prototype } = Messages;

  const summary = [
    prototype.scenario[params.scenario],
    params.ops !== 'live' && prototype.opsSummary[params.ops],
    params.role !== 'live' && prototype.role[params.role],
  ].filter((part): part is string => !!part);

  const choose = (next: Partial<HubParams>) => {
    setParams(next);
    setAnchor(null);
  };

  const group = <T extends string>(
    title: string,
    values: T[],
    labels: Record<T, string>,
    selected: T,
    toParams: (value: T) => Partial<HubParams>
  ): ReactNode[] => [
    <ListSubheader key={`${title}-title`}>{title}</ListSubheader>,
    ...values.map((value) => (
      <MenuItem
        key={`${title}-${value}`}
        role="menuitemradio"
        aria-checked={value === selected}
        selected={value === selected}
        onClick={() => choose(toParams(value))}
      >
        {value === selected && (
          <ListItemIcon>
            <Check />
          </ListItemIcon>
        )}
        <ListItemText inset={value !== selected}>{labels[value]}</ListItemText>
      </MenuItem>
    )),
  ];

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
        slotProps={{ list: { 'aria-labelledby': buttonId } }}
      >
        {group(
          prototype.groups.data,
          SCENARIOS,
          prototype.scenario,
          params.scenario,
          (scenario) => ({ scenario })
        )}
        {group(
          prototype.groups.operations,
          OPS,
          prototype.ops,
          params.ops,
          (ops) => ({ ops })
        )}
        {group(
          prototype.groups.role,
          ROLES,
          prototype.role,
          params.role,
          (role) => ({ role })
        )}
      </Menu>
    </>
  );
};
