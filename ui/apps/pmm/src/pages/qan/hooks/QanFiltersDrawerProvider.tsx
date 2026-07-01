import { FC, PropsWithChildren, useMemo, useState } from 'react';
import { QanFiltersDrawerContext } from './useQanFiltersDrawer';

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
