import Stack from "@mui/material/Stack";
import Divider from "@mui/material/Divider";
import Typography from "@mui/material/Typography";
import { FC } from "react";
import { Box } from "@mui/material";

type Props = {
  title: string;
  subtitle?: string;
  children: React.ReactNode;
}
const DetailsMetric: FC<Props> = ({ title, subtitle, children }) => {
  return <Stack>
    <span>
      <Typography variant="body1" fontFamily='Poppins' fontWeight="600" >{title}</Typography>
      {subtitle && <Typography variant="body2" fontFamily='Roboto Mono, monospace' fontWeight="400" color='text.disabled' ml={1}>{subtitle}</Typography>}
    </span>
    <Box py={1.5}>
      {children}
    </Box>
    <Divider sx={{ mt: 'auto' }} />
  </Stack>
};

export default DetailsMetric;