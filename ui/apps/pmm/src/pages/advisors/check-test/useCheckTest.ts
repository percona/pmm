import { useEffect, useMemo, useState } from 'react';
import { AxiosError } from 'axios';
import { useTestAdvisorCheck } from 'hooks/api/useAdvisors';
import { useServices } from 'hooks/api/useServices';
import {
  AdvisorCheckInput,
  AdvisorFamily,
  TestAdvisorCheckResult,
} from 'types/advisors.types';
import { VersionedService } from 'types/services.types';
import { ADVISOR_FAMILY_SERVICE_TYPE } from 'utils/advisors.utils';
import { Messages } from './CheckTest.messages';

// PMM Server's internal PostgreSQL is monitored but deliberately excluded from
// advisor check targets (models.PMMServerPostgreSQLServiceName on the backend),
// so it must not be offered as a test target either
const PMM_SERVER_POSTGRESQL_SERVICE_NAME = 'pmm-server-postgresql';

interface UseCheckTestOptions {
  // the check's family; the test target must be a service of a matching type
  family?: AdvisorFamily;
  // gates the services query (e.g. only while the overlay is open)
  enabled: boolean;
  // all test state resets when this changes (e.g. overlay open, check name)
  resetKey: unknown;
}

// useCheckTest holds the state for dry-running a check definition against a
// user-picked service: the service options, the picked service and the last
// run's outcome. Shared by the check form and the check details pane.
export const useCheckTest = ({
  family,
  enabled,
  resetKey,
}: UseCheckTestOptions) => {
  const [serviceId, setServiceId] = useState<string | null>(null);
  const [results, setResults] = useState<TestAdvisorCheckResult[] | null>(null);
  const [scriptOutput, setScriptOutput] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const closeResults = () => {
    setResults(null);
    setScriptOutput(null);
    setError(null);
  };

  // drop stale test state whenever the subject changes
  useEffect(() => {
    setServiceId(null);
    setResults(null);
    setScriptOutput(null);
    setError(null);
  }, [resetKey]);

  // a previously picked service is likely of the wrong type after a family change
  useEffect(() => {
    setServiceId(null);
  }, [family]);

  const serviceType = family ? ADVISOR_FAMILY_SERVICE_TYPE[family] : undefined;
  const { data: servicesResponse } = useServices(
    { serviceType },
    { enabled: enabled && !!serviceType }
  );
  const serviceOptions = useMemo(
    () =>
      (Object.values(servicesResponse ?? {}).flat() as VersionedService[])
        .filter((s) => s.serviceName !== PMM_SERVER_POSTGRESQL_SERVICE_NAME)
        .map((s) => ({ id: s.serviceId, label: s.serviceName })),
    [servicesResponse]
  );

  const { mutateAsync: testCheck, isPending: isTesting } =
    useTestAdvisorCheck();

  const runTest = async (check: AdvisorCheckInput) => {
    if (!serviceId) {
      return;
    }
    closeResults();
    try {
      const { results: runResults, scriptOutput: output } = await testCheck({
        check,
        serviceId,
      });
      setResults(runResults ?? []);
      setScriptOutput(output || null);
    } catch (err) {
      const message =
        err instanceof AxiosError
          ? (err.response?.data?.message ?? err.message)
          : Messages.testFailed;
      setError(message);
    }
  };

  return {
    serviceId,
    setServiceId,
    serviceOptions,
    isTesting,
    results,
    scriptOutput,
    error,
    runTest,
    closeResults,
  };
};

export type CheckTestState = ReturnType<typeof useCheckTest>;
