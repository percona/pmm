import { useLocation } from 'react-router-dom';

export const useIsRealTimeQan = () => {
  const { pathname } = useLocation();
  return pathname.includes('rta');
};
