import Grid from "@mui/material/Grid";
import { FC } from "react";
import { SyntaxHighlighter } from 'components/syntax-highlighter';
import { QueryData } from "types/rta.types";
import DetailsMetric from "./DetailsMetric";
import BigNumberMetric from "./BigNumberMetric";

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
    queryExecutionDuration,
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
  }
}) => {
  return (
    <Grid container spacing={3}>
      <Grid item xs={12} md={6}>
        <Grid container spacing={3}>
          <GridItem>
            <DetailsMetric title="Operation ID">
              <BigNumberMetric mainText={queryId} />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Elapsed exec. time">
              <BigNumberMetric mainText={queryExecutionDuration ?? undefined} subText={queryExecutionDuration ? "ms" : undefined} />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Plan summary">
              <BigNumberMetric mainText={planSummary} size="small" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Database name">
              <BigNumberMetric mainText={databaseName} size="small" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Collection">
              <BigNumberMetric mainText={collection} size="small" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Operation">
              <BigNumberMetric mainText={operation} size="small" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="User name">
              <BigNumberMetric mainText={username} size="small" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Client address">
              <BigNumberMetric mainText={clientAddress} size="small" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Service">
              <BigNumberMetric mainText={serviceName} size="small" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Host">
              <BigNumberMetric mainText={dbInstanceAddress} size="small" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Client app name">
              <BigNumberMetric mainText={clientAppName} size="small" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Operation start time">
              <BigNumberMetric mainText={operationStartTime} size="small" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Data capture time">
              <BigNumberMetric mainText={queryCollectTime} size="small" />
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
        <SyntaxHighlighter language="mongodb" showLineNumbers={true} showCopyButton content={queryText} />
      </Grid>
    </Grid>
  );
};

export default QueryAndDetails;