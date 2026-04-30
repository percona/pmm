import { FC, ReactNode } from 'react';
import { format } from 'date-fns';
import { tz } from '@date-fns/tz';
import {
  Chip,
  Grid,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';
import { useUser } from 'contexts/user';
import { TIME_FORMAT } from 'lib/constants';
import { SyntaxHighlighter } from 'components/syntax-highlighter';
import BigNumberMetric from 'pages/rta/overview/details-pane/BigNumberMetric';
import DetailsMetric from 'pages/rta/overview/details-pane/DetailsMetric';
import {
  STATUS_COLOR_MAP,
  STATUS_LABEL_MAP,
} from '../AlertsPage.constants';
import { AlertRow } from '../AlertsPage.types';

type Props = {
  alert: AlertRow;
};

type GridItemProps = {
  children: ReactNode;
  size?: {
    xs: number;
    md?: number;
  };
};

const GridItem = ({ children, size = { xs: 6 } }: GridItemProps) => (
  <Grid size={size} sx={{ '& > *': { height: '100%' } }}>
    {children}
  </Grid>
);

const formatTimestamp = (timestamp: string | undefined, timezone: string) => {
  if (!timestamp) {
    return undefined;
  }

  const date = new Date(timestamp);

  if (Number.isNaN(date.getTime())) {
    return undefined;
  }

  return format(date, TIME_FORMAT, { in: tz(timezone) });
};

const KeyValueTable = ({
  title,
  data,
}: {
  title: string;
  data: Record<string, string>;
}) => {
  const entries = Object.entries(data);

  return (
    <TableContainer component={Paper} variant="outlined">
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell colSpan={2}>
              <Typography variant="body1" fontFamily="Poppins" fontWeight="600">
                {title}
              </Typography>
            </TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {entries.length ? (
            entries.map(([key, value]) => (
              <TableRow key={key}>
                <TableCell
                  component="th"
                  scope="row"
                  sx={{ width: '35%', fontFamily: 'Roboto Mono, monospace' }}
                >
                  {key}
                </TableCell>
                <TableCell sx={{ fontFamily: 'Roboto Mono, monospace' }}>
                  {value}
                </TableCell>
              </TableRow>
            ))
          ) : (
            <TableRow>
              <TableCell colSpan={2}>No {title.toLowerCase()}.</TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </TableContainer>
  );
};

const AlertDetails: FC<Props> = ({ alert }) => {
  const { user } = useUser();
  const timezone = user?.preferences?.timezone || 'UTC';

  return (
    <Grid container spacing={3}>
      <Grid size={{ xs: 12, md: 6 }}>
        <Grid container spacing={3}>
          <GridItem>
            <DetailsMetric title="State">
              <Chip
                label={STATUS_LABEL_MAP[alert.state]}
                color={STATUS_COLOR_MAP[alert.state]}
                size="small"
              />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Alert">
              <BigNumberMetric
                mainText={alert.alertName}
                size="small"
                dataTestId="alert-name-value"
              />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Rule">
              <BigNumberMetric
                mainText={alert.ruleName}
                size="small"
                dataTestId="rule-name-value"
              />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Node">
              <BigNumberMetric
                mainText={alert.nodeId}
                size="small"
                dataTestId="node-id-value"
              />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Service">
              <BigNumberMetric
                mainText={alert.serviceName}
                size="small"
                dataTestId="service-name-value"
              />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Source">
              <BigNumberMetric
                mainText={alert.source}
                size="small"
                dataTestId="source-value"
              />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Age">
              <BigNumberMetric
                mainText={alert.age}
                size="small"
                dataTestId="age-value"
              />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Active since">
              <BigNumberMetric
                mainText={formatTimestamp(alert.activeAt, timezone)}
                size="small"
                dataTestId="active-since-value"
              />
            </DetailsMetric>
          </GridItem>
          <GridItem size={{ xs: 12 }}>
            <DetailsMetric title="Summary">
              <BigNumberMetric
                mainText={alert.summary}
                size="small"
                props={{
                  mainText: {
                    overflow: 'visible',
                    textOverflow: 'clip',
                    whiteSpace: 'pre-wrap',
                  },
                }}
                dataTestId="summary-value"
              />
            </DetailsMetric>
          </GridItem>
        </Grid>
      </Grid>
      <Grid
        size={{ xs: 12, md: 6 }}
        sx={{
          display: 'flex',
          flexDirection: 'column',
          gap: 2,
          maxHeight: '70vh',
          overflow: 'auto',
        }}
      >
        <SyntaxHighlighter
          language="text"
          showLineNumbers
          showCopyButton
          content={alert.expression}
        />
        <KeyValueTable title="Labels" data={alert.labels} />
        <KeyValueTable title="Annotations" data={alert.annotations} />
      </Grid>
    </Grid>
  );
};

export default AlertDetails;
