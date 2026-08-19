export interface VersionContextProps {
  /** The server runs a different build than the one that served this page. */
  isOutdated: boolean;
  /** Version the server reports now, as shown to users, e.g. `3.10.0`. */
  serverVersion: string;
  reload: () => void;
}
