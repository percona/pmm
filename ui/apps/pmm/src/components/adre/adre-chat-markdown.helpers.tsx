import { Box, Link } from '@mui/material';
import { ReactNode } from 'react';
import { CodeBlock } from 'pages/updates/change-log/code-block';
import { GrafanaPanelImage } from './adre-chat-markdown';
import {
  dashboardUrlFromRenderUrl,
  getRenderImageUrlsInContent,
  isGrafanaRenderImageSrc,
  parseDashboardLinkToPanelKey,
  parseRenderImageUrlToPanelKey,
  toGrafanaDashboardLink,
  toSameOriginUrl,
  withRenderCacheParam,
} from './adre-chat-markdown.utils';

/** Returns markdown component overrides for rendering Grafana panel images, code blocks, and dashboard links within chat messages. */
export function getMarkdownComponents(content: string) {
  const panelKeysFromImages = new Set(
    getRenderImageUrlsInContent(content).map(parseRenderImageUrlToPanelKey).filter(Boolean)
  );

  return {
    // Default <p> cannot contain block-level panel UI; browsers break the DOM and width
    // constraints fail, so wide panel images overflow the chat bubble.
    p: ({ children }: { children?: ReactNode }) => (
      <Box component="div" sx={{ mb: 1.25, minWidth: 0, maxWidth: '100%' }}>
        {children}
      </Box>
    ),
    code: ({
      inline,
      children,
    }: {
      inline?: boolean;
      children?: ReactNode;
    }) => {
      if (inline) {
        return (
          <Box
            component="code"
            sx={{
              px: 0.5,
              borderRadius: 0.5,
              fontFamily: 'Roboto Mono, monospace',
              bgcolor: 'action.hover',
            }}
          >
            {children}
          </Box>
        );
      }

      return <CodeBlock>{children}</CodeBlock>;
    },
    table: ({ children }: { children?: ReactNode }) => (
      <Box sx={{ width: '100%', overflowX: 'auto', my: 1 }}>
        <Box component="table" sx={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.85rem' }}>
          {children}
        </Box>
      </Box>
    ),
    th: ({ children }: { children?: ReactNode }) => (
      <Box
        component="th"
        sx={{
          textAlign: 'left',
          px: 1,
          py: 0.5,
          border: 1,
          borderColor: 'divider',
          bgcolor: 'action.hover',
          fontWeight: 600,
        }}
      >
        {children}
      </Box>
    ),
    td: ({ children }: { children?: ReactNode }) => (
      <Box
        component="td"
        sx={{
          px: 1,
          py: 0.5,
          border: 1,
          borderColor: 'divider',
          verticalAlign: 'top',
        }}
      >
        {children}
      </Box>
    ),
    a: ({ href, children }: { href?: string; children?: ReactNode }) => {
      const panelKey = href ? parseDashboardLinkToPanelKey(href) : null;
      if (panelKey !== null && panelKeysFromImages.has(panelKey)) return null;

      return (
        <Link
          href={href ? toGrafanaDashboardLink(href) : '#'}
          target="_blank"
          rel="noopener noreferrer"
          sx={{
            fontSize: '0.8125rem',
            color: 'primary.light',
            '&:hover': { color: 'primary.main' },
          }}
        >
          {children}
        </Link>
      );
    },
    img: ({ src, alt }: { src?: string; alt?: string }) => {
      if (src && isGrafanaRenderImageSrc(src)) {
        const imageSrc = toSameOriginUrl(withRenderCacheParam(src));
        const dashboardHref = dashboardUrlFromRenderUrl(src);

        return (
          <GrafanaPanelImage
            src={imageSrc}
            alt={alt ?? 'Grafana panel'}
            dashboardHref={dashboardHref}
          />
        );
      }

      return (
        <Box
          component="img"
          src={src ? toSameOriginUrl(src) : undefined}
          alt={alt ?? ''}
          sx={{ display: 'block', maxWidth: '100%', height: 'auto' }}
        />
      );
    },
  };
}
