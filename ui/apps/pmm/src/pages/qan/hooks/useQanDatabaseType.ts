import { useMemo } from 'react';
import { useManagedServices } from 'hooks/api/useServices';
import type { QanDatabaseType, QanLabelsMap } from 'types/qan.types';
import { inferDatabaseType } from '../utils/qanDatabase';
import { serviceTypeToQanDatabase } from '../utils/qanServiceType';
import { resolveServiceUuid } from '../utils/qanServiceResolve';

export function useQanDatabaseType(
  labels: QanLabelsMap,
  database?: string
): QanDatabaseType {
  const { data } = useManagedServices({}, { staleTime: 120_000 });

  const fromService = useMemo(() => {
    const serviceId = resolveServiceUuid(labels, data?.services);
    if (!serviceId || !data?.services?.length) return undefined;
    const svc = data.services.find((s) => s.serviceId === serviceId);
    return serviceTypeToQanDatabase(svc?.serviceType);
  }, [labels, data?.services]);

  return fromService ?? inferDatabaseType(database);
}
