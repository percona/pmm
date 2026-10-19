import { Box, Link, Stack, Typography } from '@mui/material';
import { FC, ReactNode } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { adreColors } from './adreAnswerSx';

const HEIGHT = 400;

/** A Grafana panel embedded live in an answer: its own legend, tooltips and zoom, with a line under it. */
export const GrafanaPanelEmbed: FC<{
  src: string;
  title: string;
  caption?: string;
  link: ReactNode;
}> = ({ src, title, caption, link }) => (
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
        borderRadius: '12px',
        bgcolor: adreColors.card,
      }}
    />
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
