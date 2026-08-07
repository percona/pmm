// react-table's plugin hooks (e.g. useRowSelect) add properties to TableInstance that aren't part of
// its base type. This augmentation merges them in, following react-table's own documented pattern:
// https://github.com/TanStack/table/blob/v7/docs/typescript.md#customizing-cell-and-column-search
import { UseRowSelectInstanceProps } from 'react-table';

declare module 'react-table' {
  // eslint-disable-next-line @typescript-eslint/no-empty-interface
  export interface TableInstance<D extends object = {}> extends UseRowSelectInstanceProps<D> {}
}
