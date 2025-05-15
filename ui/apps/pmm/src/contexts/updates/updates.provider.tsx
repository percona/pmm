import { FC, PropsWithChildren, useEffect, useMemo, useState } from 'react';
import { UpdatesContext } from './updates.context';
import { UpdateStatus } from 'types/updates.types';
import { useCheckUpdates } from 'hooks/api/useUpdates';
import { useAgentVersions } from 'hooks/api/useAgents';
import * as utils from './updates.utils';

export const UpdatesProvider: FC<PropsWithChildren> = ({ children }) => {
  const [status, setStatus] = useState(UpdateStatus.Pending);
  const { isLoading, data, error, isRefetching, refetch } = useCheckUpdates();
  const { data: clients } = useAgentVersions();
  const inProgress = useMemo(() => utils.isUpdateInProgress(status), [status]);
  const areClientsUpToDate = useMemo(
    () => utils.areClientsUpToDate(clients),
    [clients]
  );

  useEffect(() => {
    const serverUpToDate =
      data && data?.installed.version === data?.latest?.version;

    if (error) {
      setStatus(UpdateStatus.Error);
    } else if (isLoading) {
      setStatus(UpdateStatus.Checking);
    } else if (serverUpToDate && !areClientsUpToDate) {
      setStatus(UpdateStatus.UpdateClients);
    } else if (serverUpToDate) {
      setStatus(UpdateStatus.UpToDate);
    } else {
      setStatus(UpdateStatus.Pending);
    }
  }, [data, error, isLoading, clients, areClientsUpToDate]);

  return (
    <UpdatesContext.Provider
      value={{
        isLoading: isLoading || isRefetching,
        inProgress,
        clients,
        areClientsUpToDate,
        status,
        setStatus,
        versionInfo: data,
        recheck: refetch,
      }}
    >
      {children}
    </UpdatesContext.Provider>
  );
};
