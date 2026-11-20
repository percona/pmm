import StorageIcon from '@mui/icons-material/Storage';
import {
  HaproxyIcon,
  PerconaMoIcon,
  PerconaMyIcon,
  PerconaPoIcon,
  PerconaVaIcon,
  ProxyIcon,
} from '@percona/peak-ui';
import { AddServiceType } from '../AddServicePage.types';
import type { ServiceTypeCard } from './SelectServiceType.types';

export const SERVICE_TYPE_CARDS: ServiceTypeCard[] = [
  { type: AddServiceType.rds, icon: StorageIcon },
  { type: AddServiceType.mysql, icon: PerconaMyIcon },
  { type: AddServiceType.mongodb, icon: PerconaMoIcon },
  { type: AddServiceType.postgresql, icon: PerconaPoIcon },
  { type: AddServiceType.proxysql, icon: ProxyIcon },
  { type: AddServiceType.haproxy, icon: HaproxyIcon },
  { type: AddServiceType.valkey, icon: PerconaVaIcon },
  { type: AddServiceType.external, icon: StorageIcon },
  { type: AddServiceType.azure, icon: StorageIcon },
];
