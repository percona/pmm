import { MRT_Updater, MRT_VisibilityState } from 'material-react-table';
import { useEffect, useState } from 'react';
import {
  filterHiddenColumns,
  isObjectEmpty,
} from './usePersistentColumnVisibility.utils';

const usePersistentColumnVisibility = (
  key: string
): [
  MRT_VisibilityState,
  (updater: MRT_Updater<MRT_VisibilityState>) => void,
] => {
  const [localStorageValue, setLocalStorageValue] =
    useState<MRT_VisibilityState>(() => {
      try {
        const value = localStorage.getItem(key);
        if (value) {
          return JSON.parse(value);
        }
        return {};
      } catch (error) {
        return {};
      }
    });

  const setLocalStorageStateValue = (
    updater: MRT_Updater<MRT_VisibilityState>
  ) => {
    setLocalStorageValue((prevValues: MRT_VisibilityState) =>
      updater instanceof Function ? updater(prevValues) : updater
    );
  };

  useEffect(() => {
    if (!isObjectEmpty(localStorageValue)) {
      const hiddenColumns = filterHiddenColumns(localStorageValue);

      if (!isObjectEmpty(hiddenColumns)) {
        localStorage.setItem(key, JSON.stringify(hiddenColumns));
      } else {
        localStorage.removeItem(key);
      }
    }
  }, [localStorageValue]);

  return [localStorageValue, setLocalStorageStateValue];
};

export default usePersistentColumnVisibility;
