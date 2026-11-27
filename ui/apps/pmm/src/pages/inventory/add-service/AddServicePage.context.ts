import { createContext, useContext, useEffect } from 'react';

// Lets a discovery page point the back link at its instance list while an instance's form is open.
export const BackToListContext = createContext<(onBack?: () => void) => void>(
  () => undefined
);

export const useBackToList = (onBack?: () => void) => {
  const setOnBack = useContext(BackToListContext);

  useEffect(() => {
    setOnBack(onBack);
    return () => setOnBack(undefined);
  }, [setOnBack, onBack]);
};
