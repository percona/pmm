import Stack from "@mui/material/Stack";
import Grid from "@mui/material/Grid";
import { FC } from "react";
import { SyntaxHighlighter } from 'components/syntax-highlighter';
import { QueryData } from "types/rta.types";
import DetailsMetric from "./DetailsMetric";
import Typography from "@mui/material/Typography";
import { StateCell } from "pages/rta/components/state-cell";

type Props = {
  query: QueryData;
};

const GridItem = ({ children }: { children: React.ReactNode }) => (
  <Grid item xs={12} md={6} sx={{ '& > *': { height: '100%' } }}>
    {children}
  </Grid>
);

const QueryAndDetails: FC<Props> = ({ query: { queryText, state, serviceName } }) => {
  return (
    <Stack direction="row" justifyContent="space-between" gap={3} sx={{
      '& > *': {
        flexBasis: 0,
        flex: 1,
      },
    }}>
      <SyntaxHighlighter language="mongodb" showLineNumbers={true} showCopyButton content={queryText} />
      <Grid container spacing={3}>
        <GridItem>
          <DetailsMetric title="Current state">
            <StateCell state={state} />
          </DetailsMetric>
        </GridItem>
        <GridItem>
          <DetailsMetric title="Elapsed exec. time" subtitle="secs_running">
            <Typography>20s</Typography>
          </DetailsMetric>
        </GridItem>
        <GridItem>
          <DetailsMetric title="Plan summary">
            <Typography>Full collection scan (COLLSCAN)</Typography>
          </DetailsMetric>
        </GridItem>
        <GridItem>
          <DetailsMetric title="Docs examined/sent">
            <Typography>84,291/1</Typography>
          </DetailsMetric>
        </GridItem>
        <GridItem>
          <DetailsMetric title="Snapshot time">
            <Typography>2025-10-17 11:18:29</Typography>
          </DetailsMetric>
        </GridItem>
        <GridItem>
          <DetailsMetric title="Operation ID" subtitle="opid">
            <Typography>1238912</Typography>
          </DetailsMetric>
        </GridItem>
        <GridItem>
          <DetailsMetric title="Service">
            <Typography>{serviceName}</Typography>
          </DetailsMetric>
        </GridItem>
      </Grid>
    </Stack>
  );
};

export default QueryAndDetails;