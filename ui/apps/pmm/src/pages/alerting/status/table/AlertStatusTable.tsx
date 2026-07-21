import { Table, useNavigableRows } from '@percona/percona-ui';
import { type MRT_Row } from 'material-react-table';
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
  createUnsilenceUrl,
  filterTriggeredAt,
  getTableRows,
} from './AlertStatusTable.utils';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';
import NotificationsOffOutlinedIcon from '@mui/icons-material/NotificationsOffOutlined';
import NotificationsActiveOutlinedIcon from '@mui/icons-material/NotificationsActiveOutlined';
import { Icon } from 'components/icon';
import { Messages } from './AlertStatusTable.messages';
import { useUser } from 'contexts/user';
import AlertsStatusEmptyState from './empty-state/AlertsStatusEmptyState';

const AlertStatusTable: FC<AlertStatusTableProps> = ({
  rows,
  onOpenDetail,
  onNavigableRowsChange,
}) => {
  const [groupByNodes, setGroupByNodes] = useState(false);
  const [selectedState, setSelectedState] = useState(ALL_STATES_FILTER);
  const tableRows = useMemo<AlertsTableRow[]>(
    () =>
      getTableRows({
        rows,
        groupByNodes,
        selectedState,
      }),
    [rows, groupByNodes, selectedState]
  );
  const { tableProps } = useNavigableRows<AlertsTableRow>({
    data: tableRows,
    onChange: onNavigableRowsChange,
  });
  const { user } = useUser();

  return (
    <Table
      {...tableProps}
      tableName="alert-status"
      initialState={{
        expanded: true,
        pagination: {
          pageSize: 25,
          pageIndex: 0,
        },
      }}
      displayColumnDefOptions={{
        'mrt-row-expand': {
          size: 60,
          grow: false,
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
      // Node group rows only carry nodeId/state, so filter on the alert rows
      // and keep the groups that still have matching alerts.
      filterFromLeafRows
      filterFns={{
        // MRT's 'betweenInclusive' compares values as lowercased strings,
        // which breaks Date comparisons.
        triggeredAtRangeFilterFn: (row, id, filterValue) =>
          filterTriggeredAt(row as MRT_Row<AlertsTableRow>, id, filterValue),
      }}
      enableRowHoverAction
      rowHoverAction={(row) => {
        if (row.original.type === 'alert') {
          onOpenDetail(row.original as AlertRow);
        }
      }}
      emptyState={<AlertsStatusEmptyState />}
      renderTopToolbarCustomActions={() => (
        <Stack
          flex="1"
          direction="row"
          alignItems="center"
          justifyContent="space-between"
          gap={2}
        >
          <FormControl sx={{ width: 200 }} size="small">
            <InputLabel id="state">{Messages.state}</InputLabel>
            <Select
              labelId="state"
              label={Messages.state}
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
            label={Messages.groupByNode}
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
        if (row.original.type !== 'alert') {
          return [];
        }

        const alert = row.original as AlertRow;
        // View/Edit need the Grafana rule uid; the other actions work without it.
        const { ruleGroupUid } = alert;
        const items = [
          <MenuItem
            key="notification-details"
            onClick={() => {
              onOpenDetail(alert);
              closeMenu();
            }}
          >
            <ListItemIcon>
              <Icon name="bottom-panel-open" />
            </ListItemIcon>
            <ListItemText>{Messages.notificationDetails}</ListItemText>
          </MenuItem>,
        ];

        if (ruleGroupUid) {
          items.push(
            <MenuItem
              key="view"
              component={RouterLink}
              to={createAlertRuleViewUrl(ruleGroupUid)}
              onClick={(event) => {
                event.stopPropagation();
                closeMenu();
              }}
            >
              <ListItemIcon>
                <VisibilityOutlinedIcon />
              </ListItemIcon>
              <ListItemText>{Messages.viewAlertRule}</ListItemText>
            </MenuItem>
          );

          if (user?.isEditor) {
            items.push(
              <MenuItem
                key="edit"
                component={RouterLink}
                to={createAlertRuleEditUrl(ruleGroupUid)}
                onClick={(event) => {
                  event.stopPropagation();
                  closeMenu();
                }}
              >
                <ListItemIcon>
                  <EditOutlinedIcon />
                </ListItemIcon>
                <ListItemText>{Messages.editAlertRule}</ListItemText>
              </MenuItem>
            );
          }
        }

        if (user?.isEditor) {
          items.push(
            <Divider key="divider" />,
            alert.silenced ? (
              <MenuItem
                key="unsilence"
                component={RouterLink}
                to={createUnsilenceUrl()}
                onClick={(event) => {
                  event.stopPropagation();
                  closeMenu();
                }}
              >
                <ListItemIcon>
                  <NotificationsActiveOutlinedIcon />
                </ListItemIcon>
                <ListItemText>{Messages.unsilence}</ListItemText>
              </MenuItem>
            ) : (
              <MenuItem
                key="silence"
                component={RouterLink}
                to={createSilenceUrl(alert.labels)}
                onClick={(event) => {
                  event.stopPropagation();
                  closeMenu();
                }}
              >
                <ListItemIcon>
                  <NotificationsOffOutlinedIcon />
                </ListItemIcon>
                <ListItemText>{Messages.silence}</ListItemText>
              </MenuItem>
            )
          );
        }

        return items;
      }}
    />
  );
};

export default AlertStatusTable;
