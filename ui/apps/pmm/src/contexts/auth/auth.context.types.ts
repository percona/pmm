export interface AuthContextProps {
  isLoading: boolean;
  isLoggedIn: boolean;
  // Not signed in, but let through by Grafana anonymous access.
  isAnonymous: boolean;
}
