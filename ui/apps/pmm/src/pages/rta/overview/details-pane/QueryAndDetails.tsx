import Grid from '@mui/material/Grid';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { FC } from 'react';
import { format, formatDuration } from 'date-fns';
import { tz } from '@date-fns/tz';
import { SyntaxHighlighter } from 'components/syntax-highlighter';
import { QueryData, isPostgreSQLQuery } from 'types/rta.types';
import DetailsMetric from './DetailsMetric';
import BigNumberMetric from './BigNumberMetric';
import { Messages } from './QueryAndDetails.messages';
import { TIME_FORMAT } from 'lib/constants';
import { useUser } from 'contexts/user';
import { parseDuration } from 'utils/duration.utils';

type Props = {
  queryData: QueryData;
};

const GridItem = ({ children }: { children: React.ReactNode }) => (
  <Grid size={{ xs: 6 }} sx={{ '& > *': { height: '100%' } }}>
    {children}
  </Grid>
);

const formatSeconds = (seconds?: number | null) => {
  if (!seconds) {
    return { mainText: undefined, subText: undefined };
  }

  const formatted = formatDuration({ seconds }, { format: ['seconds'] });
  const parts = formatted.split(' ');

  return {
    mainText: parts.length > 1 ? parts[0] : undefined,
    subText: parts.length > 1 ? parts[1] : formatted,
  };
};

const MongoDBDetails: FC<{ queryData: QueryData; timezone: string }> = ({
  queryData: {
    queryText,
    queryId,
    queryExecutionDurationMs,
    queryCollectTime,
    serviceName,
    clientAddress,
    mongoDbPayload,
  },
  timezone,
}) => {
  if (!mongoDbPayload) {
    return null;
  }

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

  const elapsed = formatSeconds(queryExecutionDurationMs);

  return (
    <Grid container spacing={3}>
      <Grid size={{ xs: 12, md: 6 }}>
        <Grid container spacing={3}>
          <GridItem>
            <DetailsMetric title={Messages.titles.operationId} tooltip={Messages.tooltips.operationId}>
              <BigNumberMetric mainText={queryId} dataTestId="operation-id-value" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.elapsedExecTime} tooltip={Messages.tooltips.elapsedExecTime}>
              <BigNumberMetric
                mainText={elapsed.mainText}
                subText={elapsed.subText}
                dataTestId="elapsed-time-value"
              />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.dbInstanceAddress} tooltip={Messages.tooltips.dbInstanceAddress}>
              <BigNumberMetric mainText={dbInstanceAddress} size="small" dataTestId="db-instance-address-value" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.clientAddress} tooltip={Messages.tooltips.clientAddress}>
              <BigNumberMetric mainText={clientAddress} size="small" dataTestId="client-address-value" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.databaseName} tooltip={Messages.tooltips.databaseName}>
              <BigNumberMetric mainText={databaseName} size="small" dataTestId="database-name-value" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.service} tooltip={Messages.tooltips.service}>
              <BigNumberMetric mainText={serviceName} size="small" dataTestId="service-name-value" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.username} tooltip={Messages.tooltips.username}>
              <BigNumberMetric mainText={username} size="small" dataTestId="username-value" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.collection} tooltip={Messages.tooltips.collection}>
              <BigNumberMetric mainText={collection} size="small" dataTestId="collection-value" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.operation} tooltip={Messages.tooltips.operation}>
              <BigNumberMetric mainText={operation} size="small" dataTestId="operation-value" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.planSummary} tooltip={Messages.tooltips.planSummary}>
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
            <DetailsMetric title={Messages.titles.clientAppName} tooltip={Messages.tooltips.clientAppName}>
              <BigNumberMetric mainText={clientAppName} size="small" dataTestId="client-app-name-value" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.operationStartTime} tooltip={Messages.tooltips.operationStartTime}>
              <BigNumberMetric
                mainText={format(new Date(operationStartTime), TIME_FORMAT, { in: tz(timezone) })}
                size="small"
                dataTestId="operation-start-time-value"
              />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.dataCaptureTime} tooltip={Messages.tooltips.dataCaptureTime}>
              <BigNumberMetric
                mainText={format(new Date(queryCollectTime), TIME_FORMAT, { in: tz(timezone) })}
                size="small"
                dataTestId="data-capture-time-value"
              />
            </DetailsMetric>
          </GridItem>
        </Grid>
      </Grid>
      <Grid size={{ xs: 12, md: 6 }} sx={{ maxHeight: '70vh', overflow: 'auto' }}>
        <SyntaxHighlighter language="mongodb" showLineNumbers showCopyButton content={queryText} />
      </Grid>
    </Grid>
  );
};

const PostgreSQLDetails: FC<{ queryData: QueryData; timezone: string }> = ({
  queryData,
  timezone,
}) => {
  const payload = queryData.postgresqlPayload;
  if (!payload) {
    return null;
  }

  const elapsed = formatSeconds(
    queryData.transactionDurationMs ?? queryData.queryExecutionDurationMs
  );

  return (
    <Grid container spacing={3}>
      <Grid size={{ xs: 12, md: 6 }}>
        <Grid container spacing={3}>
          <GridItem>
            <DetailsMetric title={Messages.titles.operationId} tooltip={Messages.tooltips.operationId}>
              <BigNumberMetric mainText={queryData.queryId} dataTestId="operation-id-value" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric
              title={payload.state.includes('idle in transaction') ? Messages.titles.transactionDuration : Messages.titles.elapsedExecTime}
              tooltip={Messages.tooltips.elapsedExecTime}
            >
              <BigNumberMetric mainText={elapsed.mainText} subText={elapsed.subText} dataTestId="elapsed-time-value" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.pid} tooltip={Messages.tooltips.pid}>
              <BigNumberMetric mainText={String(payload.pid)} size="small" dataTestId="pid-value" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.state} tooltip={Messages.tooltips.state}>
              <BigNumberMetric mainText={payload.state} size="small" dataTestId="state-value" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.waitEvent} tooltip={Messages.tooltips.waitEvent}>
              <BigNumberMetric
                mainText={[payload.waitEventType, payload.waitEvent].filter(Boolean).join(' / ')}
                size="small"
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
            <DetailsMetric title={Messages.titles.service} tooltip={Messages.tooltips.service}>
              <BigNumberMetric mainText={queryData.serviceName} size="small" dataTestId="service-name-value" />
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
        {(payload.lockChain?.length ?? 0) > 0 && (
          <Stack spacing={1} sx={{ mt: 3 }} data-testid="lock-chain-panel">
            <Typography variant="subtitle2">{Messages.titles.lockChain}</Typography>
            {payload.lockChain?.map((link) => (
              <Stack key={`${link.pid}-${link.lockMode}`} spacing={0.5} sx={{ pl: 1, borderLeft: 2, borderColor: 'error.main' }}>
                <Typography variant="body2">
                  {Messages.lockChainEntry(link.pid, link.lockMode, link.lockType)}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  {link.queryText}
                </Typography>
                {link.duration && (
                  <Typography variant="caption" color="text.secondary">
                    {formatDuration({ seconds: parseDuration(link.duration) / 1000 }, { format: ['seconds'] })}
                  </Typography>
                )}
              </Stack>
            ))}
          </Stack>
        )}
      </Grid>
      <Grid size={{ xs: 12, md: 6 }} sx={{ maxHeight: '70vh', overflow: 'auto' }}>
        <SyntaxHighlighter language="text" showLineNumbers showCopyButton content={queryData.queryText} />
      </Grid>
    </Grid>
  );
};

const QueryAndDetails: FC<Props> = ({ queryData }) => {
  const { user } = useUser();
  const timezone = user?.preferences?.timezone || 'UTC';

  if (isPostgreSQLQuery(queryData)) {
    return <PostgreSQLDetails queryData={queryData} timezone={timezone} />;
  }

  return <MongoDBDetails queryData={queryData} timezone={timezone} />;
};

export default QueryAndDetails;
