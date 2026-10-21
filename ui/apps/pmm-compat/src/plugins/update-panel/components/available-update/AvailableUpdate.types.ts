import type { LatestInformation } from 'plugins/shared/types/updates.types';

export interface AvailableUpdateProps {
  nextVersion?: LatestInformation;
  newsLink?: string;
}
