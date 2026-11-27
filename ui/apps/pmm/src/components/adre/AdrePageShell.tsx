import { Box, Stack, ThemeProvider, Typography } from '@mui/material';
import { FC, ReactNode } from 'react';
import { Page } from 'components/page';
import { adreColors, headingFont } from './adreAnswerSx';
import { adreTheme } from './adreTheme';

/** A page in the Assistant's dark look: a Poppins title, an optional line under it and actions on the right. */
export const AdrePageShell: FC<{
  title: string;
  subtitle?: ReactNode;
  actions?: ReactNode;
  maxWidth?: number;
  children: ReactNode;
}> = ({ title, subtitle, actions, maxWidth = 1120, children }) => (
  <Page title={title} hideTitle fullWidth footer={null}>
    <ThemeProvider theme={adreTheme}>
      <Box
        sx={{
          flex: 1,
          minHeight: 0,
          overflowY: 'auto',
          bgcolor: adreColors.page,
          color: adreColors.text,
          borderRadius: 1,
        }}
      >
        <Box
          sx={{
            maxWidth,
            mx: 'auto',
            px: { xs: 2, md: 4 },
            py: { xs: 2.5, md: 3.5 },
          }}
        >
          <Stack
            component="header"
            direction="row"
            flexWrap="wrap"
            alignItems="flex-end"
            justifyContent="space-between"
            gap={2}
            sx={{ mb: 3 }}
          >
            <Box sx={{ minWidth: 0 }}>
              <Typography
                component="h1"
                sx={{
                  fontFamily: headingFont,
                  fontSize: { xs: 22, md: 26 },
                  fontWeight: 600,
                  lineHeight: 1.3,
                  color: adreColors.text,
                }}
              >
                {title}
              </Typography>
              {subtitle && (
                <Typography
                  component="div"
                  sx={{ mt: 0.5, fontSize: 14, color: adreColors.secondary }}
                >
                  {subtitle}
                </Typography>
              )}
            </Box>
            {actions && (
              <Stack direction="row" gap={1} flexWrap="wrap">
                {actions}
              </Stack>
            )}
          </Stack>
          {children}
        </Box>
      </Box>
    </ThemeProvider>
  </Page>
);
