import { MutableRefObject } from 'react';

export interface GrafanaContextProps {
  frameRef?: MutableRefObject<HTMLIFrameElement | undefined>;
  isOnGrafanaPage: boolean;
  isFrameLoaded: boolean;
}
