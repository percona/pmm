import { useCallback, useState } from 'react';

export const useLocalStorage = <T>(
  key: string,
  initialValue?: T
): [T, (value: T) => void] => {
  const [state, setState] = useState(
    !!localStorage.getItem(key)
      ? (JSON.parse(localStorage.getItem(key) || '') as T)
      : initialValue
  );

  const setValue = useCallback(
    (value: T): void => {
      localStorage.setItem(key, JSON.stringify(value));
      setState(value);
    },
    [key]
  );

  return [state as T, setValue];
};
