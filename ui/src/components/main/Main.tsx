import { Stack } from '@mui/material';
import { Outlet } from 'react-router-dom';
import { AppBar } from '../app-bar/AppBar';

export const Main = () => (
  <Stack>
    <AppBar />
    <Outlet />
  </Stack>
);
