import { Card, Stack } from '@mui/material';
import { Page } from 'components/page';
import { FC, useEffect, useRef, useState } from 'react';
import { useLocation } from 'react-router-dom';

const mapUrl = (url: string) => {
  if (url.includes('/graph/alerts')) {
    return '/graph/alerting';
  }

  return url;
};

export const Grafana: FC<{ url: string }> = ({ url: baseUrl }) => {
  const location = useLocation();
  const url = getUrl(location.pathname);
  const [src] = useState(mapUrl(baseUrl));
  const ref = useRef<HTMLIFrameElement>(null);

  useEffect(() => {
    if (url) {
      ref.current?.contentWindow?.postMessage(
        {
          type: 'NAVIGATE_TO',
          data: {
            to: url,
          },
        },
        '*'
      );
    }
  }, [url]);

  useEffect(() => {
    console.log('grafana-src', src);
  }, [src]);

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
          <iframe ref={ref} src={src}></iframe>
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
