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
 * Both the settings form and the Support diagnostics setup gate ask the same
 * question of the same LIST response, so the derivation lives here rather than
 * in either surface — TanStack Query dedupes the request itself.
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
