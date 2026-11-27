import ArrowDropDownIcon from '@mui/icons-material/ArrowDropDown';
import RefreshIcon from '@mui/icons-material/Refresh';
import SearchIcon from '@mui/icons-material/Search';
import {
  Autocomplete,
  Box,
  Button,
  Chip,
  FormControlLabel,
  IconButton,
  InputAdornment,
  Menu,
  MenuItem,
  PopperProps,
  Popover,
  Stack,
  Switch,
  TextField,
  ToggleButton,
  ToggleButtonGroup,
  Tooltip,
} from '@mui/material';
import { FC, MouseEvent, ReactNode, useEffect, useState } from 'react';
import { LogFacet, LogFacets, LogFilter, SeverityClass } from 'api/logs';
import { toDatetimeLocalValue } from 'pages/qan/utils/qanTools';
import { MONO } from '../logsTheme';
import {
  hasFilters,
  PMM_SERVER_NODE,
  QUICK_RANGES,
  QuickRange,
  SEVERITY_OPTIONS,
  severityClassOf,
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
const CONTROL_HEIGHT = 36;

const severityOptionLabel = (value: number) => {
  const option = SEVERITY_OPTIONS.find((o) => o.value === value);
  return !option || option.value === 0 ? 'Any' : `${option.label} and above`;
};

const formatRange = (start: string, end: string) => {
  const [from, to] = [new Date(start), new Date(end)];
  const day = (d: Date) =>
    d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
  const time = (d: Date) =>
    d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
  const toText = day(from) === day(to) ? time(to) : `${day(to)}, ${time(to)}`;
  return `${day(from)}, ${time(from)} → ${toText}`;
};

interface FilterButtonProps {
  label: string;
  value: ReactNode;
  active: boolean;
  onClick: (event: MouseEvent<HTMLElement>) => void;
  testId?: string;
}

/** An outlined button naming the filter and its current value. */
const FilterButton: FC<FilterButtonProps> = ({
  label,
  value,
  active,
  onClick,
  testId,
}) => (
  <Button
    variant="outlined"
    size="small"
    color={active ? 'primary' : 'inherit'}
    endIcon={<ArrowDropDownIcon />}
    onClick={onClick}
    data-testid={testId}
    sx={{
      height: CONTROL_HEIGHT,
      textTransform: 'none',
      borderColor: active ? undefined : 'divider',
      bgcolor: active ? 'primary.hover' : 'background.paper',
    }}
  >
    <Box component="span" sx={{ color: 'text.secondary', mr: 0.75 }}>
      {label}
    </Box>{' '}
    {value}
  </Button>
);

// Shows the options inside the popover instead of a second popper over it.
const InlinePopper = ({
  className,
  children,
}: Pick<PopperProps, 'className' | 'children'>) => (
  <Box className={className}>
    {typeof children === 'function' ? null : children}
  </Box>
);

interface FacetFilterProps {
  label: string;
  facets?: LogFacet[];
  value?: string[];
  onChange: (value?: string[]) => void;
  testId: string;
}

const FacetFilter: FC<FacetFilterProps> = ({
  label,
  facets = [],
  value = [],
  onChange,
  testId,
}) => {
  const [anchor, setAnchor] = useState<HTMLElement | null>(null);
  const counts = new Map(facets.map((f) => [f.value, f.count]));
  const options = [
    ...facets.map((f) => f.value),
    ...value.filter((v) => !counts.has(v)),
  ];
  const summary =
    value.length === 0
      ? 'Any'
      : value.length === 1
        ? value[0]
        : `${value[0]} +${value.length - 1}`;

  return (
    <>
      <FilterButton
        label={label}
        value={summary}
        active={value.length > 0}
        onClick={(e) => setAnchor(e.currentTarget)}
        testId={testId}
      />
      <Popover
        open={!!anchor}
        anchorEl={anchor}
        onClose={() => setAnchor(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
        slotProps={{ paper: { sx: { mt: 0.5, p: 1, width: 320 } } }}
      >
        <Autocomplete
          multiple
          open
          size="small"
          options={options}
          value={value}
          onChange={(_, v) => onChange(v.length ? v : undefined)}
          renderOption={(props, option) => (
            <li {...props} key={option}>
              {option}
              {counts.has(option) &&
                ` (${counts.get(option)!.toLocaleString()})`}
            </li>
          )}
          renderInput={(params) => (
            <TextField
              {...params}
              autoFocus
              placeholder={`Filter by ${label.toLowerCase()}`}
            />
          )}
          slots={{ popper: InlinePopper }}
          slotProps={{ paper: { elevation: 0 } }}
          noOptionsText="No values in this range"
        />
      </Popover>
    </>
  );
};

// Service labels, shown when the inventory has them; they filter lines by the nodes of the labeled services.
const LABELS = [
  {
    key: 'environments',
    label: 'Environment',
    testId: 'logs-filter-environment',
  },
  { key: 'clusters', label: 'Cluster', testId: 'logs-filter-cluster' },
  {
    key: 'replicationSets',
    label: 'Replication set',
    testId: 'logs-filter-replication-set',
  },
] as const;

// What the Severity button says for a filter that selects one class.
const CLASS_LABELS: Record<SeverityClass, string> = {
  error: 'Error and above',
  warning: 'Warning only',
  info: 'Info only',
  other: 'Other only',
};

const severityFilterLabel = (f: LogFilter) => {
  const cls = severityClassOf(f);
  if (cls) return CLASS_LABELS[cls];
  const min = severityOptionLabel(f.minSeverity ?? 0);
  return f.maxSeverity ? `${min}, up to ${f.maxSeverity}` : min;
};

const SeverityFilter: FC<{
  filter: LogFilter;
  onChange: (minSeverity?: number) => void;
}> = ({ filter, onChange }) => {
  const [anchor, setAnchor] = useState<HTMLElement | null>(null);
  const value = filter.maxSeverity ? undefined : (filter.minSeverity ?? 0);
  return (
    <>
      <FilterButton
        label="Severity"
        value={severityFilterLabel(filter)}
        active={!!(filter.minSeverity || filter.maxSeverity)}
        onClick={(e) => setAnchor(e.currentTarget)}
        testId="logs-filter-severity"
      />
      <Menu open={!!anchor} anchorEl={anchor} onClose={() => setAnchor(null)}>
        {SEVERITY_OPTIONS.map((o) => (
          <MenuItem
            key={o.value}
            selected={o.value === value}
            onClick={() => {
              setAnchor(null);
              onChange(o.value || undefined);
            }}
          >
            {severityOptionLabel(o.value)}
          </MenuItem>
        ))}
      </Menu>
    </>
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
  const [timeAnchor, setTimeAnchor] = useState<HTMLElement | null>(null);
  const hidesServer = !!filter.excludeNodeNames?.includes(PMM_SERVER_NODE);
  // Collectors without a service.name get the node name, which the Node filter already offers.
  const nodeNames = new Set(facets?.nodeNames.map((f) => f.value));
  const services = (facets?.sources ?? []).filter(
    (f) => !nodeNames.has(f.value)
  );

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
    <Stack spacing={1.5} data-testid="logs-filter-bar">
      <Stack
        direction="row"
        spacing={1}
        alignItems="center"
        sx={{ flexWrap: 'wrap', rowGap: 1 }}
      >
        <TextField
          size="small"
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder="Search log lines, any case"
          sx={{
            flex: 1,
            minWidth: 280,
            '& .MuiInputBase-root': {
              height: CONTROL_HEIGHT,
              bgcolor: 'background.paper',
            },
            '& input': MONO,
          }}
          slotProps={{
            input: {
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon fontSize="small" />
                </InputAdornment>
              ),
            },
            htmlInput: { 'aria-label': 'Search' },
          }}
          data-testid="logs-filter-text"
        />
        <ToggleButtonGroup
          exclusive
          size="small"
          value={range ?? null}
          onChange={(_, value: QuickRange | null) =>
            value && onChange(filter, value)
          }
          aria-label="Time range"
          sx={{ bgcolor: 'background.paper' }}
        >
          {QUICK_RANGES.map((r) => (
            <ToggleButton
              key={r}
              value={r}
              aria-label={RANGE_LABELS[r]}
              sx={{ height: CONTROL_HEIGHT, px: 1.5, textTransform: 'none' }}
            >
              {r}
            </ToggleButton>
          ))}
        </ToggleButtonGroup>
        <Button
          variant="outlined"
          size="small"
          color="inherit"
          onClick={(e) => setTimeAnchor(e.currentTarget)}
          data-testid="logs-time-button"
          sx={{
            height: CONTROL_HEIGHT,
            textTransform: 'none',
            borderColor: 'divider',
            bgcolor: 'background.paper',
            fontVariantNumeric: 'tabular-nums',
          }}
        >
          {formatRange(filter.start, filter.end)}
        </Button>
        <Popover
          open={!!timeAnchor}
          anchorEl={timeAnchor}
          onClose={() => setTimeAnchor(null)}
          anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
          transformOrigin={{ vertical: 'top', horizontal: 'right' }}
          slotProps={{ paper: { sx: { mt: 0.5, p: 2 } } }}
        >
          <Stack spacing={2}>
            <TextField
              label="From"
              type="datetime-local"
              size="small"
              value={toDatetimeLocalValue(filter.start)}
              onChange={(e) => setTime('start', e.target.value)}
              slotProps={{ inputLabel: { shrink: true } }}
            />
            <TextField
              label="To"
              type="datetime-local"
              size="small"
              value={toDatetimeLocalValue(filter.end)}
              onChange={(e) => setTime('end', e.target.value)}
              slotProps={{ inputLabel: { shrink: true } }}
            />
          </Stack>
        </Popover>
        <Tooltip title="Refresh">
          <IconButton onClick={onRefresh} aria-label="Refresh">
            <RefreshIcon />
          </IconButton>
        </Tooltip>
      </Stack>
      <Stack
        direction="row"
        spacing={1}
        alignItems="center"
        sx={{ flexWrap: 'wrap', rowGap: 1 }}
      >
        {LABELS.map(({ key, label, testId }) =>
          facets?.[key].length || filter[key] ? (
            <FacetFilter
              key={key}
              label={label}
              facets={facets?.[key]}
              value={filter[key]}
              onChange={(value) => onChange({ ...filter, [key]: value }, range)}
              testId={testId}
            />
          ) : null
        )}
        <FacetFilter
          label="Node"
          facets={facets?.nodeNames}
          value={filter.nodeNames}
          onChange={(nodeNames) => onChange({ ...filter, nodeNames }, range)}
          testId="logs-filter-node"
        />
        {(services.length > 0 || !!filter.sources) && (
          <FacetFilter
            label="Service"
            facets={services}
            value={filter.sources}
            onChange={(sources) => onChange({ ...filter, sources }, range)}
            testId="logs-filter-source"
          />
        )}
        <FacetFilter
          label="Log type"
          facets={facets?.presets}
          value={filter.presets}
          onChange={(presets) => onChange({ ...filter, presets }, range)}
          testId="logs-filter-preset"
        />
        <SeverityFilter
          filter={filter}
          onChange={(minSeverity) =>
            onChange({ ...filter, minSeverity, maxSeverity: undefined }, range)
          }
        />
        <FormControlLabel
          control={
            <Switch
              size="small"
              checked={hidesServer}
              onChange={(e) =>
                onChange(
                  {
                    ...filter,
                    excludeNodeNames: e.target.checked
                      ? [PMM_SERVER_NODE]
                      : undefined,
                  },
                  range
                )
              }
            />
          }
          label="Hide PMM Server"
          slotProps={{ typography: { variant: 'body2' } }}
          sx={{ ml: 0.5, color: 'text.secondary' }}
        />
        {hasFilters(filter) && (
          <Button
            size="small"
            onClick={() =>
              onChange(
                {
                  start: filter.start,
                  end: filter.end,
                  excludeNodeNames: filter.excludeNodeNames,
                },
                range
              )
            }
            sx={{ height: CONTROL_HEIGHT, textTransform: 'none' }}
          >
            Clear filters
          </Button>
        )}
        {filter.pattern && (
          <Chip
            label={`Pattern: ${filter.pattern}`}
            onDelete={() => onChange({ ...filter, pattern: undefined }, range)}
            sx={{ maxWidth: 480, ...MONO }}
            data-testid="logs-filter-pattern"
          />
        )}
        {!!filter.excludePatternHashes?.length && (
          <Chip
            label={`${filter.excludePatternHashes.length} hidden ${
              filter.excludePatternHashes.length === 1 ? 'pattern' : 'patterns'
            }`}
            onDelete={() =>
              onChange({ ...filter, excludePatternHashes: undefined }, range)
            }
            title="Show the hidden patterns again"
            data-testid="logs-filter-hidden-patterns"
          />
        )}
      </Stack>
    </Stack>
  );
};
