import { FC, PropsWithChildren, useEffect, useRef, useState } from 'react';
import { GrafanaContext } from './grafana.context';
import { useLocation } from 'react-router';
import { PMM_NEW_NAV_PATH } from 'lib/constants';

export const GrafanaProvider: FC<PropsWithChildren> = ({ children }) => {
  const location = useLocation();
  const src = location.pathname.replace(PMM_NEW_NAV_PATH, '');
  const isGrafanaPage = src.startsWith('/graph');
  const [isLoaded, setIsloaded] = useState(false);
  const frameRef = useRef<HTMLIFrameElement>();

  useEffect(() => {
    if (isGrafanaPage) {
      setIsloaded(true);
    }
  }, [isGrafanaPage]);

  return (
    <GrafanaContext.Provider
      value={{
        frameRef,
        isFrameLoaded: isLoaded,
        isOnGrafanaPage: isGrafanaPage,
      }}
    >
      {children}
    </GrafanaContext.Provider>
  );
};
