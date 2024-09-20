import {
  ContentPaste,
  Dangerous,
  Edit,
  Settings,
  ArrowDownward,
  MoreVert,
  AddCircle,
  MoreHoriz,
} from '@mui/icons-material';
import { CheckIcon, DashboardsIcon, NodeIcon, PerconaIcon } from 'icons';

export const IconMap: Record<string, JSX.Element> = {
  danger: <Dangerous htmlColor="#ff1744" />,
  note: <Edit htmlColor="#448aff" />,
  percona: <PerconaIcon />,
  configuration: <Settings />,
  inventory: <ContentPaste />,
  arrowdown: <ArrowDownward />,
  ellipsisv: <MoreVert />,
  settings: <Settings />,
  checks: <CheckIcon />,
  dashboards: <DashboardsIcon />,
  node: <NodeIcon />,
  addinstance: <AddCircle />,
  bouncingellipsis: <MoreHoriz />,
};
