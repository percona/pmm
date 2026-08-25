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
 *
 * `enabled: false` skips the request and reports the same shape a failed read
 * does. SEP holds `GET /sep/admin/settings` to administrators including its
 * reads, so a caller that may be rendered for a non-admin passes `false` rather
 * than firing a request that can only answer 403.
 */
export const useServiceNowConnection = ({
  enabled = true,
}: { enabled?: boolean } = {}) => {
  const { data: groups, isLoading, error } = useSettingsList({ enabled });

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
