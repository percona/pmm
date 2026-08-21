import { createContext } from 'react';
import type { GrafanaContextProps } from './grafana.context.types';

export const GrafanaContext = createContext<GrafanaContextProps>({
  isFrameLoaded: false,
  isOnGrafanaPage: false,
  isFullScreen: false,
});
