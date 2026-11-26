import { useWatch } from 'react-hook-form';
import type { ManagedServiceType } from 'types/services.types';
import type { EditServiceFormValues } from './EditService.schema';
import type { SettingName } from './EditService.types';
import { isSettingShown } from './EditService.utils';

export const useIsSettingShown = (serviceType: ManagedServiceType) => {
  const [tls, querySource] = useWatch<
    EditServiceFormValues,
    ['tls', 'querySource']
  >({ name: ['tls', 'querySource'] });

  return (name: SettingName) =>
    isSettingShown(serviceType, name, { tls, querySource });
};
