import { FC } from "react";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import DetailsMetric from "./DetailsMetric";

const Metric = ({ mainText, subText, tertiaryText }: { mainText: string, subText?: string, tertiaryText?: string }) => (
  <span style={{ display: 'inline-flex', alignItems: 'baseline' }}>
    <Typography variant="h5" fontWeight="600" fontFamily='Roboto Mono, monospace'>{mainText}</Typography>
    {subText && <Typography variant="body2" fontWeight="400" fontFamily='Roboto Mono, monospace' ml={0.5}>{subText}</Typography>}
    {tertiaryText && <Typography variant="body2" color='text.disabled' fontWeight="400" fontFamily='Roboto Mono, monospace' ml={2}>{tertiaryText}</Typography>}
  </span>
)

const HistoricalContext: FC = () => {
  return <Stack>
    <Typography variant="h5" mb={3}>Historical Context</Typography>
    <Stack direction="row" gap={3} mb={1} sx={{ '& > *': { flexBasis: 0, flex: 1 } }}>
      <DetailsMetric title="Max. exec. time">
        <Metric mainText="2421.83" subText="ms" tertiaryText="5.01%" />
      </DetailsMetric>
      <DetailsMetric title="Average exec. time">
        <Metric mainText="638.19" subText="ms" tertiaryText="0.98%" />
      </DetailsMetric>
      <DetailsMetric title="Exec. count">
        <Metric mainText="178" subText="x" />
      </DetailsMetric>
      <DetailsMetric title="Total exec. time">
        <Metric mainText="2" subText="min" tertiaryText="12.72%" />
      </DetailsMetric>
    </Stack>
  </Stack>
};

export default HistoricalContext;