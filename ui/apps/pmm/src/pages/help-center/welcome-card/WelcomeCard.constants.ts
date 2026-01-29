import { Messages } from './WelcomeCard.messages';
import SvgIcon from '@mui/material/SvgIcon';
import BarChartIcon from '@mui/icons-material/BarChart';
import SecurityIcon from '@mui/icons-material/Security';
import SpeedIcon from '@mui/icons-material/Speed';
import { PMM_NEW_NAV_GRAFANA_PATH } from 'lib/constants';

export const WELCOME_CARD_LIST: Array<{
  icon: typeof SvgIcon;
  content: string;
}> = [
  {
    icon: BarChartIcon,
    content: Messages.points.spot,
  },
  {
    icon: SecurityIcon,
    content: Messages.points.keep,
  },
  {
    icon: SpeedIcon,
    content: Messages.points.backup,
  },
];

export const ADD_SERVICE_LINK = `${PMM_NEW_NAV_GRAFANA_PATH}/add-instance`;
