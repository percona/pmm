import {
  type ChangeEvent,
  type FC,
  useCallback,
  useEffect,
  useRef,
  useState,
} from 'react';
import TextField from '@mui/material/TextField';
import type { MRT_Column } from 'material-react-table';
import type { QueryData } from 'types/rta.types';
import {
  isElapsedTimeBoundInput,
  toElapsedTimeBound,
} from '../OverviewTable.utils';
import { Messages } from './ElapsedTimeFilter.messages';
import { readRange } from './ElapsedTimeFilter.utils';

export interface Props {
  column: MRT_Column<QueryData>;
  rangeFilterIndex?: number;
}

// Replaces the range field material-react-table renders for the Elapsed time
// column. MRT keeps the typed text in its own state, so a rejected character
// can only be kept out of the box by owning that state here. The variant, size
// and sx mirror the muiFilterTextFieldProps peak-ui's Table applies to every
// other filter, which MRT skips for a column that supplies its own Filter.
const ElapsedTimeFilter: FC<Props> = ({ column, rangeFilterIndex = 0 }) => {
  const label = rangeFilterIndex === 1 ? Messages.max : Messages.min;
  const committed = readRange(column.getFilterValue())[rangeFilterIndex];
  const [value, setValue] = useState(() => toElapsedTimeBound(committed));
  const lastCommitted = useRef<string | undefined>(undefined);

  const commit = useCallback(
    (next: string) => {
      lastCommitted.current = next;
      column.setFilterValue((current: unknown) => {
        const range = readRange(current);
        range[rangeFilterIndex] = next;
        return range;
      });
    },
    [column, rangeFilterIndex]
  );

  useEffect(() => {
    if (committed === lastCommitted.current) {
      return;
    }
    lastCommitted.current = committed;
    const bound = toElapsedTimeBound(committed);
    setValue(bound);
    if (bound !== committed) {
      // A hand-edited or bookmarked URL can carry a bound that filters nothing
      // yet still marks the column as filtered.
      commit(bound);
    }
  }, [committed, commit]);

  const handleChange = (event: ChangeEvent<HTMLInputElement>) => {
    const next = event.target.value;
    if (!isElapsedTimeBoundInput(next)) {
      return;
    }
    setValue(next);
    commit(toElapsedTimeBound(next));
  };

  return (
    <TextField
      margin="none"
      onChange={handleChange}
      onClick={(event) => event.stopPropagation()}
      onKeyDown={(event) => event.stopPropagation()}
      placeholder={label}
      size="small"
      slotProps={{
        htmlInput: {
          'aria-label': label,
          autoComplete: 'off',
          inputMode: 'decimal',
          sx: { textOverflow: 'ellipsis' },
          title: label,
        },
      }}
      sx={{ minWidth: 0, mx: 0, width: '100%' }}
      value={value}
      variant="outlined"
    />
  );
};

export default ElapsedTimeFilter;
