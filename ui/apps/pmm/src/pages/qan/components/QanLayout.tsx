import { Box, Stack } from '@mui/material';
import { FC, PropsWithChildren } from 'react';
import { QanAiAside } from './QanAiAside';
import { QanControls } from './QanControls';
import { QanDetailsPane } from './QanDetailsPane';
import { QanFiltersAside } from './QanFiltersAside';
import { QanListing } from './QanListing';
import { useQanPanelActions, useQanPanelState } from '../hooks/useQanPanelState';

export const QanLayout: FC<PropsWithChildren> = ({ children }) => {
  const { querySelected } = useQanPanelState();
  const { getSplitRatio } = useQanPanelActions();
  const listFlex = querySelected ? `${getSplitRatio() * 100}%` : '1 1 auto';
  const detailsFlex = querySelected ? `${(1 - getSplitRatio()) * 100}%` : undefined;

  return (
    <Stack
      direction="row"
      sx={{
        flex: 1,
        minHeight: 0,
        overflow: 'hidden',
        gap: 0,
      }}
    >
      <QanFiltersAside />
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
              flex: querySelected ? `0 0 ${listFlex}` : '1 1 auto',
              minHeight: querySelected ? '30%' : 0,
              overflow: 'hidden',
              display: 'flex',
              flexDirection: 'column',
            }}
          >
            <QanListing />
          </Box>
          {querySelected ? (
            <Box
              sx={{
                flex: querySelected && detailsFlex ? `1 1 ${detailsFlex}` : '1 1 50%',
                minHeight: '20%',
                overflow: 'hidden',
                borderTop: 1,
                borderColor: 'divider',
                display: 'flex',
                flexDirection: 'column',
              }}
            >
              <QanDetailsPane />
            </Box>
          ) : null}
        </Stack>
        {children}
      </Stack>
      <QanAiAside />
    </Stack>
  );
};
