import { Page } from 'components/page';
import { FC, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';

const QueryAnalytics: FC = () => {
  const [params] = useSearchParams();
  const searchParams = useMemo(
    () =>
      new Array(...params.entries()).map(([key, value]) => ({
        key,
        value,
      })),
    [params]
  );

  return (
    <Page title="Query Analytics">
      <ul>
        {searchParams.map(({ key, value }) => (
          <li key={key}>
            <span>{key}: </span>
            <strong>{value}</strong>
          </li>
        ))}
      </ul>
    </Page>
  );
};

export default QueryAnalytics;
