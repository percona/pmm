import { RealtimeSessionStatus } from 'types/rta.types';

export const getSessionStatusText = (status: RealtimeSessionStatus) => {
  switch (status) {
    case RealtimeSessionStatus.running:
      return 'Running';
    case RealtimeSessionStatus.error:
      return 'Error';
    case RealtimeSessionStatus.down:
      return 'Down';
    case RealtimeSessionStatus.unspecified:
      return 'Unspecified';
  }
};
