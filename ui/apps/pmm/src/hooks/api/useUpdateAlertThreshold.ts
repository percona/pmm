import { useMutation } from '@tanstack/react-query';
import { grafanaApi } from 'api/api';
import {
  setOverride,
  updateAlertRuleExpr,
} from 'components/alert-thresholds/utils/update-alert-rule';

export interface UpdateAlertThresholdData {
  uid: string;
  nodeName: string;
  threshold: number;
}

export const useUpdateAlertThreshold = () =>
  useMutation({
    mutationKey: ['updateAlertThreshold'],
    mutationFn: async (data: UpdateAlertThresholdData) => updateThreshold(data),
  });

export const useUpdateAlertThresholds = () =>
  useMutation({
    mutationKey: ['updateAlertThresholds'],
    mutationFn: async (data: UpdateAlertThresholdData[]) =>
      Promise.all(data.map((d) => updateThreshold(d))),
  });

const updateThreshold = async (data: UpdateAlertThresholdData) => {
  const alertRule = (
    await grafanaApi.get('/v1/provisioning/alert-rules/' + data.uid)
  ).data;

  const updated = updateAlertRuleExpr(alertRule, (e) =>
    setOverride(e, data.nodeName, data.threshold)
  );

  const res = await grafanaApi.put(
    '/v1/provisioning/alert-rules/' + data.uid,
    updated
  );

  return res;
};
