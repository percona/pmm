import { Box, Link } from '@mui/material';
import { ReactNode } from 'react';
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
    // react-markdown v9 removed the `inline` prop, so we can no longer branch on it here.
    // Render every <code> with inline styling; the `pre` override below neutralizes this
    // chrome for fenced blocks (whose <code> is nested inside <pre>).
    code: ({ children }: { children?: ReactNode }) => (
      <Box
        component="code"
        sx={{
          px: 0.5,
          borderRadius: 0.5,
          fontFamily: 'Roboto Mono, monospace',
          bgcolor: 'action.hover',
          overflowWrap: 'anywhere',
        }}
      >
        {children}
      </Box>
    ),
    pre: ({ children }: { children?: ReactNode }) => (
      <Box
        component="pre"
        sx={(theme) => ({
          maxWidth: '100%',
          minWidth: 0,
          overflowX: 'auto',
          my: 1,
          py: 1,
          px: 1.5,
          m: 0,
          border: 2,
          borderColor: 'divider',
          borderRadius: Number(theme.shape.borderRadius) / 4,
          bgcolor: theme.palette.mode === 'dark' ? theme.palette.grey[800] : theme.palette.action.hover,
          // Fenced code: undo the inline <code> chrome so the block reads as one preformatted unit.
          '& code': {
            display: 'block',
            p: 0,
            bgcolor: 'transparent',
            borderRadius: 0,
            whiteSpace: 'pre',
            overflowWrap: 'normal',
            fontSize: '0.8125rem',
          },
        })}
      >
        {children}
      </Box>
    ),
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
          overflowWrap: 'anywhere',
          wordBreak: 'break-word',
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
          overflowWrap: 'anywhere',
          wordBreak: 'break-word',
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
