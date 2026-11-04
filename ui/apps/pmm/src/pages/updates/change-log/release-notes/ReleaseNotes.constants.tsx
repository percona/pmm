import Dangerous from '@mui/icons-material/DangerousOutlined';
import Edit from '@mui/icons-material/EditOutlined';
import Settings from '@mui/icons-material/SettingsOutlined';
import ContentPaste from '@mui/icons-material/ContentPasteOutlined';
import ArrowDownward from '@mui/icons-material/ArrowDownwardOutlined';
import MoreVert from '@mui/icons-material/MoreVertOutlined';
import AddCircle from '@mui/icons-material/AddCircleOutlineOutlined';
import MoreHoriz from '@mui/icons-material/MoreHorizOutlined';
import DashboardOutlined from '@mui/icons-material/DashboardOutlined';
import {
  Graph6Icon,
  NetworkIntelligenceIcon,
  PerconaIcon,
} from '@percona/peak-ui';

export const IconMap: Record<string, JSX.Element> = {
  danger: <Dangerous htmlColor="#ff1744" />,
  note: <Edit htmlColor="#448aff" />,
  percona: <PerconaIcon />,
  configuration: <Settings />,
  inventory: <ContentPaste />,
  arrowdown: <ArrowDownward />,
  ellipsisv: <MoreVert />,
  settings: <Settings />,
  checks: <NetworkIntelligenceIcon />,
  dashboards: <DashboardOutlined />,
  node: <Graph6Icon />,
  addinstance: <AddCircle />,
  bouncingellipsis: <MoreHoriz />,
};
