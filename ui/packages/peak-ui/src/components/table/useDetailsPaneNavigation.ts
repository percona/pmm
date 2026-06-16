// eslint-disable-next-line @typescript-eslint/no-explicit-any
export interface UseDetailsPaneNavigationOptions<T extends Record<string, any>> {
  rows: T[];
  selected: T | undefined;
  getRowId: (row: T) => string;
  onSelect: (row: T) => void;
}

export interface UseDetailsPaneNavigationResult {
  index: number;
  isFirst: boolean;
  isLast: boolean;
  hasPrevious: boolean;
  hasNext: boolean;
  next: () => void;
  previous: () => void;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function useDetailsPaneNavigation<T extends Record<string, any>>({
  rows,
  selected,
  getRowId,
  onSelect,
}: UseDetailsPaneNavigationOptions<T>): UseDetailsPaneNavigationResult {
  const index = selected ? rows.findIndex((row) => getRowId(row) === getRowId(selected)) : -1;

  const hasPrevious = index > 0;
  const hasNext = index >= 0 && index < rows.length - 1;

  const goTo = (offset: -1 | 1) => {
    if (index < 0) {
      return;
    }
    const nextIndex = index + offset;
    if (nextIndex < 0 || nextIndex >= rows.length) {
      return;
    }
    onSelect(rows[nextIndex]);
  };

  return {
    index,
    isFirst: !hasPrevious,
    isLast: !hasNext,
    hasPrevious,
    hasNext,
    next: () => goTo(1),
    previous: () => goTo(-1),
  };
}

export default useDetailsPaneNavigation;
