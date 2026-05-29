import { createContext, FC, PropsWithChildren, useContext, useMemo, useState } from 'react';

type QanFiltersDrawerContextValue = {
  open: boolean;
  setOpen: (open: boolean) => void;
  toggle: () => void;
};

const QanFiltersDrawerContext = createContext<QanFiltersDrawerContextValue | null>(null);

export const QanFiltersDrawerProvider: FC<PropsWithChildren> = ({ children }) => {
  const [open, setOpen] = useState(false);
  const value = useMemo(
    () => ({
      open,
      setOpen,
      toggle: () => setOpen((v) => !v),
    }),
    [open]
  );
  return (
    <QanFiltersDrawerContext.Provider value={value}>
      {children}
    </QanFiltersDrawerContext.Provider>
  );
};

export function useQanFiltersDrawer(): QanFiltersDrawerContextValue {
  const ctx = useContext(QanFiltersDrawerContext);
  if (!ctx) {
    throw new Error('useQanFiltersDrawer must be used within QanFiltersDrawerProvider');
  }
  return ctx;
}
