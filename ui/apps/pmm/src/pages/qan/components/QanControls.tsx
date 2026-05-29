import {
  Button,
  FormControl,
  InputLabel,
  MenuItem,
  Select,
  Stack,
  TextField,
  Tooltip,
} from '@mui/material';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import { FC, useCallback, useEffect, useMemo, useState } from 'react';
import { useSnackbar } from 'notistack';
import { useQanPanelActions, useQanPanelState } from '../hooks/useQanPanelState';
import {
  buildNativeQanShareLink,
  toDatetimeLocalValue,
  toUnixTimestamp,
} from '../utils/qanTools';
import type { QanGroupBy } from 'types/qan.types';
import { QanManageColumns } from './QanManageColumns';
import { QanControlsToolbar } from './QanControlsToolbar';

const SEARCH_DEBOUNCE_MS = 400;

const GROUP_BY_OPTIONS: { value: QanGroupBy; label: string }[] = [
  { value: 'queryid', label: 'Query' },
  { value: 'service_name', label: 'Service' },
  { value: 'database', label: 'Database' },
  { value: 'schema', label: 'Schema' },
  { value: 'username', label: 'User' },
  { value: 'client_host', label: 'Client host' },
];

export const QanControls: FC = () => {
  const state = useQanPanelState();
  const actions = useQanPanelActions();
  const { enqueueSnackbar } = useSnackbar();
  const [searchDraft, setSearchDraft] = useState(state.dimensionSearchText ?? '');

  useEffect(() => {
    setSearchDraft(state.dimensionSearchText ?? '');
  }, [state.dimensionSearchText]);

  useEffect(() => {
    const id = window.setTimeout(() => {
      if (searchDraft !== (state.dimensionSearchText ?? '')) {
        actions.setSearchText(searchDraft);
      }
    }, SEARCH_DEBOUNCE_MS);
    return () => clearTimeout(id);
  }, [searchDraft, state.dimensionSearchText, actions]);

  const fromLocal = useMemo(() => toDatetimeLocalValue(state.from), [state.from]);
  const toLocal = useMemo(() => toDatetimeLocalValue(state.to), [state.to]);

  const copyLink = useCallback(() => {
    const link = buildNativeQanShareLink(
      toUnixTimestamp(state.from),
      toUnixTimestamp(state.to)
    );
    void navigator.clipboard.writeText(link);
    enqueueSnackbar('Link copied to clipboard', { variant: 'success' });
  }, [state.from, state.to, enqueueSnackbar]);

  return (
    <Stack
      direction="row"
      alignItems="center"
      justifyContent="space-between"
      spacing={2}
      sx={{
        minHeight: 48,
        py: 1,
        borderBottom: 1,
        borderColor: 'divider',
        mb: 1,
        flexWrap: 'wrap',
        rowGap: 1,
      }}
      data-testid="qan-controls"
    >
      <QanControlsToolbar />
      <Stack
        direction="row"
        alignItems="center"
        spacing={1.5}
        sx={{ flexWrap: 'wrap', rowGap: 1, flexShrink: 0 }}
      >
        <TextField
          label="From"
          type="datetime-local"
          size="small"
          value={fromLocal}
          onChange={(e) => {
            const ms = new Date(e.target.value).getTime();
            if (!Number.isNaN(ms)) {
              actions.setTimeRange(ms, toUnixTimestamp(state.to));
            }
          }}
          InputLabelProps={{ shrink: true }}
          sx={{ width: 200 }}
        />
        <TextField
          label="To"
          type="datetime-local"
          size="small"
          value={toLocal}
          onChange={(e) => {
            const ms = new Date(e.target.value).getTime();
            if (!Number.isNaN(ms)) {
              actions.setTimeRange(toUnixTimestamp(state.from), ms);
            }
          }}
          InputLabelProps={{ shrink: true }}
          sx={{ width: 200 }}
        />
        <FormControl size="small" sx={{ minWidth: 130 }}>
          <InputLabel>Group by</InputLabel>
          <Select
            label="Group by"
            value={state.groupBy}
            onChange={(e) => actions.setGroupBy(e.target.value as QanGroupBy)}
          >
            {GROUP_BY_OPTIONS.map((o) => (
              <MenuItem key={o.value} value={o.value}>
                {o.label}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
        <TextField
          label="Search"
          size="small"
          value={searchDraft}
          onChange={(e) => setSearchDraft(e.target.value)}
          sx={{ minWidth: 160 }}
        />
        <QanManageColumns />
        <Tooltip title="Copy share link">
          <Button
            variant="outlined"
            size="small"
            startIcon={<ContentCopyIcon />}
            onClick={copyLink}
            data-testid="copy-link-button"
          >
            Copy link
          </Button>
        </Tooltip>
      </Stack>
    </Stack>
  );
};
