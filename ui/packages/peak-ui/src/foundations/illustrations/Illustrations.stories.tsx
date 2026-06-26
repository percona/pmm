import type { Meta, StoryObj } from '@storybook/react';
import { type ComponentType, useMemo, useState } from 'react';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import InputAdornment from '@mui/material/InputAdornment';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import { ContentCopy, Search } from '@mui/icons-material';
import { type SvgIconProps } from '@mui/material/SvgIcon';
import {
  NothingFoundIllustration,
  RadarIllustration,
  RealTimeDatabaseOffIllustration,
  UnknownErrorIllustration,
} from '../../components/illustrations';

interface IllustrationEntry {
  component: ComponentType<SvgIconProps>;
  importName: string;
  designName: string;
  layers: string;
  keywords: string;
}

// Add new Peak Design illustrations here as they are created.
const illustrations: IllustrationEntry[] = [
  {
    component: RealTimeDatabaseOffIllustration,
    importName: 'RealTimeDatabaseOffIllustration',
    designName: 'real-time-database-off',
    layers: 'accent · base · liveBadge',
    keywords: 'realtime real-time database analysis off disabled monitoring live',
  },
  {
    component: RadarIllustration,
    importName: 'RadarIllustration',
    designName: 'radar',
    layers: 'arrays · nodes · wiper',
    keywords: 'radar scan detection objects nodes discovery monitoring',
  },
  {
    component: UnknownErrorIllustration,
    importName: 'UnknownErrorIllustration',
    designName: 'unknown-error',
    layers: 'eyes · mouth · windowFrame · controls · nose',
    keywords: 'unknown error window crash failure generic problem',
  },
  {
    component: NothingFoundIllustration,
    importName: 'NothingFoundIllustration',
    designName: 'nothing-found',
    layers: 'swooshArrays · glass · questionMark',
    keywords: 'nothing found empty search no results magnifier',
  },
];

const IllustrationsGallery = () => {
  const [query, setQuery] = useState('');

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return illustrations;
    return illustrations.filter(
      ({ designName, importName, keywords }) =>
        designName.toLowerCase().includes(q) ||
        importName.toLowerCase().includes(q) ||
        keywords.toLowerCase().includes(q)
    );
  }, [query]);

  const handleCopy = (importName: string) => {
    navigator.clipboard?.writeText(`import { ${importName} } from '@percona/percona-ui';`);
  };

  return (
    <Box sx={{ maxWidth: 934 }}>
      <TextField
        fullWidth
        size="small"
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        placeholder="Search by name or keyword…"
        aria-label="Search illustrations"
        slotProps={{
          input: {
            startAdornment: (
              <InputAdornment position="start">
                <Search fontSize="large" />
              </InputAdornment>
            ),
          },
        }}
        sx={{ mb: 2 }}
      />

      {filtered.length === 0 ? (
        <Typography variant="body1" color="text.primary">
          No illustrations match “{query}”.
        </Typography>
      ) : (
        <Box
          sx={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))',
            gap: 2,
          }}
        >
          {filtered.map(({ component: Illustration, importName, designName, layers }) => (
            <Stack
              key={importName}
              alignItems="center"
              spacing={1.5}
              sx={{
                position: 'relative',
                px: 4,
                py: 3,
                borderRadius: '3px',
                border: '1px solid',
                borderColor: 'dividers.contour',
                textAlign: 'center',
                '&:hover .copy-button': { opacity: 1 },
              }}
            >
              <Illustration sx={{ fontSize: 64, color: 'text.primary' }} />
              <Box>
                <Typography
                  variant="body2"
                  sx={{ fontFamily: '"Roboto Mono", monospace', fontWeight: 500 }}
                  mb={1}
                >
                  {designName}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  {importName}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  layers: {layers}
                </Typography>
              </Box>
              <Tooltip title="Copy import name">
                <IconButton
                  className="copy-button"
                  size="medium"
                  onClick={() => handleCopy(importName)}
                  aria-label={`Copy import statement for ${importName}`}
                  sx={{
                    position: 'absolute',
                    top: 8,
                    right: 12,
                    opacity: 0,
                    transition: 'opacity 120ms ease',
                  }}
                >
                  <ContentCopy />
                </IconButton>
              </Tooltip>
            </Stack>
          ))}
        </Box>
      )}
    </Box>
  );
};

const meta = {
  title: 'Foundations/Illustrations/Illustrations',
  parameters: {
    layout: 'padded',
  },
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const Gallery: Story = {
  tags: ['!dev'],
  parameters: { canvasSurface: 'paper' },
  render: () => <IllustrationsGallery />,
};
