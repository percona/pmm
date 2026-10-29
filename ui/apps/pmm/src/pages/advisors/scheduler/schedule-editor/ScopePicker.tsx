import Box from '@mui/material/Box';
import Checkbox from '@mui/material/Checkbox';
import Chip from '@mui/material/Chip';
import FormControlLabel from '@mui/material/FormControlLabel';
import Link from '@mui/material/Link';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { ADVISOR_TECHNOLOGY } from 'lib/constants';
import { type FC, useMemo, useState } from 'react';
import { type AdvisorTechnology } from 'types/advisors.types';
import { SCHEDULE_TECHNOLOGIES } from '../AdvisorScheduler.utils';
import { TechnologyLabel } from '../AdvisorScheduler.components';
import { Messages } from './ScheduleEditor.messages';

export interface PickerItem {
  id: string;
  label: string;
  technology: AdvisorTechnology;
  // the value the group filter matches: category or environment
  group: string;
  meta: string[];
  // text the search matches, lowercase
  search: string;
  // a note shown as a chip, e.g. a check disabled in the Catalog
  warning?: { label: string; tooltip: string };
}

interface ScopePickerProps {
  items: PickerItem[];
  selected: string[];
  onChange: (selected: string[]) => void;
  searchLabel: string;
  groupLabel: string;
  allGroupsLabel: string;
  noMatch: string;
  // offers "Show selected only"
  canShowSelectedOnly?: boolean;
  testId: string;
}

export const ScopePicker: FC<ScopePickerProps> = ({
  items,
  selected,
  onChange,
  searchLabel,
  groupLabel,
  allGroupsLabel,
  noMatch,
  canShowSelectedOnly,
  testId,
}) => {
  const [query, setQuery] = useState('');
  const [technologies, setTechnologies] = useState<AdvisorTechnology[]>([]);
  const [group, setGroup] = useState('');
  const [onlySelected, setOnlySelected] = useState(false);

  const groups = useMemo(
    () => [...new Set(items.map((i) => i.group).filter(Boolean))].sort(),
    [items]
  );
  const term = query.trim().toLowerCase();
  const shown = items.filter(
    (item) =>
      (!term || item.search.includes(term)) &&
      (!technologies.length || technologies.includes(item.technology)) &&
      (!group || item.group === group) &&
      (!onlySelected || selected.includes(item.id))
  );
  // picks that no longer exist do not count
  const selectedCount = items.filter((i) => selected.includes(i.id)).length;

  const toggle = (id: string, on: boolean) =>
    onChange(on ? [...selected, id] : selected.filter((s) => s !== id));

  return (
    <Box
      sx={{
        border: 1,
        borderColor: 'divider',
        borderRadius: 1.5,
        overflow: 'hidden',
      }}
      data-testid={testId}
    >
      <Stack
        direction="row"
        flexWrap="wrap"
        gap={1}
        alignItems="center"
        sx={{ p: 1.25, borderBottom: 1, borderColor: 'divider' }}
      >
        <TextField
          size="small"
          type="search"
          placeholder={searchLabel}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          slotProps={{
            htmlInput: {
              'aria-label': searchLabel,
              'data-testid': `${testId}-search`,
            },
          }}
          sx={{ flex: '1 1 200px' }}
        />
        {SCHEDULE_TECHNOLOGIES.map((technology) => {
          const on = technologies.includes(technology);
          return (
            <Chip
              key={technology}
              size="small"
              variant={on ? 'filled' : 'outlined'}
              label={<TechnologyLabel technology={technology} />}
              aria-pressed={on}
              onClick={() =>
                setTechnologies(
                  on
                    ? technologies.filter((t) => t !== technology)
                    : [...technologies, technology]
                )
              }
              data-testid={`${testId}-tech-${ADVISOR_TECHNOLOGY[technology]}`}
            />
          );
        })}
        <TextField
          select
          size="small"
          // deterministic id: React's useId default breaks jsdom selector matching
          id={`${testId}-group`}
          value={group || allGroupsLabel}
          onChange={(e) =>
            setGroup(e.target.value === allGroupsLabel ? '' : e.target.value)
          }
          slotProps={{ htmlInput: { 'aria-label': groupLabel } }}
          sx={{ minWidth: 160 }}
        >
          <MenuItem value={allGroupsLabel}>{allGroupsLabel}</MenuItem>
          {groups.map((g) => (
            <MenuItem key={g} value={g}>
              {g}
            </MenuItem>
          ))}
        </TextField>
      </Stack>
      <Stack
        direction="row"
        flexWrap="wrap"
        alignItems="center"
        columnGap={2}
        rowGap={0.5}
        sx={{ px: 1.5, py: 0.75, borderBottom: 1, borderColor: 'divider' }}
      >
        <Link
          component="button"
          type="button"
          variant="body2"
          disabled={!shown.length}
          onClick={() =>
            onChange([
              ...selected,
              ...shown.map((i) => i.id).filter((id) => !selected.includes(id)),
            ])
          }
          data-testid={`${testId}-select-shown`}
        >
          {Messages.picker.selectShown(shown.length)}
        </Link>
        <Link
          component="button"
          type="button"
          variant="body2"
          onClick={() => onChange([])}
          data-testid={`${testId}-clear`}
        >
          {Messages.picker.clear}
        </Link>
        {canShowSelectedOnly && (
          <FormControlLabel
            control={
              <Checkbox
                size="small"
                checked={onlySelected}
                onChange={(e) => setOnlySelected(e.target.checked)}
              />
            }
            label={
              <Typography variant="body2">
                {Messages.picker.onlySelected}
              </Typography>
            }
          />
        )}
        <Typography
          variant="body2"
          color="text.secondary"
          sx={{ ml: 'auto' }}
          data-testid={`${testId}-count`}
        >
          {Messages.picker.count(selectedCount, items.length)}
        </Typography>
      </Stack>
      <Box sx={{ maxHeight: 280, overflowY: 'auto' }}>
        {shown.length ? (
          shown.map((item) => (
            <Stack
              key={item.id}
              component="label"
              direction="row"
              alignItems="center"
              gap={1}
              sx={{
                px: 1,
                py: 0.25,
                cursor: 'pointer',
                '&:hover': { bgcolor: 'action.hover' },
              }}
            >
              <Checkbox
                size="small"
                checked={selected.includes(item.id)}
                onChange={(e) => toggle(item.id, e.target.checked)}
                slotProps={{
                  input: {
                    // @ts-expect-error data-testid is passed through to the DOM
                    'data-testid': `${testId}-item-${item.id}`,
                  },
                }}
              />
              <Typography variant="body2" sx={{ flex: 1, minWidth: 0 }}>
                {item.label}
              </Typography>
              <Stack
                direction="row"
                gap={1.25}
                alignItems="center"
                sx={{ fontSize: 12, color: 'text.secondary', flex: 'none' }}
              >
                {item.warning && (
                  <Tooltip title={item.warning.tooltip} arrow>
                    <Chip
                      size="small"
                      color="warning"
                      variant="outlined"
                      label={item.warning.label}
                    />
                  </Tooltip>
                )}
                {item.meta.filter(Boolean).map((m, i) => (
                  <span key={i}>{m}</span>
                ))}
                <TechnologyLabel technology={item.technology} />
              </Stack>
            </Stack>
          ))
        ) : (
          <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
            {noMatch}
          </Typography>
        )}
      </Box>
    </Box>
  );
};
