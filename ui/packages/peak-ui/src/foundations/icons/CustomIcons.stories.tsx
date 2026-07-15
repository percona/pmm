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
  BottomPanelCloseIcon,
  BottomPanelOpenIcon,
  CirclesExtIcon,
  ClusterDegradedIcon,
  ClusterHealthyIcon,
  ClusterInoperationalIcon,
  ClusterTransitioningIcon,
  ElectricBoltIcon,
  ElectricBoltOffIcon,
  Graph4Icon,
  Graph5Icon,
  Graph6Icon,
  HaproxyIcon,
  HomeIcon,
  KnowledgeBaseIcon,
  NetworkIntelligenceIcon,
  NetworkNodeIcon,
  PerconaIcon,
  PerconaMoIcon,
  PerconaMyIcon,
  PerconaPoIcon,
  PerconaVaIcon,
  ProxyIcon,
  QueryStatsIcon,
  SearchInsightsIcon,
} from '../../components/icons';

interface CustomIconEntry {
  component: ComponentType<SvgIconProps>;
  importName: string;
  designName: string;
  keywords: string;
}

// Add new Peak Design icons here as they are created.
const customIcons: CustomIconEntry[] = [
  {
    component: PerconaIcon,
    importName: 'PerconaIcon',
    designName: 'percona-icon',
    keywords: 'percona brand logo',
  },
  {
    component: PerconaMyIcon,
    importName: 'PerconaMyIcon',
    designName: 'percona-my',
    keywords: 'mysql my database brand percona',
  },
  {
    component: PerconaMoIcon,
    importName: 'PerconaMoIcon',
    designName: 'percona-mo',
    keywords: 'mongodb mongo database brand percona',
  },
  {
    component: PerconaPoIcon,
    importName: 'PerconaPoIcon',
    designName: 'percona-po',
    keywords: 'postgresql postgres database brand percona',
  },
  {
    component: PerconaVaIcon,
    importName: 'PerconaVaIcon',
    designName: 'percona-va',
    keywords: 'valkey redis database brand percona',
  },
  {
    component: ProxyIcon,
    importName: 'ProxyIcon',
    designName: 'proxy',
    keywords: 'proxy proxysql routing network',
  },
  {
    component: HaproxyIcon,
    importName: 'HaproxyIcon',
    designName: 'haproxy',
    keywords: 'haproxy load balancer proxy network',
  },
  {
    component: ClusterHealthyIcon,
    importName: 'ClusterHealthyIcon',
    designName: 'cluster-healthy',
    keywords: 'cluster healthy status nodes up topology',
  },
  {
    component: ClusterDegradedIcon,
    importName: 'ClusterDegradedIcon',
    designName: 'cluster-degraded',
    keywords: 'cluster degraded status warning topology',
  },
  {
    component: ClusterInoperationalIcon,
    importName: 'ClusterInoperationalIcon',
    designName: 'cluster-inoperational',
    keywords: 'cluster inoperational down error status topology',
  },
  {
    component: ClusterTransitioningIcon,
    importName: 'ClusterTransitioningIcon',
    designName: 'cluster-transitioning',
    keywords: 'cluster transitioning updating status topology',
  },
  {
    component: HomeIcon,
    importName: 'HomeIcon',
    designName: 'home',
    keywords: 'home house dashboard start main landing',
  },
  {
    component: KnowledgeBaseIcon,
    importName: 'KnowledgeBaseIcon',
    designName: 'knowledge-base',
    keywords: 'knowledge base book docs documentation library guide help',
  },
  {
    component: BottomPanelOpenIcon,
    importName: 'BottomPanelOpenIcon',
    designName: 'bottom-panel-open',
    keywords: 'bottom panel open expand layout show drawer terminal',
  },
  {
    component: BottomPanelCloseIcon,
    importName: 'BottomPanelCloseIcon',
    designName: 'bottom-panel-close',
    keywords: 'bottom panel close collapse layout hide drawer terminal',
  },
  {
    component: Graph4Icon,
    importName: 'Graph4Icon',
    designName: 'graph-4',
    keywords: 'graph molecule nodes network connections topology data',
  },
  {
    component: Graph5Icon,
    importName: 'Graph5Icon',
    designName: 'graph-5',
    keywords: 'graph nodes network connections diagram topology data',
  },
  {
    component: Graph6Icon,
    importName: 'Graph6Icon',
    designName: 'graph-6',
    keywords: 'graph nodes network connections cluster topology data',
  },
  {
    component: CirclesExtIcon,
    importName: 'CirclesExtIcon',
    designName: 'circles-ext',
    keywords: 'circles connected nodes group cluster relationship',
  },
  {
    component: NetworkNodeIcon,
    importName: 'NetworkNodeIcon',
    designName: 'network-node',
    keywords: 'network node topology connection tree hierarchy',
  },
  {
    component: NetworkIntelligenceIcon,
    importName: 'NetworkIntelligenceIcon',
    designName: 'network-intelligence',
    keywords: 'network intelligence routing smart connections',
  },
  {
    component: QueryStatsIcon,
    importName: 'QueryStatsIcon',
    designName: 'query-stats',
    keywords: 'query stats analytics chart search metrics performance',
  },
  {
    component: SearchInsightsIcon,
    importName: 'SearchInsightsIcon',
    designName: 'search-insights',
    keywords: 'search insights analytics bars chart magnify explore',
  },
  {
    component: ElectricBoltIcon,
    importName: 'ElectricBoltIcon',
    designName: 'electric-bolt',
    keywords: 'electric bolt lightning energy power fast flash',
  },
  {
    component: ElectricBoltOffIcon,
    importName: 'ElectricBoltOffIcon',
    designName: 'electric-bolt-off',
    keywords: 'electric bolt off lightning disabled no power flash off',
  },
];

const CustomIconsGallery = () => {
  const [query, setQuery] = useState('');

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return customIcons;
    return customIcons.filter(
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
        aria-label="Search custom icons"
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
          No custom icons match “{query}”.
        </Typography>
      ) : (
        <Box
          sx={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(192px, 1fr))',
            gap: 2,
          }}
        >
          {filtered.map(({ component: Icon, importName, designName }) => (
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
              <Icon sx={{ fontSize: 32, color: 'text.primary' }} />
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
  title: 'Foundations/Icons/Custom Icons',
  tags: ['stable'],
  parameters: {
    layout: 'padded',
  },
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const Gallery: Story = {
  tags: ['!dev'],
  parameters: { canvasSurface: 'paper' },
  render: () => <CustomIconsGallery />,
};
