import type { RefObject } from 'react';

export interface GrafanaContextProps {
  frameRef?: RefObject<HTMLIFrameElement | null>;
  isOnGrafanaPage: boolean;
  isFrameLoaded: boolean;
  isFullScreen: boolean;
}
