import type { Meta, StoryObj } from '@storybook/react';
import { type ComponentType, type ReactNode, useMemo, useState } from 'react';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import InputAdornment from '@mui/material/InputAdornment';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import ContentCopy from '@mui/icons-material/ContentCopy';
import Search from '@mui/icons-material/Search';
import { type SvgIconProps } from '@mui/material/SvgIcon';
import {
  NothingFoundIllustration,
  RadarIllustration,
  RealTimeDatabaseOffIllustration,
  UnknownErrorIllustration,
} from '../../components/illustrations';
import { primitives } from '../../design';

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
            gridTemplateColumns: 'repeat(auto-fill, minmax(320px, 1fr))',
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
              <Illustration sx={{ fontSize: 128, color: 'text.primary' }} />
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

const DemoColumn = ({ label, children }: { label: ReactNode; children: ReactNode }) => (
  <Stack alignItems="center" spacing={2} sx={{ minWidth: 180 }}>
    <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: 120 }}>
      {children}
    </Box>
    <Typography variant="caption" color="text.secondary" sx={{ textAlign: 'center' }}>
      {label}
    </Typography>
  </Stack>
);

const swatch = (token: string, value: string) => (
  <Stack direction="row" spacing={1} alignItems="center" justifyContent="center">
    <Box
      sx={{ width: 12, height: 12, borderRadius: '2px', backgroundColor: value, flexShrink: 0 }}
    />
    <Typography variant="caption" sx={{ fontFamily: '"Roboto Mono", monospace' }}>
      {token}
    </Typography>
  </Stack>
);

export const LayerColoringNothingFound: Story = {
  tags: ['!dev'],
  parameters: { canvasSurface: 'paper' },
  render: () => (
    <Stack
      direction="row"
      spacing={8}
      alignItems="flex-start"
      flexWrap="wrap"
      useFlexGap
      justifyContent="center"
    >
      <DemoColumn label="Default — every layer uses currentColor">
        <NothingFoundIllustration sx={{ fontSize: 120, color: 'text.primary' }} />
      </DemoColumn>
      <DemoColumn
        label={
          <Stack spacing={0.5}>
            {swatch('glass · primary.purple.400', primitives.primary.purple[400])}
            {swatch('questionMark · primitives.primary.black.700', primitives.primary.black[700])}
            {swatch('swooshArrays · primitives.primary.yellow.400', primitives.primary.yellow[400])}
          </Stack>
        }
      >
        <NothingFoundIllustration
          sx={{ fontSize: 120, color: 'text.primary' }}
          layerColors={{
            glass: primitives.primary.purple[400],
            questionMark: primitives.primary.black[700],
            swooshArrays: primitives.primary.yellow[400],
          }}
        />
      </DemoColumn>
    </Stack>
  ),
};

export const LayerColoringUnknownError: Story = {
  tags: ['!dev'],
  parameters: { canvasSurface: 'paper' },
  render: () => (
    <Stack
      direction="row"
      spacing={8}
      alignItems="flex-start"
      flexWrap="wrap"
      useFlexGap
      justifyContent="center"
    >
      <DemoColumn label="Default — every layer uses currentColor">
        <UnknownErrorIllustration sx={{ fontSize: 120, color: 'text.primary' }} />
      </DemoColumn>
      <DemoColumn
        label={
          <Stack spacing={0.5}>
            {swatch('windowFrame · primitives.tech.mysql.400', primitives.tech.mysql[400])}
            {swatch('eyes · mouth · nose · primitives.tech.mysql.700', primitives.tech.mysql[700])}
            {swatch('controls · primitives.tech.mysql.300', primitives.tech.mysql[300])}
          </Stack>
        }
      >
        <UnknownErrorIllustration
          sx={{ fontSize: 120, color: 'text.primary' }}
          layerColors={{
            windowFrame: primitives.tech.mysql[400],
            eyes: primitives.tech.mysql[700],
            mouth: primitives.tech.mysql[700],
            nose: primitives.tech.mysql[700],
            controls: primitives.tech.mysql[300],
          }}
        />
      </DemoColumn>
    </Stack>
  ),
};
