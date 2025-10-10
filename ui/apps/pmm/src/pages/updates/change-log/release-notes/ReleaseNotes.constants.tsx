import Dangerous from '@mui/icons-material/DangerousOutlined';
import Edit from '@mui/icons-material/EditOutlined';
import Settings from '@mui/icons-material/SettingsOutlined';
import ContentPaste from '@mui/icons-material/ContentPasteOutlined';
import ArrowDownward from '@mui/icons-material/ArrowDownwardOutlined';
import MoreVert from '@mui/icons-material/MoreVertOutlined';
import AddCircle from '@mui/icons-material/AddCircleOutlineOutlined';
import MoreHoriz from '@mui/icons-material/MoreHorizOutlined';
import { Icon } from 'components/icon';

export const IconMap: Record<string, JSX.Element> = {
  danger: <Dangerous htmlColor="#ff1744" />,
  note: <Edit htmlColor="#448aff" />,
  percona: <Icon name="percona" />,
  configuration: <Settings />,
  inventory: <ContentPaste />,
  arrowdown: <ArrowDownward />,
  ellipsisv: <MoreVert />,
  settings: <Settings />,
  checks: <Icon name="check" />,
  dashboards: <Icon name="dashboards" />,
  node: <Icon name="graph-6" />,
  addinstance: <AddCircle />,
  bouncingellipsis: <MoreHoriz />,
};
