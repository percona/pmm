import { MRT_Updater, MRT_VisibilityState } from 'material-react-table';
import { useEffect, useState } from 'react';
import {
  filterHiddenColumns,
  isObjectEmpty,
} from './usePersistentColumnVisibility.utils';

const readVisibility = (key: string): MRT_VisibilityState => {
  try {
    const value = localStorage.getItem(key);
    return value ? JSON.parse(value) : {};
  } catch {
    return {};
  }
};

const usePersistentColumnVisibility = (
  key: string
): [
  MRT_VisibilityState,
  (updater: MRT_Updater<MRT_VisibilityState>) => void,
] => {
  const [stored, setStored] = useState(() => ({
    key,
    value: readVisibility(key),
  }));

  if (stored.key !== key) {
    setStored({ key, value: readVisibility(key) });
  }

  const setVisibility = (updater: MRT_Updater<MRT_VisibilityState>) => {
    setStored((prev) => ({
      key: prev.key,
      value: updater instanceof Function ? updater(prev.value) : updater,
    }));
  };

  useEffect(() => {
    const hiddenColumns = filterHiddenColumns(stored.value);

    if (!isObjectEmpty(hiddenColumns)) {
      localStorage.setItem(stored.key, JSON.stringify(hiddenColumns));
    } else {
      localStorage.removeItem(stored.key);
    }
  }, [stored]);

  return [stored.value, setVisibility];
};

export default usePersistentColumnVisibility;
