// eslint-disable-next-line @typescript-eslint/no-explicit-any
export interface UseDetailsPaneNavigationOptions<T extends Record<string, any>> {
  /** Rows in the order the user sees them, typically `navigableRows` from `useNavigableRows`. */
  rows: T[];
  /** Row currently open in the details pane, or undefined when none is. */
  selected: T | undefined;
  /** Returns a stable identifier for a row, used to find the selected one. */
  getRowId: (row: T) => string;
  /** Called with the row to open when the user goes to the previous or next one. */
  onSelect: (row: T) => void;
}

export interface UseDetailsPaneNavigationResult {
  /** Position of the selected row among `rows`, or -1 when nothing is selected or the row is not in the list. */
  index: number;
  /** True when there is no previous row, including when nothing is selected. */
  isFirst: boolean;
  /** True when there is no next row, including when nothing is selected. */
  isLast: boolean;
  /** True when a previous row exists, so a "previous" button can be enabled. */
  hasPrevious: boolean;
  /** True when a next row exists, so a "next" button can be enabled. */
  hasNext: boolean;
  /** Opens the next row. */
  next: () => void;
  /** Opens the previous row. */
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
