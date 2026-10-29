import type { RefObject } from 'react';

export interface GrafanaContextProps {
  frameRef?: RefObject<HTMLIFrameElement | null>;
  grafanaReady: boolean;
  isOnGrafanaPage: boolean;
  isFrameLoaded: boolean;
  isFullScreen: boolean;
}
