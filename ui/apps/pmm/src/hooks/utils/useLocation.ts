import { useLocation } from 'react-router-dom';

export const useIsRealtimeQan = () => {
  const { pathname } = useLocation();
  return pathname.includes('rta');
};
