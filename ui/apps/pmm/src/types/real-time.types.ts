export type RealTimeQueryState =
  | 'Running'
  | 'Blocked'
  | 'Sorting result'
  | 'Waiting';

export interface RealTimeQuery {
  query: string;
  service: string;
  duration: string;
  state: RealTimeQueryState;
}
