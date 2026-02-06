import { FC } from "react";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import DetailsMetric from "./DetailsMetric";
import BigNumberMetric from "./BigNumberMetric";

const HistoricalContext: FC = () => {
  return <Stack>
    <Typography variant="h5" mb={3}>Historical Context</Typography>
    <Stack direction="row" gap={3} mb={1} sx={{ '& > *': { flexBasis: 0, flex: 1 } }}>
      <DetailsMetric title="Max. exec. time">
        <BigNumberMetric mainText="2421.83" subText="ms" tertiaryText="5.01%" />
      </DetailsMetric>
      <DetailsMetric title="Average exec. time">
        <BigNumberMetric mainText="638.19" subText="ms" tertiaryText="0.98%" />
      </DetailsMetric>
      <DetailsMetric title="Exec. count">
        <BigNumberMetric mainText="178" subText="x" />
      </DetailsMetric>
      <DetailsMetric title="Total exec. time">
        <BigNumberMetric mainText="2" subText="min" tertiaryText="12.72%" />
      </DetailsMetric>
    </Stack>
  </Stack>
};

export default HistoricalContext;