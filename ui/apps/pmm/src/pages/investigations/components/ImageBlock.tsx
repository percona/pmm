import { Box, Card, CardContent, Typography } from '@mui/material';
import { FC } from 'react';
import type { InvestigationBlock } from 'api/investigations';

type BlockData = {
  url?: string;
  image_url?: string;
  content?: string;
  alt?: string;
  caption?: string;
};

function pickImageURL(block: InvestigationBlock): string {
  const cfg = (block.configJson ?? {}) as BlockData;
  const data = (block.dataJson ?? {}) as BlockData;
  const url = cfg.url || cfg.image_url || data.url || data.image_url || data.content || '';
  return typeof url === 'string' ? url.trim() : '';
}

function isPMMHosted(url: string): boolean {
  if (!url) return false;
  if (url.startsWith('/')) return true;
  try {
    return new URL(url).origin === window.location.origin;
  } catch {
    return false;
  }
}

export const ImageBlock: FC<{ block: InvestigationBlock }> = ({ block }) => {
  const cfg = (block.configJson ?? {}) as BlockData;
  const data = (block.dataJson ?? {}) as BlockData;
  const src = pickImageURL(block);
  const alt = (cfg.alt || data.alt || block.title || 'Investigation image').trim();
  const caption = (cfg.caption || data.caption || '').trim();

  return (
    <Card variant="outlined" sx={{ mb: 2 }}>
      <CardContent>
        {block.title && (
          <Typography variant="subtitle1" fontWeight={600} sx={{ mb: 1 }}>
            {block.title}
          </Typography>
        )}
        {!src ? (
          <Typography variant="body2" color="text.secondary">
            Image block has no URL.
          </Typography>
        ) : !isPMMHosted(src) ? (
          <Typography variant="body2" color="text.secondary">
            Image URL must be hosted by PMM.
          </Typography>
        ) : (
          <Box>
            <Box
              component="img"
              src={src}
              alt={alt}
              loading="lazy"
              sx={{
                maxWidth: '100%',
                width: '100%',
                height: 'auto',
                borderRadius: 1,
                border: 1,
                borderColor: 'divider',
                display: 'block',
              }}
            />
            {caption && (
              <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.75 }}>
                {caption}
              </Typography>
            )}
          </Box>
        )}
      </CardContent>
    </Card>
  );
};
