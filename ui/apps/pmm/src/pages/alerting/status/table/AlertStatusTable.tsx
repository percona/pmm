import { Table, useNavigableRows } from '@percona/percona-ui';
import { FC, useMemo, useState } from 'react';
import { ALERT_STATUS_COLUMNS } from './AlertStatusTable.constants';
import Switch from '@mui/material/Switch';
import FormControlLabel from '@mui/material/FormControlLabel';
import { AlertStatusTableProps } from './AlertStatusTable.types';
import { AlertRow, AlertsTableRow } from '../AlertsPage.types';
import { Link as RouterLink } from 'react-router-dom';
import {
  Stack,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Divider,
  ListItemIcon,
  ListItemText,
} from '@mui/material';
import { ALL_STATES_FILTER, STATE_OPTIONS } from '../AlertsPage.constants';
import {
  createAlertRuleEditUrl,
  createAlertRuleViewUrl,
  createSilenceUrl,
  getTableRows,
} from './AlertStatusTable.utils';
import { useTimezone } from 'hooks/utils/useTimezone';
import ContentCopyOutlinedIcon from '@mui/icons-material/ContentCopyOutlined';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';
import NotificationsOffOutlinedIcon from '@mui/icons-material/NotificationsOffOutlined';
import { Icon } from 'components/icon';

const AlertStatusTable: FC<AlertStatusTableProps> = ({
  rows,
  onOpenDetail,
  onNavigableRowsChange,
}) => {
  const timezone = useTimezone();
  const [groupByNodes, setGroupByNodes] = useState(false);
  const [selectedState, setSelectedState] = useState(ALL_STATES_FILTER);
  const tableRows = useMemo<AlertsTableRow[]>(
    () =>
      getTableRows({
        rows,
        timezone,
        groupByNodes,
        selectedState,
      }),
    [rows, timezone, groupByNodes, selectedState]
  );
  const { tableProps } = useNavigableRows<AlertsTableRow>({
    data: tableRows,
    onChange: onNavigableRowsChange,
  });

  return (
    <Table
      {...tableProps}
      tableName="alert-status"
      initialState={{
        pagination: {
          pageSize: 25,
          pageIndex: 0,
        },
      }}
      columns={ALERT_STATUS_COLUMNS}
      data={tableRows}
      getRowId={(row) => row.id}
      getSubRows={(row) => (row.type === 'node' ? row.alerts : undefined)}
      enableStickyHeader
      enableExpanding
      enableRowActions
      enableHiding={false}
      enableGlobalFilter={false}
      enableRowHoverAction
      rowHoverAction={(row) => {
        if (row.original.type === 'alert') {
          onOpenDetail(row.original as AlertRow);
        }
      }}
      renderTopToolbarCustomActions={() => (
        <Stack
          flex="1"
          direction="row"
          alignItems="center"
          justifyContent="space-between"
          gap={2}
        >
          <FormControl sx={{ width: 200 }} size="small">
            <InputLabel id="state">State</InputLabel>
            <Select
              labelId="state"
              label="State"
              value={selectedState}
              onChange={(e) => setSelectedState(e.target.value)}
            >
              {STATE_OPTIONS.map((opt) => (
                <MenuItem key={opt.value} value={opt.value}>
                  {opt.label}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
          <FormControlLabel
            label="Group by node"
            control={
              <Switch
                size="small"
                checked={groupByNodes}
                onChange={() => setGroupByNodes(!groupByNodes)}
              />
            }
          />
        </Stack>
      )}
      renderRowActionMenuItems={({ row, closeMenu }) => {
        if (row.original.type !== 'alert' || !row.original.ruleGroupUid) {
          return [];
        }

        return [
          <MenuItem
            key="notification-details"
            onClick={() => {
              onOpenDetail(row.original as AlertRow);
              closeMenu();
            }}
          >
            <ListItemIcon>
              <Icon name="bottom-panel-open" />
            </ListItemIcon>
            <ListItemText>Notification details</ListItemText>
          </MenuItem>,
          <MenuItem
            key="view"
            component={RouterLink}
            to={createAlertRuleViewUrl(row.original.ruleGroupUid)}
            onClick={(event) => {
              event.stopPropagation();
              closeMenu();
            }}
          >
            <ListItemIcon>
              <VisibilityOutlinedIcon />
            </ListItemIcon>
            <ListItemText>View alert rule</ListItemText>
          </MenuItem>,
          <MenuItem
            key="edit"
            component={RouterLink}
            to={createAlertRuleEditUrl(row.original.ruleGroupUid)}
            onClick={(event) => {
              event.stopPropagation();
              closeMenu();
            }}
          >
            <ListItemIcon>
              <EditOutlinedIcon />
            </ListItemIcon>
            <ListItemText>Edit alert rule</ListItemText>
          </MenuItem>,
          <MenuItem key="copy-as-text">
            <ListItemIcon>
              <ContentCopyOutlinedIcon />
            </ListItemIcon>
            <ListItemText>Copy as text</ListItemText>
          </MenuItem>,
          <Divider />,
          <MenuItem
            key="silence"
            component={RouterLink}
            to={createSilenceUrl(row.original.labels)}
            onClick={(event) => {
              event.stopPropagation();
              closeMenu();
            }}
          >
            <ListItemIcon>
              <NotificationsOffOutlinedIcon />
            </ListItemIcon>
            <ListItemText>Silence</ListItemText>
          </MenuItem>,
        ];
      }}
    />
  );
};

export default AlertStatusTable;
