import { useQuery, UseQueryOptions } from '@tanstack/react-query';
import { getFolderRuleGroups } from 'api/alert-rule-groups';
import { RulerRuleGroup } from 'types/alert-rule-groups.types';

export const useFolderRuleGroups = (
  folderUid: string,
  options?: Partial<UseQueryOptions<RulerRuleGroup[]>>
) =>
  useQuery({
    queryKey: ['alert-rule-groups', folderUid],
    queryFn: () => getFolderRuleGroups(folderUid),
    enabled: !!folderUid,
    ...options,
  });
