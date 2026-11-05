import { Drawer } from '@mui/material';
import { FC } from 'react';
import { useQanFiltersDrawer } from '../hooks/useQanFiltersDrawer';
import { QanFiltersPanel } from './QanFiltersPanel';

export const QanFilterDrawer: FC = () => {
  const { open, setOpen } = useQanFiltersDrawer();

  return (
    <Drawer
      anchor="left"
      open={open}
      onClose={() => setOpen(false)}
      variant="temporary"
      ModalProps={{ keepMounted: true }}
      sx={{
        '& .MuiDrawer-paper': {
          width: 240,
          boxSizing: 'border-box',
        },
      }}
      data-testid="qan-filter-drawer"
    >
      <QanFiltersPanel />
    </Drawer>
  );
};
