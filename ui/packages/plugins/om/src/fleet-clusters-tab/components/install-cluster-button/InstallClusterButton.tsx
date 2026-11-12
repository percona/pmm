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

import { Link as RouterLink } from 'react-router-dom';
import Button from '@mui/material/Button';
import AddIcon from '@mui/icons-material/Add';
import { AdminAction } from '../../../components/AdminAction';
import { OM_ROUTE_NODES } from '../../../constants';
import { useOmBase } from '../../../useOmBase';
import { Messages } from '../../FleetClustersTab.messages';

/** Installing starts from the nodes to install on, so this leads to the Nodes page. */
export const InstallClusterButton = () => {
  const omBase = useOmBase();
  return (
    <AdminAction title={Messages.installTooltip}>
      <Button
        variant="contained"
        startIcon={<AddIcon />}
        component={RouterLink}
        to={`${omBase}/${OM_ROUTE_NODES}`}
      >
        {Messages.install}
      </Button>
    </AdminAction>
  );
};
