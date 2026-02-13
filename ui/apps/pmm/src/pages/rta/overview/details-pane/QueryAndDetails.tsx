import Grid from "@mui/material/Grid";
import { FC } from "react";
import { SyntaxHighlighter } from 'components/syntax-highlighter';
import { QueryData } from "types/rta.types";
import DetailsMetric from "./DetailsMetric";
import Typography from "@mui/material/Typography";
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
              <BigNumberMetric mainText={queryId} size="small" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Elapsed exec. time">
              <BigNumberMetric mainText={queryExecutionDuration ?? undefined} subText="ms" />
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Plan summary">
              <Typography variant="body2">{planSummary}</Typography>
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Database name">
              <Typography variant="body2">{databaseName}</Typography>
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Collection">
              <Typography variant="body2">{collection}</Typography>
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Operation">
              <Typography variant="body2">{operation}</Typography>
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="User name">
              <Typography variant="body2">{username}</Typography>
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Client address">
              <Typography variant="body2">{clientAddress}</Typography>
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Service">
              <Typography variant="body2">{serviceName}</Typography>
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Host">
              <Typography variant="body2">{dbInstanceAddress}</Typography>
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Client app name">
              <Typography variant="body2">{clientAppName}</Typography>
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Operation start time">
              <Typography variant="body2">{operationStartTime}</Typography>
            </DetailsMetric>
          </GridItem>
          <GridItem>
            <DetailsMetric title="Data capture time">
              <Typography variant="body2">{queryCollectTime}</Typography>
            </DetailsMetric>
          </GridItem>
        </Grid>
      </Grid>
      <Grid item xs={12} md={6} sx={{
        mt: {
          xs: 4,
          md: 0,
        }
      }}>
        <SyntaxHighlighter language="mongodb" showLineNumbers={true} showCopyButton content={queryText} maxHeight="70vh" />
      </Grid>
    </Grid>
  );
};

export default QueryAndDetails;