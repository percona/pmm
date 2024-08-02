import { FC, PropsWithChildren, useEffect, useMemo, useState } from 'react';
import { UpdatesContext } from './updates.context';
import { UpdateStatus } from 'types/updates.types';
import { useCheckUpdates } from 'hooks/api/useUpdates';
import { useAgentVersions } from 'hooks/api/useAgents';
import { AgentUpdateSeverity } from 'types/agent.types';

export const UpdatesProvider: FC<PropsWithChildren> = ({ children }) => {
  const [status, setStatus] = useState(UpdateStatus.Pending);
  const { isLoading, data, error, isRefetching, refetch } = useCheckUpdates();
  const { data: clients } = useAgentVersions();
  const inProgress = useMemo(
    () =>
      status === UpdateStatus.Updating ||
      status === UpdateStatus.Restarting ||
      status === UpdateStatus.Completed,
    [status]
  );

  useEffect(() => {
    const serverUpToDate =
      data && data?.installed.version === data?.latest?.version;
    const clientsUpToDate = clients?.every(
      (client) => client.severity === AgentUpdateSeverity.UP_TO_DATE
    );

    if (error) {
      setStatus(UpdateStatus.Error);
    } else if (isLoading) {
      setStatus(UpdateStatus.Checking);
    } else if (serverUpToDate && !clientsUpToDate) {
      setStatus(UpdateStatus.UpdateClients);
    } else if (serverUpToDate) {
      setStatus(UpdateStatus.UpToDate);
    } else {
      setStatus(UpdateStatus.Pending);
    }
  }, [data, error, isLoading, clients]);

  return (
    <UpdatesContext.Provider
      value={{
        isLoading: isLoading || isRefetching,
        inProgress,
        clients,
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
