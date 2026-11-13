/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

import type { ReactElement } from 'react';
import DnsOutlinedIcon from '@mui/icons-material/DnsOutlined';
import HubOutlinedIcon from '@mui/icons-material/HubOutlined';
import StorageOutlinedIcon from '@mui/icons-material/StorageOutlined';
import type { OmClusterType } from '../../../types';

export const TYPE_ICON: Partial<Record<OmClusterType, ReactElement>> = {
  CLUSTER_TYPE_SHARDED: <HubOutlinedIcon fontSize="small" />,
  CLUSTER_TYPE_REPLICA_SET: <StorageOutlinedIcon fontSize="small" />,
  CLUSTER_TYPE_STANDALONE: <DnsOutlinedIcon fontSize="small" />,
};
