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

import { useId, useMemo } from 'react';
import Box from '@mui/material/Box';
import Checkbox from '@mui/material/Checkbox';
import Chip from '@mui/material/Chip';
import FormControl from '@mui/material/FormControl';
import FormControlLabel from '@mui/material/FormControlLabel';
import IconButton from '@mui/material/IconButton';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select, { type SelectChangeEvent } from '@mui/material/Select';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import CloseIcon from '@mui/icons-material/Close';
import DragIndicatorIcon from '@mui/icons-material/DragIndicator';
import {
  DndContext,
  KeyboardSensor,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors,
  type DragEndEvent,
} from '@dnd-kit/core';
import {
  SortableContext,
  arrayMove,
  rectSortingStrategy,
  sortableKeyboardCoordinates,
  useSortable,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';

export interface ChainValue {
  chain_task_names: string[];
  chain_on_failure: boolean;
}

export interface AvailableTask {
  name: string;
}

export interface ChainBuilderProps {
  availableTasks: AvailableTask[];
  currentTaskName: string;
  value: ChainValue;
  onChange: (next: ChainValue) => void;
  label?: string;
  disabled?: boolean;
}

const DEFAULT_LABEL = 'Chain tasks after execution (optional)';
const FAILURE_TITLE =
  'When enabled, the chain continues even if a task fails, stops, or is lost';

// Compose dnd-kit ids that stay unique even if the chain contains duplicate
// names (e.g. stale data). The leading index is the source of truth for
// removal and reorder; the name suffix is just for debuggability.
function chipId(index: number, name: string): string {
  return `${index}:${name}`;
}

function indexFromChipId(id: string): number {
  const colon = id.indexOf(':');
  return colon === -1 ? -1 : Number.parseInt(id.slice(0, colon), 10);
}

function wouldCreateCycle(
  name: string,
  currentTaskName: string,
  chain: readonly string[]
): boolean {
  return name === currentTaskName || chain.includes(name);
}

export function ChainBuilder({
  availableTasks,
  currentTaskName,
  value,
  onChange,
  label = DEFAULT_LABEL,
  disabled = false,
}: ChainBuilderProps) {
  const { chain_task_names: chain, chain_on_failure: chainOnFailure } = value;
  const groupLabelId = useId();
  const selectId = useId();
  const helperId = useId();

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates })
  );

  const disabledOptions = useMemo(() => {
    const set = new Set<string>(chain);
    set.add(currentTaskName);
    return set;
  }, [chain, currentTaskName]);

  const hasSelectableTask = useMemo(
    () => availableTasks.some((t) => !disabledOptions.has(t.name)),
    [availableTasks, disabledOptions]
  );

  const itemIds = useMemo(
    () => chain.map((name, i) => chipId(i, name)),
    [chain]
  );

  const handleAdd = (event: SelectChangeEvent<string>) => {
    const name = event.target.value;
    if (!name || wouldCreateCycle(name, currentTaskName, chain)) {
      return;
    }
    onChange({ ...value, chain_task_names: [...chain, name] });
  };

  const handleRemoveAt = (index: number) => {
    if (index < 0 || index >= chain.length) {
      return;
    }
    const next = chain.slice();
    next.splice(index, 1);
    onChange({ ...value, chain_task_names: next });
  };

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) {
      return;
    }
    const from = indexFromChipId(String(active.id));
    const to = indexFromChipId(String(over.id));
    if (from < 0 || to < 0 || from === to) {
      return;
    }
    onChange({
      ...value,
      chain_task_names: arrayMove(chain.slice(), from, to),
    });
  };

  const handleFailureToggle = (_: unknown, checked: boolean) => {
    onChange({ ...value, chain_on_failure: checked });
  };

  const hasChain = chain.length > 0;
  const selectDisabled = disabled || !hasSelectableTask;
  const selectHelper = !hasSelectableTask
    ? 'No tasks available to chain'
    : undefined;

  return (
    <Box
      role="group"
      aria-labelledby={groupLabelId}
      data-testid="chain-builder"
    >
      <Typography
        id={groupLabelId}
        component="div"
        variant="body2"
        sx={{ display: 'block', mb: 1 }}
      >
        {label}
      </Typography>

      {hasChain && (
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragEnd={handleDragEnd}
        >
          <SortableContext items={itemIds} strategy={rectSortingStrategy}>
            <Box
              data-testid="chain-sequence"
              sx={{
                display: 'flex',
                flexWrap: 'wrap',
                alignItems: 'center',
                gap: 0.5,
                mb: 1.5,
              }}
            >
              {chain.map((name, index) => (
                <SortableChainItem
                  key={itemIds[index]}
                  id={itemIds[index]}
                  name={name}
                  index={index}
                  isLast={index === chain.length - 1}
                  disabled={disabled}
                  onRemove={handleRemoveAt}
                />
              ))}
            </Box>
          </SortableContext>
        </DndContext>
      )}

      <FormControl fullWidth size="small" disabled={selectDisabled}>
        <InputLabel id={`${selectId}-label`}>Add task to chain…</InputLabel>
        <Select
          labelId={`${selectId}-label`}
          id={selectId}
          label="Add task to chain…"
          value=""
          displayEmpty={false}
          onChange={handleAdd}
          inputProps={
            {
              'data-testid': 'chain-add-select',
              ...(selectHelper ? { 'aria-describedby': helperId } : {}),
            } as React.InputHTMLAttributes<HTMLInputElement>
          }
        >
          {availableTasks.map((t) => (
            <MenuItem
              key={t.name}
              value={t.name}
              disabled={disabledOptions.has(t.name)}
            >
              {t.name}
            </MenuItem>
          ))}
        </Select>
        {selectHelper && (
          <Typography
            id={helperId}
            variant="caption"
            color="text.secondary"
            sx={{ mt: 0.5 }}
          >
            {selectHelper}
          </Typography>
        )}
      </FormControl>

      {hasChain && (
        <Tooltip title={FAILURE_TITLE} placement="top-start">
          <FormControlLabel
            sx={{ mt: 1 }}
            control={
              <Checkbox
                checked={chainOnFailure}
                disabled={disabled}
                onChange={handleFailureToggle}
                inputProps={
                  {
                    'data-testid': 'chain-on-failure-checkbox',
                  } as React.InputHTMLAttributes<HTMLInputElement>
                }
              />
            }
            label="Continue chain on failure"
          />
        </Tooltip>
      )}
    </Box>
  );
}

interface SortableChainItemProps {
  id: string;
  name: string;
  index: number;
  isLast: boolean;
  disabled: boolean;
  onRemove: (index: number) => void;
}

function SortableChainItem({
  id,
  name,
  index,
  isLast,
  disabled,
  onRemove,
}: SortableChainItemProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({
    id,
    disabled,
  });

  const style = {
    transform: CSS.Translate.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  return (
    <Box
      ref={setNodeRef}
      style={style}
      component="span"
      sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5 }}
    >
      <Box
        sx={{
          display: 'inline-flex',
          alignItems: 'center',
          border: 1,
          borderColor: 'divider',
          borderRadius: 16,
          pl: 0.25,
          pr: 0.25,
        }}
      >
        <IconButton
          size="small"
          aria-label={`Drag ${name}`}
          disabled={disabled}
          sx={{ cursor: disabled ? 'not-allowed' : 'grab', p: 0.25 }}
          {...attributes}
          {...listeners}
        >
          <DragIndicatorIcon fontSize="small" />
        </IconButton>
        <Chip
          component="span"
          size="small"
          variant="outlined"
          label={name}
          sx={{ border: 'none' }}
        />
        <IconButton
          size="small"
          aria-label={`Remove ${name}`}
          disabled={disabled}
          onClick={() => onRemove(index)}
          sx={{ p: 0.25 }}
        >
          <CloseIcon fontSize="small" />
        </IconButton>
      </Box>
      {!isLast && (
        <Typography component="span" variant="body2" aria-hidden>
          →
        </Typography>
      )}
    </Box>
  );
}
