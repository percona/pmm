import { Card, Stack } from '@mui/material';
import { Page } from 'components/page';
import { useMessenger } from 'contexts/messages/messages.provider';
import { FC, useEffect, useState } from 'react';
import { useLocation, Location } from 'react-router-dom';
import { constructUrlFromLocation } from 'utils/url';

const mapUrl = (url: string) => {
  if (url.includes('/graph/alerts')) {
    return '/graph/alerting';
  }

  return url;
};

export const Grafana: FC<{ url: string; shouldNavigate: boolean }> = ({
  url: baseUrl,
  shouldNavigate,
}) => {
  const location = useLocation();
  const url = getUrl(location);
  const [src] = useState(mapUrl(baseUrl));
  const { frameRef, isReady, sendMessage } = useMessenger();
  const [render, setRender] = useState(false);

  useEffect(() => {
    if (isReady && shouldNavigate && url) {
      console.log('NAVIGATE_TO_URL', url);
      sendMessage({
        type: 'NAVIGATE_TO',
        data: {
          to: url,
        },
      });
    }
  }, [isReady, shouldNavigate, url]);

  useEffect(() => {
    // load grafana on demand
    if (shouldNavigate) {
      setRender(true);
    }
  }, [shouldNavigate]);

  if (!render) {
    return;
  }

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
          <iframe ref={frameRef} src={src}></iframe>
        </Stack>
      </Card>
    </Page>
  );
};

const getUrl = (location: Location<unknown>) => {
  if (location.pathname.startsWith('/graph')) {
    return constructUrlFromLocation(location);
  }

  return '';
};
