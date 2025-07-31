import { RefObject } from 'react';

export interface GrafanaContextProps {
  frameRef?: RefObject<HTMLIFrameElement>;
  isOnGrafanaPage: boolean;
  isFrameLoaded: boolean;
}
