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

import { useMemo, useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import Button from '@mui/material/Button';
import InputAdornment from '@mui/material/InputAdornment';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import MenuItem from '@mui/material/MenuItem';
import MenuList from '@mui/material/MenuList';
import Popover from '@mui/material/Popover';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import CheckIcon from '@mui/icons-material/Check';
import SearchIcon from '@mui/icons-material/Search';
import UnfoldMoreIcon from '@mui/icons-material/UnfoldMore';
import { CLUSTER_TYPE_LABEL, clusterPath } from '../../../constants';
import { ClusterHealthIcon } from '../../../fleet-clusters-tab/components/cluster-health-icon';
import {
  clusterDisplayName,
  environmentLabel,
} from '../../../fleet-clusters-tab/FleetClustersTab.utils';
import type { OmClusterRow } from '../../../types';
import { useOmBase } from '../../../useOmBase';
import { Messages } from '../../ClusterDetails.messages';

export const ClusterSwitcher = ({
  current,
  clusters,
}: {
  current: OmClusterRow;
  clusters: OmClusterRow[];
}) => {
  const base = useOmBase();
  const [anchor, setAnchor] = useState<HTMLElement | null>(null);
  const [query, setQuery] = useState('');
  const sorted = useMemo(
    () =>
      [...clusters].sort((a, b) =>
        clusterDisplayName(a).localeCompare(clusterDisplayName(b))
      ),
    [clusters]
  );
  const open = anchor !== null;
  const needle = query.trim().toLowerCase();
  const matches = needle
    ? sorted.filter((cluster) =>
        clusterDisplayName(cluster).toLowerCase().includes(needle)
      )
    : sorted;

  const close = () => {
    setAnchor(null);
    setQuery('');
  };

  return (
    <>
      <Button
        variant="outlined"
        color="inherit"
        size="small"
        endIcon={<UnfoldMoreIcon fontSize="small" />}
        aria-label={Messages.switcher.open}
        aria-haspopup="dialog"
        onClick={(event) => setAnchor(event.currentTarget)}
        sx={{ fontWeight: 600, textTransform: 'none' }}
        data-testid="om-cluster-switcher"
      >
        {clusterDisplayName(current)}
      </Button>
      <Popover
        open={open}
        anchorEl={anchor}
        onClose={close}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
        slotProps={{ paper: { sx: { width: 320, mt: 0.5 } } }}
      >
        <TextField
          autoFocus
          fullWidth
          size="small"
          placeholder={Messages.switcher.search}
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          sx={{ p: 1 }}
          slotProps={{
            htmlInput: { 'aria-label': Messages.switcher.search },
            input: {
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon fontSize="small" />
                </InputAdornment>
              ),
            },
          }}
        />
        {open && matches.length > 0 && (
          <MenuList
            dense
            sx={{ maxHeight: 360, overflowY: 'auto', overflowX: 'hidden' }}
          >
            {matches.map((cluster) => (
              <MenuItem
                key={cluster.id}
                component={RouterLink}
                to={clusterPath(base, cluster.id)}
                selected={cluster.id === current.id}
                onClick={close}
                sx={{ whiteSpace: 'normal' }}
              >
                <ListItemIcon>
                  <ClusterHealthIcon health={cluster.health} labelled />
                </ListItemIcon>
                <ListItemText
                  primary={clusterDisplayName(cluster)}
                  secondary={[
                    CLUSTER_TYPE_LABEL[cluster.type],
                    environmentLabel(cluster),
                  ].join(' · ')}
                />
                {cluster.id === current.id && (
                  <CheckIcon fontSize="small" color="primary" />
                )}
              </MenuItem>
            ))}
          </MenuList>
        )}
        {open && !matches.length && (
          <Typography
            variant="body2"
            color="text.secondary"
            sx={{ px: 2, pb: 2 }}
          >
            {Messages.switcher.empty}
          </Typography>
        )}
      </Popover>
    </>
  );
};
