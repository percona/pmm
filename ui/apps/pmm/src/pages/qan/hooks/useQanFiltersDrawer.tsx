import { createContext, useContext } from 'react';

export type QanFiltersDrawerContextValue = {
  open: boolean;
  setOpen: (open: boolean) => void;
  toggle: () => void;
};

export const QanFiltersDrawerContext = createContext<QanFiltersDrawerContextValue | null>(null);

export function useQanFiltersDrawer(): QanFiltersDrawerContextValue {
  const ctx = useContext(QanFiltersDrawerContext);
  if (!ctx) {
    throw new Error('useQanFiltersDrawer must be used within QanFiltersDrawerProvider');
  }
  return ctx;
}
