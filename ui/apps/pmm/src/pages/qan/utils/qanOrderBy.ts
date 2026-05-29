import type { MRT_SortingState } from 'material-react-table';

/** qan-api2: `-column` = DESC, `column` = ASC (matches Grafana QAN). */
export function sortingFromOrderBy(orderBy: string): MRT_SortingState {
  const id = orderBy.replace(/^-/, '');
  if (!id) return [];
  return [{ id, desc: orderBy.startsWith('-') }];
}

export function orderByFromSorting(sorting: MRT_SortingState): string {
  const first = sorting[0];
  if (!first) return '';
  return first.desc ? `-${first.id}` : first.id;
}
