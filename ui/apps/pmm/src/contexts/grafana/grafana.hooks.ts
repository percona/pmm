import { useContext } from 'react';
import { GrafanaContext } from './grafana.context';

export const useGrafana = () => useContext(GrafanaContext);
