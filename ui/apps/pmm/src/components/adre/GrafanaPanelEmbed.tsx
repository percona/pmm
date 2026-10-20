import Check from '@mui/icons-material/Check';
import ContentCopy from '@mui/icons-material/ContentCopy';
import { Box, IconButton, Link, Stack, Typography } from '@mui/material';
import { FC, ReactNode, useEffect, useRef, useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { adreColors } from './adreAnswerSx';

const HEIGHT = 400;
const COPIED_MS = 1500;

/** The query behind a chart, with a copy button, under the panel. */
const QueryBar: FC<{ query: string }> = ({ query }) => {
  const [copied, setCopied] = useState(false);
  const timer = useRef<number>();
  useEffect(() => () => window.clearTimeout(timer.current), []);
  // The clipboard is missing outside secure contexts.
  const copy = () =>
    Promise.resolve(navigator.clipboard?.writeText(query)).then(
      () => {
        setCopied(true);
        window.clearTimeout(timer.current);
        timer.current = window.setTimeout(() => setCopied(false), COPIED_MS);
      },
      () => undefined
    );

  return (
    <Stack
      direction="row"
      alignItems="flex-start"
      gap={1}
      sx={{
        px: 1.5,
        py: 1,
        border: `1px solid ${adreColors.border}`,
        borderTop: 0,
        borderRadius: '0 0 12px 12px',
        bgcolor: '#0D0C0F',
      }}
    >
      <Box
        sx={{
          flex: 1,
          minWidth: 0,
          maxHeight: 96,
          overflowY: 'auto',
          fontFamily: 'Roboto Mono, monospace',
          fontSize: 12,
          lineHeight: 1.6,
          color: adreColors.secondary,
          whiteSpace: 'pre-wrap',
          overflowWrap: 'anywhere',
        }}
      >
        {query}
      </Box>
      <IconButton
        size="small"
        aria-label={copied ? 'Copied' : 'Copy query'}
        title={copied ? 'Copied' : 'Copy query'}
        onClick={copy}
        sx={{ color: adreColors.secondary, mt: -0.5 }}
      >
        {copied ? (
          <Check sx={{ fontSize: 16 }} />
        ) : (
          <ContentCopy sx={{ fontSize: 16 }} />
        )}
      </IconButton>
    </Stack>
  );
};

/** A Grafana panel embedded live in an answer: its own legend, tooltips and zoom, with its query and a line under it. */
export const GrafanaPanelEmbed: FC<{
  src: string;
  title: string;
  query?: string;
  caption?: string;
  link: ReactNode;
}> = ({ src, title, query, caption, link }) => (
  <Box
    component="figure"
    aria-label={title}
    sx={{ m: 0, my: 1.5, minWidth: 0 }}
  >
    <Box
      component="iframe"
      src={src}
      title={title}
      loading="lazy"
      sx={{
        display: 'block',
        width: '100%',
        height: HEIGHT,
        border: `1px solid ${adreColors.border}`,
        borderRadius: query ? '12px 12px 0 0' : '12px',
        bgcolor: adreColors.card,
      }}
    />
    {query && <QueryBar query={query} />}
    <Stack
      component="figcaption"
      direction="row"
      flexWrap="wrap"
      justifyContent="space-between"
      columnGap={2}
      sx={{ mt: 0.5, fontSize: 12 }}
    >
      <Typography sx={{ fontSize: 12, color: adreColors.muted }}>
        {caption}
      </Typography>
      {link}
    </Stack>
  </Box>
);

const linkSx = { fontSize: 12, color: adreColors.accentText } as const;

/** The link under an embedded panel: a page in PMM (to), or a full view in a new tab (href). */
export const EmbedLink: FC<
  { children: ReactNode } & ({ to: string } | { href: string })
> = ({ children, ...target }) =>
  'to' in target ? (
    <Link component={RouterLink} to={target.to} sx={linkSx}>
      {children}
    </Link>
  ) : (
    <Link
      href={target.href}
      target="_blank"
      rel="noopener noreferrer"
      sx={linkSx}
    >
      {children}
    </Link>
  );
