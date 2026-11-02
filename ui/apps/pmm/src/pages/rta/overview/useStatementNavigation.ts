import { useState } from 'react';
import { useDetailsPaneNavigation } from '@percona/peak-ui';
import type { QueryData } from 'types/rta.types';
import { statementRowId } from './table/OverviewTable.utils';

interface Options {
  rows: QueryData[];
  selected: QueryData | undefined;
  onSelect: (query: QueryData) => void;
}

interface Result {
  isFirst: boolean;
  isLast: boolean;
  next: () => void;
  previous: () => void;
}

// Once the open statement has finished and left the list, the arrows move from the slot it was
// last seen at, as though it were still there.
export const useStatementNavigation = ({
  rows,
  selected,
  onSelect,
}: Options): Result => {
  const navigation = useDetailsPaneNavigation<QueryData>({
    rows,
    selected,
    getRowId: statementRowId,
    onSelect,
  });
  const [lastIndex, setLastIndex] = useState(-1);

  // Set during render, not in an effect, so the fallback never reads a stale position.
  if (navigation.index >= 0 && navigation.index !== lastIndex) {
    setLastIndex(navigation.index);
  }

  if (!selected || navigation.index >= 0 || lastIndex < 0) {
    return navigation;
  }

  const previousIndex = Math.min(lastIndex, rows.length) - 1;
  const nextIndex = lastIndex;
  const hasPrevious = previousIndex >= 0;
  const hasNext = nextIndex < rows.length;

  return {
    isFirst: !hasPrevious,
    isLast: !hasNext,
    next: () => {
      if (hasNext) {
        onSelect(rows[nextIndex]);
      }
    },
    previous: () => {
      if (hasPrevious) {
        onSelect(rows[previousIndex]);
      }
    },
  };
};
