import { Box, Link } from '@mui/material';
import {
  Children,
  ReactElement,
  ReactNode,
  cloneElement,
  isValidElement,
} from 'react';
import { GrafanaPanelImage } from './adre-chat-markdown';
import { AdreCodeBlock } from './adre-code-block';
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
    getRenderImageUrlsInContent(content)
      .map(parseRenderImageUrlToPanelKey)
      .filter(Boolean)
  );

  return {
    // Default <p> cannot contain block-level panel UI; browsers break the DOM and width
    // constraints fail, so wide panel images overflow the chat bubble.
    p: ({ children }: { children?: ReactNode }) => (
      <Box
        component="div"
        className="adre-md-p"
        sx={{ mb: 1.25, minWidth: 0, maxWidth: '100%' }}
      >
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
      <AdreCodeBlock>{children}</AdreCodeBlock>
    ),
    // Each item carries its number, so lists split by code blocks keep counting from their start.
    ol: ({ start, children }: { start?: number; children?: ReactNode }) => {
      let n = (start ?? 1) - 1;
      return (
        <ol start={start}>
          {Children.map(children, (child) => {
            if (!isValidElement(child) || child.type !== 'li') return child;
            n += 1;
            return cloneElement(
              child as ReactElement<Record<string, unknown>>,
              { 'data-n': n }
            );
          })}
        </ol>
      );
    },
    table: ({ children }: { children?: ReactNode }) => (
      <Box sx={{ width: '100%', overflowX: 'auto', my: 1 }}>
        <Box
          component="table"
          sx={{
            width: '100%',
            borderCollapse: 'collapse',
            fontSize: '0.85rem',
          }}
        >
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
