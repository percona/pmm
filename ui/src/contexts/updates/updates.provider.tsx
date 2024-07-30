import { FC, PropsWithChildren, useEffect, useMemo, useState } from 'react';
import { UpdatesContext } from './updates.context';
import { UpdateStatus } from 'types/updates.types';
import { useCheckUpdates } from 'hooks/api/useUpdates';

export const UpdatesProvider: FC<PropsWithChildren> = ({ children }) => {
  const [status, setStatus] = useState(UpdateStatus.Pending);
  const { isLoading, data, error, isRefetching, refetch } = useCheckUpdates();
  const inProgress = useMemo(
    () =>
      status === UpdateStatus.Updating ||
      status === UpdateStatus.Restarting ||
      status === UpdateStatus.Completed,
    [status]
  );

  useEffect(() => {
    if (error) {
      setStatus(UpdateStatus.Error);
    } else if (isLoading) {
      setStatus(UpdateStatus.Checking);
    } else if (data && data?.installed.version === data?.latest?.version) {
      setStatus(UpdateStatus.UpToDate);
    }
  }, [data, error, isLoading]);

  return (
    <UpdatesContext.Provider
      value={{
        isLoading: isLoading || isRefetching,
        inProgress,
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
