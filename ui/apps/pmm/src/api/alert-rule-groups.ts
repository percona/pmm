import {
  RulerRuleGroup,
  RulerRulesConfig,
} from 'types/alert-rule-groups.types';
import { grafanaApi } from './api';

// Mirrors Grafana's useFetchGroupsForFolder: the ruler namespace is the folder
// UID, and the response is keyed by namespace -> groups.
export const getFolderRuleGroups = async (
  folderUid: string
): Promise<RulerRuleGroup[]> => {
  const res = await grafanaApi.get<RulerRulesConfig>(
    `/ruler/grafana/api/v1/rules/${encodeURIComponent(folderUid)}`,
    { params: { subtype: 'cortex' } }
  );
  return Object.values(res.data ?? {}).flat();
};
