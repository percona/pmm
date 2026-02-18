import Grid from '@mui/material/Grid';
import { FC } from 'react';
import { format } from 'date-fns';
import { SyntaxHighlighter } from 'components/syntax-highlighter';
import { QueryData } from 'types/rta.types';
import DetailsMetric from './DetailsMetric';
import BigNumberMetric from './BigNumberMetric';
import { Messages } from './QueryAndDetails.messages';
import formatDuration from 'date-fns/formatDuration';
import { TIME_FORMAT } from 'lib/constants';

type Props = {
  queryData: QueryData;
};

const GridItem = ({ children }: { children: React.ReactNode }) => (
  <Grid item xs={6} sx={{ '& > *': { height: '100%' } }}>
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
      <Grid item xs={12} md={6}>
        <Grid container spacing={3}>
          <GridItem>
            <DetailsMetric title={Messages.titles.operationId}>
              <BigNumberMetric mainText={queryId} />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.elapsedExecTime}>
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
              />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.dbInstanceAddress}>
              <BigNumberMetric mainText={dbInstanceAddress} size="small" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.clientAddress}>
              <BigNumberMetric mainText={clientAddress} size="small" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.databaseName}>
              <BigNumberMetric mainText={databaseName} size="small" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.service}>
              <BigNumberMetric mainText={serviceName} size="small" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.username}>
              <BigNumberMetric mainText={username} size="small" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.collection}>
              <BigNumberMetric mainText={collection} size="small" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.operation}>
              <BigNumberMetric mainText={operation} size="small" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.planSummary}>
              <BigNumberMetric mainText={planSummary.replace(/,/g, ',\n')} props={{
                mainText: {
                  overflow: 'visible',
                  textOverflow: 'clip',
                  whiteSpace: 'pre',
                }
              }} size="small" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.clientAppName}>
              <BigNumberMetric mainText={clientAppName} size="small" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.operationStartTime}>
              <BigNumberMetric
                mainText={format(new Date(operationStartTime), TIME_FORMAT)}
                size="small"
              />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title={Messages.titles.dataCaptureTime}>
              <BigNumberMetric
                mainText={format(new Date(queryCollectTime), TIME_FORMAT)}
                size="small"
              />
            </DetailsMetric>
          </GridItem>
        </Grid>
      </Grid>
      <Grid
        item
        xs={12}
        md={6}
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
