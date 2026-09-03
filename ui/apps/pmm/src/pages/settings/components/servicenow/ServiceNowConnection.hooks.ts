import { useMemo } from 'react';
import { useSettingsList } from '@sep/api';
import {
  connectionStatus,
  declaredSecretNames,
  storedDeliveryInputs,
} from './ServiceNowConnection.utils';

/**
 * What SEP currently holds for ServiceNow delivery, read once and derived.
 *
 * SEP holds `GET /sep/admin/settings` to administrators, reads included, so
 * every caller must already be admin-only — today that is the settings form.
 */
export const useServiceNowConnection = () => {
  const { data: groups, isLoading, error } = useSettingsList();

  const declaredNames = useMemo(() => declaredSecretNames(groups), [groups]);
  const stored = useMemo(() => storedDeliveryInputs(groups), [groups]);

  return {
    declaredNames,
    stored,
    status: connectionStatus(declaredNames, stored),
    isLoading,
    error,
  };
};
