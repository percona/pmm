import Stack from "@mui/material/Stack";
import Divider from "@mui/material/Divider";
import Typography from "@mui/material/Typography";
import { FC } from "react";

type Props = {
  title: string;
  children: React.ReactNode;
}
const DetailsMetric: FC<Props> = ({ title, children }) => {
  return <Stack>
    <Typography variant="body1" fontWeight="600" mb={1.5}>{title}</Typography>
    {children}
    <Divider sx={{ mt: 1.5 }} />
  </Stack>
};

export default DetailsMetric;