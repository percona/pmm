import { useSearchParams } from 'react-router-dom';

export const useKioskMode = () => {
  const [params] = useSearchParams();
  const kioskParam = params.get('kiosk');
  return { active: kioskParam === '' || kioskParam === 'true' };
};
