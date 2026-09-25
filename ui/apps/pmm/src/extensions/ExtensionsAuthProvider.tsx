import { FC, PropsWithChildren, useMemo } from 'react';
import { AuthContext, type AuthSession } from '@pmm-extensions/api';
import { useUser } from 'contexts/user';

/**
 * Fills the PMM Extensions auth context from the PMM session.
 *
 * The side-car's own shell derives the flag from its `/api/users/me` record. PMM does
 * not need that round trip: the side-car's Grafana auth provider maps `isGrafanaAdmin`
 * to its super-admin role and otherwise takes the highest org role, which is
 * exactly what `isPMMAdmin` (`isGrafanaAdmin || orgRole === Admin`) already
 * says — and PMM has it loaded before a PMM Extensions route renders, so admins never see
 * their controls appear a beat late.
 *
 * It is not the authorization boundary. The side-car's API re-checks the identity behind
 * the exchanged bearer on every unsafe route; this only decides which controls
 * are worth offering.
 */
export const ExtensionsAuthProvider: FC<PropsWithChildren> = ({ children }) => {
  const { user } = useUser();
  const session = useMemo<AuthSession>(
    () => ({ isAdmin: Boolean(user?.isPMMAdmin) }),
    [user?.isPMMAdmin]
  );

  return (
    <AuthContext.Provider value={session}>{children}</AuthContext.Provider>
  );
};
