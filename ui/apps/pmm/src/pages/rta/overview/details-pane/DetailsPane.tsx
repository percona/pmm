import { FC } from 'react';
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
import Slide from '@mui/material/Slide';
import Divider from '@mui/material/Divider';
import { QueryData } from 'types/rta.types';

interface Props {
  query?: QueryData;
  isFirstQuery: boolean;
  isLastQuery: boolean;
  onClose: () => void;
  onNext: () => void;
  onPrevious: () => void;
}

const DetailsPane: FC<Props> = ({
  query,
  isFirstQuery,
  isLastQuery,
  onClose,
  onNext,
  onPrevious,
}) => (
  <Slide in={!!query} direction="up">
    <Paper
      variant="outlined"
      sx={(theme) => ({
        p: 1,
        px: 2,
        top: 0,
        bottom: theme.spacing(-2),
        width: '100%',
        position: 'absolute',
        zIndex: theme.zIndex.modal,
      })}
    >
      <Stack direction="row" justifyContent="space-between" sx={{}}>
        <Tabs value={0}>
          <Tab value={0} label="Details" />
          <Tab value={1} label="Explain" />
          <Tab value={2} label="Raw data" />
        </Tabs>
        <Stack gap={1} direction="row" alignItems="center">
          <IconButton onClick={onPrevious} disabled={isFirstQuery}>
            <KeyboardArrowUpOutlinedIcon />
          </IconButton>
          <IconButton onClick={onNext} disabled={isLastQuery}>
            <KeyboardArrowDownOutlinedIcon />
          </IconButton>
          <IconButton onClick={onClose}>
            <Icon name="bottom-panel-close" />
          </IconButton>
        </Stack>
      </Stack>
      <Divider sx={{}} />
      {query ? (
        <CardContent
          sx={{
            p: 0,
            pt: 3,
            flexGrow: 1,
            minHeight: 300,
            overflowY: 'auto',
            overflowX: 'hidden',
          }}
        >
          <SyntaxHighlighter language="mongodb" showLineNumbers={true}>
            {query.queryText}
          </SyntaxHighlighter>
        </CardContent>
      ) : null}
    </Paper>
  </Slide>
);

export default DetailsPane;
