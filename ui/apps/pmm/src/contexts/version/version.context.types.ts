export interface VersionContextProps {
  /** The server runs a different build than the one that served this page. */
  isOutdated: boolean;
  /** Version the server reports now, as shown to users, e.g. `3.10.0`. */
  serverVersion: string;
  /** Identity of the build the server runs, which a rebuild changes too. */
  serverBuild: string;
  reload: () => void;
}
