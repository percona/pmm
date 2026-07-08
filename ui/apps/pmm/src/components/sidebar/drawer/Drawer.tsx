import MuiDrawer from '@mui/material/Drawer';
import { closedMixin, openedMixin } from './Drawer.styles';
import { styled } from '@mui/material/styles';
import { DRAWER_WIDTH } from './Drawer.constants';

const Drawer = styled(MuiDrawer, {
  shouldForwardProp: (prop) => prop !== 'open',
})(({ theme }) => ({
  width: DRAWER_WIDTH,
  flexShrink: 0,
  whiteSpace: 'nowrap',
  boxSizing: 'border-box',
  variants: [
    {
      props: ({ open }) => open,
      style: {
        ...openedMixin(theme),
        '& .MuiDrawer-paper': openedMixin(theme),
        [theme.breakpoints.down('md')]: {
          ...closedMixin(theme),
          '& .MuiDrawer-paper': {
            ...openedMixin(theme),
            position: 'fixed',
            top: 0,
            left: 0,
            height: '100vh',
            zIndex: theme.zIndex.drawer + 1,
          },
        },
      },
    },
    {
      props: ({ open }) => !open,
      style: {
        ...closedMixin(theme),
        '& .MuiDrawer-paper': closedMixin(theme),
      },
    },
  ],
}));

export default Drawer;
