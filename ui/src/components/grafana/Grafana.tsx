import { Card, Stack } from '@mui/material';
import { Page } from 'components/page';
import { FC, useEffect, useRef } from 'react';
import { useLocation } from 'react-router-dom';

export const Grafana: FC<{ url: string }> = () => {
  const location = useLocation();
  const url = getUrl(location.pathname);
  const ref = useRef<HTMLIFrameElement>(null);

  useEffect(() => {
    if (url) {
      ref.current?.contentWindow?.postMessage(
        {
          type: 'NAVIGATE-TO',
          to: url,
        },
        '*'
      );
    }
  }, [url]);

  return (
    <Page>
      <Card>
        <Stack
          sx={{
            iframe: {
              border: 'none',
              height: '100vh',
            },
          }}
        >
          <iframe ref={ref} src="/graph"></iframe>
        </Stack>
      </Card>
    </Page>
  );
};

const getUrl = (pathname: string) => {
  if (pathname.startsWith('/d')) {
    return pathname;
  }

  if (pathname.startsWith('/alerts')) {
    return '/alerting';
  }

  return '';
};
