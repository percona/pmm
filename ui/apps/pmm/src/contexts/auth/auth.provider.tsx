import {
  FC,
  PropsWithChildren,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { Navigate, useLocation } from 'react-router-dom';
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
  const location = useLocation();
  const returnToHandledRef = useRef(false);
  const [returnTo, setReturnTo] = useState<string | null>(null);
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

  // Pick up the URL the user asked for before being bounced to login.
  useEffect(() => {
    if (!isAuthenticated || returnToHandledRef.current) {
      return;
    }

    // Ref, not state: StrictMode double-invokes this effect with the same state snapshot.
    returnToHandledRef.current = true;

    const target = consumeReturnTo();

    // Grafana's OAuth flow may already have landed us on the target.
    setReturnTo(target && target !== constructUrl(location) ? target : null);
    setReturnToChecked(true);
    // location is deliberately not a dependency; returnToHandledRef gates re-runs
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated]);

  // The redirect is one-shot: drop it on the first location change it produces. Grafana rewrites
  // the dashboard URL (timezone, var-*) once the iframe loads and GrafanaProvider relays that back
  // into the router — leaving the target armed would yank every such rewrite back to the original
  // URL, which Grafana then rewrites again, forever.
  useEffect(() => {
    if (returnTo === null) {
      return;
    }

    setReturnTo(null);
    // returnTo is deliberately not a dependency: this must fire on the location change, not on
    // the render that arms the redirect.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [location]);

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

  // Hold children back until the return-to check has run, then until the restored URL has actually
  // landed. GrafanaPage derives its iframe src from window.location once, when GrafanaProvider
  // first reports a session, so a child render at the stale URL would pin the iframe to the wrong
  // page. Redirect declaratively rather than via navigate(): react-router runs an imperative
  // navigate inside a transition, which commits *after* an urgent setState, so children would
  // render at the old location first.
  if (isAuthenticated && !returnToChecked) {
    return null;
  }

  if (returnTo !== null) {
    return <Navigate to={returnTo} replace />;
  }

  return (
    <AuthContext.Provider value={{ isLoading, isLoggedIn }}>
      {children}
    </AuthContext.Provider>
  );
};
