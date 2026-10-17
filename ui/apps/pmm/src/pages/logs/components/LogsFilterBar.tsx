import RefreshIcon from '@mui/icons-material/Refresh';
import {
  Autocomplete,
  Chip,
  FormControl,
  IconButton,
  InputLabel,
  MenuItem,
  Select,
  Stack,
  TextField,
  Tooltip,
} from '@mui/material';
import { FC, useEffect, useState } from 'react';
import { LogFacet, LogFacets, LogFilter } from 'api/logs';
import { toDatetimeLocalValue } from 'pages/qan/utils/qanTools';
import {
  QUICK_RANGES,
  QuickRange,
  SEVERITY_OPTIONS,
} from '../utils/logsFilter';

interface LogsFilterBarProps {
  filter: LogFilter;
  range?: QuickRange;
  facets?: LogFacets;
  onChange: (filter: LogFilter, range?: QuickRange) => void;
  onRefresh: () => void;
}

const RANGE_LABELS: Record<QuickRange, string> = {
  '15m': 'Last 15 minutes',
  '1h': 'Last hour',
  '6h': 'Last 6 hours',
  '24h': 'Last 24 hours',
  '7d': 'Last 7 days',
};

const SEARCH_DEBOUNCE_MS = 400;

interface FacetPickerProps {
  label: string;
  facets?: LogFacet[];
  value?: string[];
  onChange: (value?: string[]) => void;
  testId: string;
}

const FacetPicker: FC<FacetPickerProps> = ({
  label,
  facets = [],
  value = [],
  onChange,
  testId,
}) => {
  const counts = new Map(facets.map((f) => [f.value, f.count]));
  const options = [
    ...facets.map((f) => f.value),
    ...value.filter((v) => !counts.has(v)),
  ];
  return (
    <Autocomplete
      multiple
      size="small"
      limitTags={2}
      options={options}
      value={value}
      onChange={(_, v) => onChange(v.length ? v : undefined)}
      renderOption={(props, option) => (
        <li {...props} key={option}>
          {option}
          {counts.has(option) && ` (${counts.get(option)!.toLocaleString()})`}
        </li>
      )}
      renderInput={(params) => <TextField {...params} label={label} />}
      sx={{ minWidth: 200 }}
      data-testid={testId}
    />
  );
};

export const LogsFilterBar: FC<LogsFilterBarProps> = ({
  filter,
  range,
  facets,
  onChange,
  onRefresh,
}) => {
  const [text, setText] = useState(filter.text ?? '');

  useEffect(() => setText(filter.text ?? ''), [filter.text]);

  useEffect(() => {
    if (text === (filter.text ?? '')) return;
    const timer = setTimeout(
      () => onChange({ ...filter, text: text || undefined }, range),
      SEARCH_DEBOUNCE_MS
    );
    return () => clearTimeout(timer);
  }, [text, filter, range, onChange]);

  const setTime = (key: 'start' | 'end', local: string) => {
    const ms = new Date(local).getTime();
    if (!Number.isNaN(ms)) {
      onChange({ ...filter, [key]: new Date(ms).toISOString() });
    }
  };

  return (
    <Stack
      direction="row"
      spacing={1.5}
      alignItems="center"
      sx={{ flexWrap: 'wrap', rowGap: 1.5 }}
      data-testid="logs-filter-bar"
    >
      <FormControl size="small" sx={{ minWidth: 160 }}>
        <InputLabel id="logs-range-label">Time range</InputLabel>
        <Select
          labelId="logs-range-label"
          label="Time range"
          value={range ?? 'custom'}
          onChange={(e) => {
            const value = e.target.value as QuickRange | 'custom';
            if (value !== 'custom') onChange(filter, value);
          }}
        >
          {QUICK_RANGES.map((r) => (
            <MenuItem key={r} value={r}>
              {RANGE_LABELS[r]}
            </MenuItem>
          ))}
          <MenuItem value="custom" disabled>
            Custom
          </MenuItem>
        </Select>
      </FormControl>
      <TextField
        label="From"
        type="datetime-local"
        size="small"
        value={toDatetimeLocalValue(filter.start)}
        onChange={(e) => setTime('start', e.target.value)}
        InputLabelProps={{ shrink: true }}
        sx={{ width: 200 }}
      />
      <TextField
        label="To"
        type="datetime-local"
        size="small"
        value={toDatetimeLocalValue(filter.end)}
        onChange={(e) => setTime('end', e.target.value)}
        InputLabelProps={{ shrink: true }}
        sx={{ width: 200 }}
      />
      <Tooltip title="Refresh">
        <IconButton onClick={onRefresh} aria-label="Refresh">
          <RefreshIcon />
        </IconButton>
      </Tooltip>
      <FacetPicker
        label="Node"
        facets={facets?.nodeNames}
        value={filter.nodeNames}
        onChange={(nodeNames) => onChange({ ...filter, nodeNames }, range)}
        testId="logs-filter-node"
      />
      <FacetPicker
        label="Source"
        facets={facets?.sources}
        value={filter.sources}
        onChange={(sources) => onChange({ ...filter, sources }, range)}
        testId="logs-filter-source"
      />
      <FacetPicker
        label="Log type"
        facets={facets?.presets}
        value={filter.presets}
        onChange={(presets) => onChange({ ...filter, presets }, range)}
        testId="logs-filter-preset"
      />
      <FormControl size="small" sx={{ minWidth: 140 }}>
        <InputLabel id="logs-severity-label">Minimum severity</InputLabel>
        <Select
          labelId="logs-severity-label"
          label="Minimum severity"
          value={filter.minSeverity ?? 0}
          onChange={(e) => {
            const minSeverity = Number(e.target.value);
            onChange(
              { ...filter, minSeverity: minSeverity || undefined },
              range
            );
          }}
        >
          {SEVERITY_OPTIONS.map((o) => (
            <MenuItem key={o.value} value={o.value}>
              {o.label}
            </MenuItem>
          ))}
        </Select>
      </FormControl>
      <TextField
        label="Search"
        size="small"
        value={text}
        onChange={(e) => setText(e.target.value)}
        placeholder="Text in the line, any case"
        sx={{ minWidth: 240, flex: 1 }}
        data-testid="logs-filter-text"
      />
      {filter.pattern && (
        <Chip
          label={`Pattern: ${filter.pattern}`}
          onDelete={() => onChange({ ...filter, pattern: undefined }, range)}
          sx={{ maxWidth: 480 }}
          data-testid="logs-filter-pattern"
        />
      )}
    </Stack>
  );
};
