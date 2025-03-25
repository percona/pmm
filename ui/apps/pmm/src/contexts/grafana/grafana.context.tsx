import { createContext } from 'react';
import { GrafanaContextProps } from './grafana.context.types';

export const GrafanaContext = createContext<GrafanaContextProps>({
  isFrameLoaded: false,
  isOnGrafanaPage: false,
});
