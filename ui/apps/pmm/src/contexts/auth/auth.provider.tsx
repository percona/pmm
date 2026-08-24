import {
  FC,
  PropsWithChildren,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { AuthContext } from './auth.context';
import { useQuery } from '@tanstack/react-query';
import { rotateToken } from 'api/auth';
import {
  establishClientSession,
  ensureClientSessionListener,
  useClientSession,
} from './auth.clientSession';
import { ROTATE_TOKEN_QUERY_KEY } from './auth.queryKeys';
import { getRefetchInterval, redirectToLogin } from './auth.utils';
import { consumeReturnTo } from './auth.returnTo';
import { constructUrl } from 'utils/link.utils';
import { AxiosError, HttpStatusCode } from 'axios';
import { useFrontendSettings } from 'hooks/api/useSettings';

export const AuthProvider: FC<PropsWithChildren> = ({ children }) => {
  const settings = useFrontendSettings({ retry: false });
  const clientSessionEstablished = useClientSession();
  const navigate = useNavigate();
  const location = useLocation();
  const returnToHandledRef = useRef(false);
  const [returnToChecked, setReturnToChecked] = useState(false);

  useEffect(() => {
    ensureClientSessionListener();
  }, []);

  const { error, isLoading, data } = useQuery({
    queryKey: ROTATE_TOKEN_QUERY_KEY,
    queryFn: async () => {
      const token = await rotateToken();
      establishClientSession();
      return token;
    },
    refetchInterval: () => getRefetchInterval(),
    refetchIntervalInBackground: true,
    retry: false,
  });

  const hasServerSession = Boolean(data);
  const isLoggedIn = hasServerSession && clientSessionEstablished;
  const isAuthenticated =
    isLoggedIn || Boolean(settings.data?.anonymousEnabled);

  // Restore the URL the user asked for before being bounced to login. navigate() mutates history
  // synchronously and batches with setReturnToChecked, so window.location is already the target by
  // the time children — GrafanaPage included — first render.
  useEffect(() => {
    if (!isAuthenticated || returnToHandledRef.current) {
      return;
    }

    // Ref, not state: StrictMode double-invokes this effect with the same state snapshot.
    returnToHandledRef.current = true;

    const target = consumeReturnTo();

    // Grafana's OAuth flow may already have landed us on the target.
    if (target && target !== constructUrl(location)) {
      navigate(target, { replace: true });
    }

    setReturnToChecked(true);
    // location is deliberately not a dependency; returnToHandledRef gates re-runs
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated]);

  const shouldRedirectToLogin = useMemo(() => {
    if (settings.data?.anonymousEnabled) {
      return false;
    }

    const response = (error as AxiosError)?.response;
    if (
      response?.status === HttpStatusCode.Unauthorized ||
      response?.status === HttpStatusCode.InternalServerError
    ) {
      return true;
    }

    return hasServerSession && !clientSessionEstablished;
  }, [
    clientSessionEstablished,
    error,
    hasServerSession,
    settings.data?.anonymousEnabled,
  ]);

  if (isLoading || settings.isLoading) {
    return null;
  }

  if (shouldRedirectToLogin) {
    redirectToLogin();
    return null;
  }

  // Hold children back until the return-to check has run. GrafanaPage derives its iframe src from
  // window.location once, so the restore has to land before it mounts.
  if (isAuthenticated && !returnToChecked) {
    return null;
  }

  return (
    <AuthContext.Provider value={{ isLoading, isLoggedIn }}>
      {children}
    </AuthContext.Provider>
  );
};
