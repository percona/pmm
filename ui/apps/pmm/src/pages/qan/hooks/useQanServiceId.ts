import { useMemo } from 'react';
import { useManagedServices } from 'hooks/api/useServices';
import type { QanQueryExample } from 'types/qan.types';
import { useQanPanelState } from './useQanPanelState';
import { resolveServiceUuid, serviceIdFromExample } from '../utils/qanServiceResolve';

/** Resolved PMM service UUID for QAN detail tabs, ADRE, and explain. */
export function useQanServiceId(examples?: QanQueryExample[]): string {
  const { labels } = useQanPanelState();
  const { data } = useManagedServices({}, { staleTime: 120_000 });

  return useMemo(
    () =>
      resolveServiceUuid(
        labels,
        data?.services,
        serviceIdFromExample(examples) || undefined
      ),
    [labels, data?.services, examples]
  );
}
