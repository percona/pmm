import { useEffect, useMemo, useState } from 'react';
import { AxiosError } from 'axios';
import {
  useAdvisorCheckTestTargets,
  useTestAdvisorCheck,
} from 'hooks/api/useAdvisors';
import {
  AdvisorCheckInput,
  AdvisorTechnology,
  TestAdvisorCheckResult,
} from 'types/advisors.types';
import { Messages } from './CheckTest.messages';

interface UseCheckTestOptions {
  // the check's technology; the test target must be a service of a matching type
  technology?: AdvisorTechnology;
  // gates the services query (e.g. only while the overlay is open)
  enabled: boolean;
  // all test state resets when this changes (e.g. overlay open, check name)
  resetKey: unknown;
}

// useCheckTest holds the state for dry-running a check definition against a
// user-picked service: the service options, the picked service and the last
// run's outcome. Shared by the check form and the check details pane.
export const useCheckTest = ({
  technology,
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

  // a previously picked service is likely of the wrong type after a technology change
  useEffect(() => {
    setServiceId(null);
  }, [technology]);

  // the backend decides eligibility (service type, agent availability, and
  // exclusions like PMM Server's internal PostgreSQL) - the picker offers
  // exactly what checks:test will accept
  const { data: targets } = useAdvisorCheckTestTargets(technology, {
    enabled: enabled && !!technology,
  });
  const serviceOptions = useMemo(
    () =>
      (targets ?? []).map((t) => ({ id: t.serviceId, label: t.serviceName })),
    [targets]
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
