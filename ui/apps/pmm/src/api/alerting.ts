import {
  AlertEvalResponse,
  AlertmanagerAlert,
  AlertmanagerSilence,
  GrafanaAlertQuery,
  GrafanaRulerRuleDTO,
  PrometheusAlertRulesResponse,
} from 'types/alerting.types';
import { grafanaApi } from './api';

export const getPrometheusAlertRules = async () => {
  const response = await grafanaApi.get<PrometheusAlertRulesResponse>(
    '/prometheus/grafana/api/v1/rules'
  );
  return response.data;
};

// Fetch only the suppressed (silenced) alerts from Grafana's internal
// Alertmanager. Each carries `status.silencedBy`, which we correlate onto the
// rules-API rows by labels to flag them as silenced.
export const getAlertmanagerAlerts = async () => {
  const response = await grafanaApi.get<AlertmanagerAlert[]>(
    '/alertmanager/grafana/api/v2/alerts',
    {
      params: {
        silenced: true,
        active: false,
        inhibited: false,
        unprocessed: false,
      },
    }
  );
  return response.data;
};

// Fetch the configured silences; joined to an alert's `silencedBy` ids to derive
// how long the alert has been silenced.
export const getAlertmanagerSilences = async () => {
  const response = await grafanaApi.get<AlertmanagerSilence[]>(
    '/alertmanager/grafana/api/v2/silences'
  );
  return response.data;
};

export const evalAlertQueries = async (data: GrafanaAlertQuery[]) => {
  const response = await grafanaApi.post<AlertEvalResponse>('/v1/eval', {
    data,
  });
  return response.data;
};

export const getRulerRule = async (uid: string) => {
  const res = await grafanaApi.get<GrafanaRulerRuleDTO>(
    `/ruler/grafana/api/v1/rule/${uid}`
  );
  return res.data;
};
