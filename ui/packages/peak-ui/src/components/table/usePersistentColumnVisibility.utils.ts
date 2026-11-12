import { MRT_VisibilityState } from 'material-react-table';

export const filterHiddenColumns = (localStorageValue: MRT_VisibilityState) => {
  const hiddenColumns: { [key: string]: boolean } = {};
  for (const [key, value] of Object.entries(localStorageValue)) {
    if (!value) {
      hiddenColumns[key] = value;
    }
  }
  return hiddenColumns;
};

export const isObjectEmpty = (obj: object) => Object.keys(obj).length === 0;
