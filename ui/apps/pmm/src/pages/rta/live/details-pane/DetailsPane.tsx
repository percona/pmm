import { FC } from 'react';
import { DetailsPaneProps } from './DetailsPane.types';
import Stack from '@mui/material/Stack';
import CardContent from '@mui/material/CardContent';
import Tabs from '@mui/material/Tabs';
import Tab from '@mui/material/Tab';
import IconButton from '@mui/material/IconButton';
import KeyboardArrowUpOutlinedIcon from '@mui/icons-material/KeyboardArrowUpOutlined';
import KeyboardArrowDownOutlinedIcon from '@mui/icons-material/KeyboardArrowDownOutlined';
import { Icon } from 'components/icon';
import Paper from '@mui/material/Paper';
import { SyntaxHighlighter } from 'components/syntax-highlighter';

const DetailsPane: FC<DetailsPaneProps> = ({
  query,
  onClose,
  onExpand,
  onCollapse,
  expanded,
}) => {
  return (
    <Paper
      variant="outlined"
      sx={(theme) => ({
        flex: 1,
        display: 'flex',
        flexDirection: 'column',
        borderTopLeftRadius: theme.shape.borderRadius * 2,
        borderTopRightRadius: theme.shape.borderRadius * 2,
        borderBottomLeftRadius: 0,
        borderBottomRightRadius: 0,
        justifySelf: 'flex-end',
        overflow: 'hidden',
      })}
    >
      <Stack
        direction="row"
        justifyContent="space-between"
        sx={(theme) => ({
          mx: 2,
          pt: 1,
          borderBottom: `1px solid ${theme.palette.divider}`,
          flexShrink: 0,
        })}
      >
        <Tabs value={0}>
          <Tab value={0} label="Details" />
          <Tab value={1} label="Explain" />
          <Tab value={2} label="Raw data" />
        </Tabs>
        <Stack gap={1} direction="row" alignItems="center">
          <IconButton>
            <KeyboardArrowUpOutlinedIcon />
          </IconButton>
          <IconButton>
            <KeyboardArrowDownOutlinedIcon />
          </IconButton>
          {expanded ? (
            <IconButton onClick={onCollapse}>
              <Icon name="collapse-content" />
            </IconButton>
          ) : (
            <IconButton onClick={onExpand}>
              <Icon name="expand-content" />
            </IconButton>
          )}
          <IconButton onClick={onClose}>
            <Icon name="bottom-panel-close" />
          </IconButton>
        </Stack>
      </Stack>
      {query ? (
        <CardContent
          sx={{
            flexGrow: 1,
            minHeight: 300,
            overflowY: 'auto',
            overflowX: 'hidden',
          }}
        >
          <SyntaxHighlighter language="mongodb" showLineNumbers={true}>
            {query.query}
          </SyntaxHighlighter>
        </CardContent>
      ) : null}
    </Paper>
  );
};

export default DetailsPane;
