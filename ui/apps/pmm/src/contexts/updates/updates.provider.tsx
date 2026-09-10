import { FC, PropsWithChildren, useEffect, useMemo, useState } from 'react';
import { UpdatesContext } from './updates.context';
import { UpdateStatus } from 'types/updates.types';
import { useCheckUpdates } from 'hooks/api/useUpdates';
import { useAgentVersions } from 'hooks/api/useAgents';
import * as utils from './updates.utils';
import { useSettings } from 'contexts/settings';
import { useUser } from 'contexts/user';

export const UpdatesProvider: FC<PropsWithChildren> = ({ children }) => {
  const { settings, isLoading: isLoadingSettings } = useSettings();
  const [status, setStatus] = useState(UpdateStatus.Pending);
  const { user } = useUser();
  const { isLoading, data, error, isRefetching, refetch } = useCheckUpdates({
    // wait for settings to settle, otherwise a full check can fire before we
    // know whether this deployment allows one. Gate on loading rather than on
    // success: if settings fail there is no retry, and blocking on that would
    // cost us the version in the footer for the rest of the session.
    enabled:
      !isLoadingSettings &&
      !settings?.frontend?.anonymousEnabled &&
      !!user?.isPMMAdmin,
    onlyInstalledVersion: settings?.updatesEnabled === false,
  });
  const { data: clients } = useAgentVersions({
    enabled: !settings?.frontend?.anonymousEnabled && !!user?.isPMMAdmin,
  });
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
