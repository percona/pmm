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

// useStatementNavigation drives the details pane's previous and next. While the open statement is
// in the list it is the plain peak-ui navigation. Once it has finished it is no longer in the list,
// and the arrows move from the position it was last seen at: the row now occupying that slot is
// "next" and the one above it "previous", as though the finished row were still between them.
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

  // Recorded during render rather than in an effect, so the fallback below never reads a
  // position one refresh out of date.
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
