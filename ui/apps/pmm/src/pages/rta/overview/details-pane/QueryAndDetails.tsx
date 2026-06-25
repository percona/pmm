import Grid from '@mui/material/Grid';
import Stack from '@mui/material/Stack';
import Chip from '@mui/material/Chip';
import Typography from '@mui/material/Typography';
import Tooltip from '@mui/material/Tooltip';
import { FC } from 'react';
import { format, formatDuration } from 'date-fns';
import { tz } from '@date-fns/tz';
import { SyntaxHighlighter } from 'components/syntax-highlighter';
import { isPostgresQuery, QueryData } from 'types/rta.types';
import DetailsMetric from './DetailsMetric';
import BigNumberMetric from './BigNumberMetric';
import { Messages } from './QueryAndDetails.messages';
import { TIME_FORMAT } from 'lib/constants';
import { useUser } from 'contexts/user';

type Props = {
  queryData: QueryData;
};

const GridItem = ({ children }: { children: React.ReactNode }) => (
  <Grid size={{ xs: 6 }} sx={{ '& > *': { height: '100%' } }}>
    {children}
  </Grid>
);

const formatElapsed = (queryExecutionDurationMs?: number | null) => {
  if (!queryExecutionDurationMs) {
    return { mainText: undefined, subText: undefined };
  }

  const formatted = formatDuration(
    { seconds: queryExecutionDurationMs },
    { format: ['seconds'] }
  );
  const parts = formatted.split(' ');

  return {
    mainText: parts.length > 1 ? parts[0] : undefined,
    subText: parts.length > 1 ? parts[1] : undefined,
  };
};

const PostgresDetails: FC<Props> = ({ queryData }) => {
  const { user } = useUser();
  const timezone = user?.preferences?.timezone || 'UTC';
  const payload = queryData.postgresPayload!;
  const elapsed = formatElapsed(queryData.queryExecutionDurationMs);
  const isIdleInTransaction = payload.sessionState === 'idle in transaction';

  return (
    <Grid container spacing={3}>
      <Grid size={{ xs: 12, md: 6 }}>
        <Grid container spacing={3}>
          <GridItem>
            <DetailsMetric title="Session state" tooltip="PostgreSQL backend state">
              <Stack direction="row" spacing={1} alignItems="center">
                <BigNumberMetric mainText={payload.sessionState} dataTestId="session-state-value" />
                {isIdleInTransaction && (
                  <Chip label="idle in transaction" color="warning" size="small" />
                )}
              </Stack>
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.operationId} tooltip={Messages.tooltips.operationId}>
              <BigNumberMetric mainText={queryData.queryId} dataTestId="operation-id-value" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.elapsedExecTime} tooltip={Messages.tooltips.elapsedExecTime}>
              <BigNumberMetric {...elapsed} dataTestId="elapsed-time-value" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Wait event" tooltip="PostgreSQL wait event type and name">
              <BigNumberMetric
                mainText={[payload.waitEventType, payload.waitEvent].filter(Boolean).join(' / ') || undefined}
                dataTestId="wait-event-value"
              />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.databaseName} tooltip={Messages.tooltips.databaseName}>
              <BigNumberMetric mainText={payload.databaseName} size="small" dataTestId="database-name-value" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.username} tooltip={Messages.tooltips.username}>
              <BigNumberMetric mainText={payload.username} size="small" dataTestId="username-value" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.clientAppName} tooltip={Messages.tooltips.clientAppName}>
              <BigNumberMetric mainText={payload.applicationName} size="small" dataTestId="client-app-name-value" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.clientAddress} tooltip={Messages.tooltips.clientAddress}>
              <BigNumberMetric mainText={queryData.clientAddress} size="small" dataTestId="client-address-value" />
            </DetailsMetric>
          </GridItem>
          {payload.lockChain && payload.lockChain.length > 0 && (
            <Grid size={{ xs: 12 }}>
              <DetailsMetric title="Lock chain" tooltip="Blocker to blocked lock relationships">
                <Stack spacing={1}>
                  {payload.lockChain.map((link, index) => (
                    <Typography key={`${link.blockerPid}-${index}`} variant="body2">
                      PID {link.blockerPid} blocks {link.blockedPid} ({link.lockMode}
                      {link.relationName ? ` on ${link.relationName}` : ''})
                      {link.blockerQueryText ? `: ${link.blockerQueryText}` : ''}
                    </Typography>
                  ))}
                </Stack>
              </DetailsMetric>
            </Grid>
          )}
          <GridItem>
            <DetailsMetric title={Messages.titles.dataCaptureTime} tooltip={Messages.tooltips.dataCaptureTime}>
              <BigNumberMetric
                mainText={format(new Date(queryData.queryCollectTime), TIME_FORMAT, { in: tz(timezone) })}
                size="small"
                dataTestId="data-capture-time-value"
              />
            </DetailsMetric>
          </GridItem>
        </Grid>
      </Grid>
      <Grid size={{ xs: 12, md: 6 }} sx={{ maxHeight: '70vh', overflow: 'auto' }}>
        {payload.queryTextTruncated && (
          <Tooltip
            title={`Query text truncated at ${payload.trackActivityQuerySize} bytes. Increase track_activity_query_size and restart PostgreSQL to capture longer queries.`}
          >
            <Chip label="Truncated query" color="warning" size="small" sx={{ mb: 1 }} />
          </Tooltip>
        )}
        <SyntaxHighlighter language="sql" showLineNumbers showCopyButton content={queryData.queryText} />
      </Grid>
    </Grid>
  );
};

const QueryAndDetails: FC<Props> = ({ queryData }) => {
  if (isPostgresQuery(queryData)) {
    return <PostgresDetails queryData={queryData} />;
  }

  const {
    queryText,
    queryId,
    queryExecutionDurationMs,
    queryCollectTime,
    serviceName,
    clientAddress,
    mongoDbPayload = {
      planSummary: '',
      databaseName: '',
      collection: '',
      operation: '',
      username: '',
      dbInstanceAddress: '',
      clientAppName: '',
      operationStartTime: '',
    },
  } = queryData;

  const {
    planSummary,
    databaseName,
    collection,
    operation,
    username,
    dbInstanceAddress,
    clientAppName,
    operationStartTime,
  } = mongoDbPayload;

  const { user } = useUser();
  const timezone = user?.preferences?.timezone || 'UTC';
  const elapsed = formatElapsed(queryExecutionDurationMs);

  return (
    <Grid container spacing={3}>
      <Grid size={{ xs: 12, md: 6 }}>
        <Grid container spacing={3}>
          <GridItem>
            <DetailsMetric
              title={Messages.titles.operationId}
              tooltip={Messages.tooltips.operationId}
            >
              <BigNumberMetric mainText={queryId} dataTestId="operation-id-value" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric
              title={Messages.titles.elapsedExecTime}
              tooltip={Messages.tooltips.elapsedExecTime}
            >
              <BigNumberMetric {...elapsed} dataTestId="elapsed-time-value" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric
              title={Messages.titles.dbInstanceAddress}
              tooltip={Messages.tooltips.dbInstanceAddress}
            >
              <BigNumberMetric
                mainText={dbInstanceAddress}
                size="small"
                dataTestId="db-instance-address-value"
              />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric
              title={Messages.titles.clientAddress}
              tooltip={Messages.tooltips.clientAddress}
            >
              <BigNumberMetric
                mainText={clientAddress}
                size="small"
                dataTestId="client-address-value"
              />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric
              title={Messages.titles.databaseName}
              tooltip={Messages.tooltips.databaseName}
            >
              <BigNumberMetric
                mainText={databaseName}
                size="small"
                dataTestId="database-name-value"
              />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric
              title={Messages.titles.service}
              tooltip={Messages.tooltips.service}
            >
              <BigNumberMetric
                mainText={serviceName}
                size="small"
                dataTestId="service-name-value"
              />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric
              title={Messages.titles.username}
              tooltip={Messages.tooltips.username}
            >
              <BigNumberMetric
                mainText={username}
                size="small"
                dataTestId="username-value"
              />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric
              title={Messages.titles.collection}
              tooltip={Messages.tooltips.collection}
            >
              <BigNumberMetric
                mainText={collection}
                size="small"
                dataTestId="collection-value"
              />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric
              title={Messages.titles.operation}
              tooltip={Messages.tooltips.operation}
            >
              <BigNumberMetric
                mainText={operation}
                size="small"
                dataTestId="operation-value"
              />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric
              title={Messages.titles.planSummary}
              tooltip={Messages.tooltips.planSummary}
            >
              <BigNumberMetric
                mainText={planSummary.replace(/,/g, ',\n')}
                props={{
                  mainText: {
                    overflow: 'visible',
                    textOverflow: 'clip',
                    whiteSpace: 'pre',
                  },
                }}
                size="small"
                dataTestId="plan-summary-value"
              />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric
              title={Messages.titles.clientAppName}
              tooltip={Messages.tooltips.clientAppName}
            >
              <BigNumberMetric
                mainText={clientAppName}
                size="small"
                dataTestId="client-app-name-value"
              />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric
              title={Messages.titles.operationStartTime}
              tooltip={Messages.tooltips.operationStartTime}
            >
              <BigNumberMetric
                mainText={format(new Date(operationStartTime), TIME_FORMAT, {
                  in: tz(timezone),
                })}
                size="small"
                dataTestId="operation-start-time-value"
              />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric
              title={Messages.titles.dataCaptureTime}
              tooltip={Messages.tooltips.dataCaptureTime}
            >
              <BigNumberMetric
                mainText={format(new Date(queryCollectTime), TIME_FORMAT, {
                  in: tz(timezone),
                })}
                size="small"
                dataTestId="data-capture-time-value"
              />
            </DetailsMetric>
          </GridItem>
        </Grid>
      </Grid>
      <Grid
        size={{ xs: 12, md: 6 }}
        sx={{
          maxHeight: '70vh',
          overflow: 'auto',
        }}
      >
        <SyntaxHighlighter
          language="mongodb"
          showLineNumbers={true}
          showCopyButton
          content={queryText}
        />
      </Grid>
    </Grid>
  );
};

export default QueryAndDetails;
