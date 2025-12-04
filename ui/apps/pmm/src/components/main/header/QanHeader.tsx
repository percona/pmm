import { FC, useState } from 'react';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Tabs from '@mui/material/Tabs';
import Tab from '@mui/material/Tab';
import IconButton from '@mui/material/IconButton';
import ShareOutlinedIcon from '@mui/icons-material/ShareOutlined';
import ElectricBoltOutlinedIcon from '@mui/icons-material/ElectricBoltOutlined';
import { useLocation, useNavigate } from 'react-router-dom';
import { Badge } from '@mui/material';

const QanHeader: FC = () => {
  const pathname = useLocation().pathname;
  const navigate = useNavigate();

  const handleChange = (event: React.SyntheticEvent, newValue: string) => {
    console.log('newValue', newValue);

    if (newValue === 'historical') {
      navigate('/next/graph/d/pmm-qan/pmm-query-analytics');
    } else {
      navigate('/next/rta');
    }
  };

  return (
    <Stack
      sx={{
        pt: 1,
        px: 2,
        gap: 3,
        flexDirection: 'row',
        justifyContent: 'flex-start',
        alignItems: 'center',
      }}
    >
      <Typography variant="h6">Query Analytics</Typography>
      <Tabs
        sx={{
          flex: 1,
        }}
        value={pathname.includes('rta') ? 'real-time' : 'historical'}
        onChange={handleChange}
      >
        <Tab value="historical" label="Historical" />
        <Tab value="real-time" label="Real-Time" />
      </Tabs>
      <Stack gap={1} flex={1} flexDirection="row" justifyContent="flex-end">
        <IconButton>
          <Badge color="warning" badgeContent={3}>
            <ElectricBoltOutlinedIcon />
          </Badge>
        </IconButton>
        <IconButton>
          <ShareOutlinedIcon />
        </IconButton>
      </Stack>
    </Stack>
  );
};

export default QanHeader;
