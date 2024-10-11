import { useQuery } from '@tanstack/react-query';
import { getUpdateStatus } from 'api/updates';
import { useUpdates } from 'contexts/updates';
import { useEffect, useState } from 'react';
import { UpdateStatus } from 'types/updates.types';

export const useUpdateLog = (authToken: string) => {
  const [output, setOutput] = useState('');
  const [isDone, setIsDone] = useState(false);
  const [logOffset, setLogOffset] = useState(0);
  const { setStatus } = useUpdates();
  const { data } = useQuery({
    queryKey: ['updateStatus', { authToken, logOffset }],
    queryFn: ({ queryKey }) => {
      if (typeof queryKey[1] === 'string') {
        return;
      }

      const [, { authToken, logOffset }] = queryKey;

      return getUpdateStatus({ authToken, logOffset });
    },
    refetchInterval: (query) => (query.state.data?.done ? undefined : 500),
  });

  useEffect(() => {
    if (!data) {
      return;
    }

    setOutput((prev) => prev + data.logLines.join('\n'));
    setIsDone(data.done);
    setLogOffset(data.logOffset);

    if (data.done) {
      setStatus(UpdateStatus.Completed);
    }
  }, [data, setStatus]);

  return { output, isDone, logOffset };
};
