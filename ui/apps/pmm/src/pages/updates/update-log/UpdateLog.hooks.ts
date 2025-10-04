import { useQuery } from '@tanstack/react-query';
import { getUpdateStatus } from 'api/updates';
import { useEffect, useState } from 'react';

export const useUpdateLog = (authToken: string) => {
  const [output, setOutput] = useState('');
  const [isDone, setIsDone] = useState(false);
  const [logOffset, setLogOffset] = useState(0);
  const { data } = useQuery({
    queryKey: ['updateStatus', { authToken, logOffset }],
    queryFn: ({ queryKey }) => {
      if (typeof queryKey[1] === 'string') {
        return;
      }

      const [, { authToken, logOffset }] = queryKey;

      return getUpdateStatus({ authToken, logOffset });
    },
    refetchInterval: ({ state }) => (state.data?.done ? 0 : 1000),
  });

  useEffect(() => {
    if (!data) {
      return;
    }

    setOutput((prev) => prev + data.logLines.join('\n'));
    setIsDone(data.done);
    setLogOffset(data.logOffset);
  }, [data]);

  return { output, isDone, logOffset };
};
