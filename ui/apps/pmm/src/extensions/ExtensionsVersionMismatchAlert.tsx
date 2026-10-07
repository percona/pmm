import type { FC } from 'react';
import Alert from '@mui/material/Alert';
import { useAppInfo } from '@pmm-extensions/api';
import { useVersion } from 'contexts/version';
import { Messages } from './ExtensionsVersionMismatchAlert.messages';
import { toReleaseVersion } from './toReleaseVersion';

/**
 * Non-blocking warning when the running PMM Extensions release differs from
 * PMM Server's (see PMM-15671). Mounted inside `ExtensionsAuthGate` so the
 * app-info request only fires after the side-car session is ready, and never
 * replaces the page content below it.
 */
export const ExtensionsVersionMismatchAlert: FC = () => {
  const { serverVersion } = useVersion();
  const { data, isSuccess, isError, isPending } = useAppInfo();

  if (!serverVersion || isPending || isError || !isSuccess) {
    return null;
  }

  const extensionsVersion = data.version;
  if (extensionsVersion == null || extensionsVersion === '') {
    return (
      <Alert severity="warning" data-testid="extensions-version-mismatch">
        {Messages.undetermined}
      </Alert>
    );
  }

  const serverRelease = toReleaseVersion(serverVersion);
  const extensionsRelease = toReleaseVersion(extensionsVersion);

  if (!serverRelease) {
    return null;
  }

  if (!extensionsRelease) {
    return (
      <Alert severity="warning" data-testid="extensions-version-mismatch">
        {Messages.undetermined}
      </Alert>
    );
  }

  if (serverRelease === extensionsRelease) {
    return null;
  }

  return (
    <Alert severity="warning" data-testid="extensions-version-mismatch">
      {Messages.mismatch(serverVersion, extensionsVersion)}
    </Alert>
  );
};
