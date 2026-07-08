import Grid from '@mui/material/Grid';
import { FC } from 'react';
import { format, formatDuration } from 'date-fns';
import { tz } from '@date-fns/tz';
import { SyntaxHighlighter } from 'components/syntax-highlighter';
import { QueryData } from 'types/rta.types';
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

const QueryAndDetails: FC<Props> = ({
  queryData: {
    queryText,
    queryId,
    queryExecutionDurationMs,
    queryCollectTime,
    serviceName,
    clientAddress,
    mongoDbPayload: {
      planSummary,
      databaseName,
      collection,
      operation,
      username,
      dbInstanceAddress,
      clientAppName,
      operationStartTime,
    },
  },
}) => {
  const { user } = useUser();
  const timezone = user?.preferences?.timezone || 'UTC';

  const formattedQueryExecutionDuration = queryExecutionDurationMs
    ? formatDuration(
        {
          seconds: queryExecutionDurationMs,
        },
        {
          format: ['seconds'],
        }
      )
    : '';

  const formattedQueryExecutionDurationParts = formattedQueryExecutionDuration
    ? formattedQueryExecutionDuration.split(' ')
    : [];

  return (
    <Grid container spacing={3}>
      <Grid size={{ xs: 12, md: 6 }}>
        <Grid container spacing={3}>
          <GridItem>
            <DetailsMetric
              title={Messages.titles.operationId}
              tooltip={Messages.tooltips.operationId}
            >
              <BigNumberMetric
                mainText={queryId}
                dataTestId="operation-id-value"
              />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric
              title={Messages.titles.elapsedExecTime}
              tooltip={Messages.tooltips.elapsedExecTime}
            >
              <BigNumberMetric
                mainText={
                  formattedQueryExecutionDurationParts.length > 1
                    ? formattedQueryExecutionDurationParts[0]
                    : undefined
                }
                subText={
                  formattedQueryExecutionDurationParts.length > 1
                    ? formattedQueryExecutionDurationParts[1]
                    : undefined
                }
                dataTestId="elapsed-time-value"
              />
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
