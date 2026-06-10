import { Box, Stack } from '@mui/material';
import { FC, PropsWithChildren } from 'react';
import { QanAiAside } from './QanAiAside';
import { QanControls } from './QanControls';
import { QanFilterDrawer } from './QanFilterDrawer';
import { QanListing } from './QanListing';
import { QanSectionTab } from './QanSectionTab';
import { useQanPanelActions, useQanPanelState } from '../hooks/useQanPanelState';
import { QanFiltersDrawerProvider } from '../hooks/QanFiltersDrawerProvider';

export const QanLayout: FC<PropsWithChildren> = ({ children }) => {
  const { querySelected, totals } = useQanPanelState();
  const { getSplitRatio } = useQanPanelActions();
  const showSectionTab = querySelected && !totals;
  const listFlex = showSectionTab ? `${getSplitRatio() * 100}%` : '1 1 auto';
  const detailsFlex = showSectionTab ? `${(1 - getSplitRatio()) * 100}%` : undefined;

  return (
    <QanFiltersDrawerProvider>
      <Stack
        direction="row"
        sx={{
          flex: 1,
          minHeight: 0,
          overflow: 'hidden',
          gap: 3,
        }}
      >
        <QanFilterDrawer />
        <Stack
          sx={{
            flex: 1,
            minWidth: 0,
            minHeight: 0,
            px: 3,
            pb: 2,
          }}
        >
          <QanControls />
          <Stack
            sx={{
              flex: 1,
              minHeight: 0,
              overflow: 'hidden',
            }}
          >
            <Box
              sx={{
                flex: showSectionTab ? `0 0 ${listFlex}` : '1 1 auto',
                minHeight: showSectionTab ? '30%' : 0,
                overflow: 'hidden',
                display: 'flex',
                flexDirection: 'column',
              }}
            >
              <QanListing />
            </Box>
            {showSectionTab ? (
              <Box
                sx={{
                  flex: detailsFlex ? `1 1 ${detailsFlex}` : '1 1 50%',
                  minHeight: '20%',
                  overflow: 'hidden',
                  display: 'flex',
                  flexDirection: 'column',
                }}
              >
                <QanSectionTab />
              </Box>
            ) : null}
          </Stack>
          {children}
        </Stack>
        <QanAiAside />
      </Stack>
    </QanFiltersDrawerProvider>
  );
};
