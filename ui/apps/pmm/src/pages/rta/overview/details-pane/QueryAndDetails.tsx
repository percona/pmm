import Stack from "@mui/material/Stack";
import Grid from "@mui/material/Grid";
import { FC } from "react";
import { SyntaxHighlighter } from 'components/syntax-highlighter';
import { QueryData } from "types/rta.types";
import DetailsMetric from "./DetailsMetric";
import Typography from "@mui/material/Typography";

type Props = {
  query: QueryData;
};

const QueryAndDetails: FC<Props> = ({ query: { queryText } }) => {
  return (
    <Stack direction="row" justifyContent="space-between" gap={3} sx={{
      '& > *': {
        flexBasis: 0,
        flex: 1,
      },
    }}>
      {/* <Typography variant="h6">{query.serviceName}</Typography>
      <Typography variant="body2">{query.queryId}</Typography>
      <Typography variant="body2">{query.state}</Typography> */}
      <SyntaxHighlighter language="mongodb" showLineNumbers={true}>
        {queryText}
      </SyntaxHighlighter>
      <Grid container spacing={3}>
        <Grid item xs={12} md={6}>
          <DetailsMetric title="Current state">
            <Typography>Running</Typography>
          </DetailsMetric>
        </Grid>
        <Grid item xs={12} md={6}>
          <DetailsMetric title="Elapsed exec. time">
            <Typography>20s</Typography>
          </DetailsMetric>
        </Grid>
        <Grid item xs={12} md={6}>
          <DetailsMetric title="Plan summary">
            <Typography>Full collection scan (COLLSCAN)</Typography>
          </DetailsMetric>
        </Grid>
        <Grid item xs={12} md={6}>
          <DetailsMetric title="Docs examined/sent">
            <Typography>84,291/1</Typography>
          </DetailsMetric>
        </Grid>
        <Grid item xs={12} md={6}>
          <DetailsMetric title="Snapshot time">
            <Typography>2025-10-17 11:18:29</Typography>
          </DetailsMetric>
        </Grid>
        <Grid item xs={12} md={6}>
          <DetailsMetric title="Operation ID">
            <Typography>1238912</Typography>
          </DetailsMetric>
        </Grid>
        <Grid item xs={12} md={6}>
          <DetailsMetric title="Service">
            <Typography>mc-ga-s01-primary</Typography>
          </DetailsMetric>
        </Grid>
      </Grid>
    </Stack>
  );
};

export default QueryAndDetails;