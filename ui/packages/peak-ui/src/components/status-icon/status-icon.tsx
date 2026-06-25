// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
import { useTheme } from '@mui/material/styles';
import Box, { type BoxProps } from '@mui/material/Box';

export type StatusIconSeverity = 'success' | 'info' | 'indeterminate' | 'warning' | 'error';

export type StatusIconSize = 'medium' | 'small';

export interface StatusIconProps extends Omit<BoxProps, 'color'> {
  /** Semantic status. Determines both the color and the (locked) glyph. */
  severity: StatusIconSeverity;
  /** medium = 32px (symbol inside a tinted circular wrapper), small = 22px (bare symbol). */
  size?: StatusIconSize;
  /** Elevation surface the icon sits on; backs the small mask and medium base so translucent surfaces composite. Defaults to theme paper. */
  surfaceColor?: string;
}

type PaletteKey = 'success' | 'info' | 'warning' | 'error' | 'neutral';

interface SymbolSpec {
  paletteKey: PaletteKey;
  viewBox: string;
  /** Material Symbol with the glyph as a transparent cutout. */
  path: string;
  /** Solid outer shape (no glyph) used as the small-variant knock-out mask. */
  maskPath: string;
  /** Symbol box inside the 32px medium wrapper. */
  medium: { width: number; height: number; offsetY: number };
  /** Bare symbol box for the 22px small variant. */
  small: { width: number; height: number; offsetY: number };
}

// Solid circle (cx10 cy10 r10) for the round severities' knock-out mask.
const CIRCLE_MASK = 'M0 10A10 10 0 1 1 20 10A10 10 0 1 1 0 10Z';

// Outward offset of the small-variant mask, in viewBox units. Every symbol
// renders at 0.833px per unit, so this yields a ~1.5px surface-colored border
// (a stroke extends half its width outward, hence 2x the offset).
const MASK_BORDER = 3.6;

const SYMBOLS: Record<StatusIconSeverity, SymbolSpec> = {
  success: {
    paletteKey: 'success',
    viewBox: '0 0 20 20',
    path: 'M10 0C11.3833 1.35726e-07 12.6837 0.262109 13.9004 0.787109C15.117 1.3121 16.1752 2.02485 17.0752 2.9248C17.9751 3.82476 18.6879 4.88302 19.2129 6.09961C19.7379 7.31628 20 8.61667 20 10C20 11.3833 19.7379 12.6837 19.2129 13.9004C18.6879 15.117 17.9751 16.1752 17.0752 17.0752C16.1752 17.9751 15.117 18.6879 13.9004 19.2129C12.6837 19.7379 11.3833 20 10 20C8.61667 20 7.31628 19.7379 6.09961 19.2129C4.88302 18.6879 3.82476 17.9751 2.9248 17.0752C2.02485 16.1752 1.3121 15.117 0.787109 13.9004C0.262109 12.6837 1.35728e-07 11.3833 0 10C0 8.61667 0.262109 7.31628 0.787109 6.09961C1.3121 4.88302 2.02485 3.82476 2.9248 2.9248C3.82476 2.02485 4.88302 1.3121 6.09961 0.787109C7.31628 0.262109 8.61667 0 10 0ZM8.59961 11.7998L5.75 8.9502L4.34961 10.3496L8.59961 14.5996L15.6504 7.5498L14.25 6.15039L8.59961 11.7998Z',
    maskPath: CIRCLE_MASK,
    medium: { width: 20, height: 20, offsetY: 0 },
    small: { width: 16.666, height: 16.666, offsetY: 0 },
  },
  info: {
    paletteKey: 'info',
    viewBox: '0 0 20 20',
    path: 'M10 0C11.3833 1.35726e-07 12.6837 0.262109 13.9004 0.787109C15.117 1.3121 16.1752 2.02485 17.0752 2.9248C17.9751 3.82476 18.6879 4.88302 19.2129 6.09961C19.7379 7.31628 20 8.61667 20 10C20 11.3833 19.7379 12.6837 19.2129 13.9004C18.6879 15.117 17.9751 16.1752 17.0752 17.0752C16.1752 17.9751 15.117 18.6879 13.9004 19.2129C12.6837 19.7379 11.3833 20 10 20C8.61667 20 7.31628 19.7379 6.09961 19.2129C4.88302 18.6879 3.82476 17.9751 2.9248 17.0752C2.02485 16.1752 1.3121 15.117 0.787109 13.9004C0.262109 12.6837 1.35728e-07 11.3833 0 10C0 8.61667 0.262109 7.31628 0.787109 6.09961C1.3121 4.88302 2.02485 3.82476 2.9248 2.9248C3.82476 2.02485 4.88302 1.3121 6.09961 0.787109C7.31628 0.262109 8.61667 0 10 0ZM9 15H11V9H9V15ZM10 5C9.71667 5 9.47878 5.09544 9.28711 5.28711C9.09544 5.47878 9 5.71667 9 6C9 6.28333 9.09544 6.52122 9.28711 6.71289C9.47878 6.90456 9.71667 7 10 7C10.2833 7 10.5212 6.90456 10.7129 6.71289C10.9046 6.52122 11 6.28333 11 6C11 5.71667 10.9046 5.47878 10.7129 5.28711C10.5212 5.09544 10.2833 5 10 5Z',
    maskPath: CIRCLE_MASK,
    medium: { width: 20, height: 20, offsetY: 0 },
    small: { width: 16.666, height: 16.666, offsetY: 0 },
  },
  indeterminate: {
    paletteKey: 'neutral',
    viewBox: '0 0 20 20',
    path: 'M10 0C11.3833 1.20569e-07 12.6837 0.262443 13.9004 0.787109C15.1169 1.31243 16.1753 2.0249 17.0752 2.9248C17.9751 3.82472 18.6873 4.88309 19.2119 6.09961C19.7372 7.31628 20 8.61667 20 10C20 11.3833 19.7372 12.6837 19.2119 13.9004C18.6873 15.1169 17.9751 16.1753 17.0752 17.0752C16.1753 17.9751 15.1169 18.6873 13.9004 19.2119C12.6837 19.7372 11.3833 20 10 20C8.61667 20 7.31628 19.7372 6.09961 19.2119C4.88309 18.6873 3.82472 17.9751 2.9248 17.0752C2.02489 16.1753 1.31274 15.1169 0.788086 13.9004C0.262753 12.6837 1.20724e-07 11.3833 0 10C0 8.61667 0.262753 7.31628 0.788086 6.09961C1.31274 4.88309 2.02489 3.82472 2.9248 2.9248C3.82471 2.0249 4.8831 1.31243 6.09961 0.787109C7.31628 0.262443 8.61667 0 10 0ZM9.9502 13.5C9.6002 13.5 9.30352 13.6213 9.06152 13.8633C8.82044 14.1045 8.7002 14.4002 8.7002 14.75C8.7002 15.0998 8.82044 15.3955 9.06152 15.6367C9.30352 15.8787 9.6002 16 9.9502 16C10.3001 16 10.5959 15.8787 10.8379 15.6367C11.0792 15.3954 11.2002 15.1 11.2002 14.75C11.2002 14.4 11.0792 14.1046 10.8379 13.8633C10.5959 13.6213 10.3001 13.5 9.9502 13.5ZM10.0996 4C9.14982 4.00007 8.37933 4.2501 7.78809 4.75C7.19612 5.24997 6.78314 5.84985 6.5498 6.5498L8.2002 7.2002C8.28353 6.9002 8.4707 6.57461 8.7627 6.22461C9.05397 5.87474 9.49985 5.70027 10.0996 5.7002C10.6328 5.7002 11.0331 5.84555 11.2998 6.13672C11.5664 6.42861 11.7001 6.74976 11.7002 7.09961C11.7002 7.4329 11.6003 7.74581 11.4004 8.03711C11.2005 8.32897 10.9502 8.59971 10.6504 8.84961C9.91706 9.49961 9.46647 9.99186 9.2998 10.3252C9.13322 10.6586 9.0498 11.2673 9.0498 12.1504H10.9004C10.9004 11.6004 10.9626 11.1663 11.0879 10.8496C11.2127 10.533 11.5672 10.0996 12.1504 9.5498C12.5836 9.11658 12.9249 8.70343 13.1748 8.31152C13.4246 7.92033 13.5497 7.45011 13.5498 6.90039C13.5498 5.96717 13.2086 5.24999 12.5254 4.75C11.8421 4.25 11.0329 4 10.0996 4Z',
    maskPath: CIRCLE_MASK,
    medium: { width: 20, height: 20, offsetY: 0 },
    small: { width: 16.666, height: 16.666, offsetY: 0 },
  },
  warning: {
    paletteKey: 'warning',
    viewBox: '0 0 22 19',
    path: 'M22 19H0L11 0L22 19ZM11 14C10.7167 14 10.4788 14.0954 10.2871 14.2871C10.0954 14.4788 10 14.7167 10 15C10 15.2833 10.0954 15.5212 10.2871 15.7129C10.4788 15.9046 10.7167 16 11 16C11.2833 16 11.5212 15.9046 11.7129 15.7129C11.9046 15.5212 12 15.2833 12 15C12 14.7167 11.9046 14.4788 11.7129 14.2871C11.5212 14.0954 11.2833 14 11 14ZM10 13H12V8H10V13Z',
    maskPath: 'M22 19H0L11 0L22 19Z',
    medium: { width: 22, height: 19, offsetY: -2.5 },
    small: { width: 18.334, height: 15.833, offsetY: -0.42 },
  },
  error: {
    paletteKey: 'error',
    viewBox: '0 0 18 18',
    path: 'M18 5.25V12.75L12.75 18H5.25L0 12.75V5.25L5.25 0H12.75L18 5.25ZM9 12C8.71667 12 8.47878 12.0954 8.28711 12.2871C8.09544 12.4788 8 12.7167 8 13C8 13.2833 8.09544 13.5212 8.28711 13.7129C8.47878 13.9046 8.71667 14 9 14C9.28333 14 9.52122 13.9046 9.71289 13.7129C9.90456 13.5212 10 13.2833 10 13C10 12.7167 9.90456 12.4788 9.71289 12.2871C9.52122 12.0954 9.28333 12 9 12ZM8 10H10V4H8V10Z',
    maskPath: 'M18 5.25V12.75L12.75 18H5.25L0 12.75V5.25L5.25 0H12.75L18 5.25Z',
    medium: { width: 18, height: 18, offsetY: 0 },
    small: { width: 15, height: 15, offsetY: 0 },
  },
};

const StatusIcon = ({ severity, size = 'medium', surfaceColor, sx, ...props }: StatusIconProps) => {
  const theme = useTheme();
  const spec = SYMBOLS[severity];
  const palette = theme.palette[spec.paletteKey] as {
    light: string;
    contrastText: string;
    surface: string;
  };

  const isSmall = size === 'small';
  const box = isSmall ? spec.small : spec.medium;

  // medium: dark symbol on the tinted wrapper. small: the saturated symbol.
  const symbolColor = isSmall ? palette.light : palette.contrastText;
  // The elevation surface the icon sits on. Backs the small knock-out mask and
  // the medium wrapper so the (sometimes translucent) severity surface composites
  // over an opaque base instead of whatever is behind the icon.
  const contextSurface = surfaceColor ?? theme.palette.background.paper;

  const symbol = (
    <Box
      component="svg"
      viewBox={spec.viewBox}
      sx={{
        position: 'relative',
        top: box.offsetY,
        width: box.width,
        height: box.height,
        display: 'block',
        overflow: 'visible',
      }}
    >
      {isSmall && (
        <path
          d={spec.maskPath}
          fill={contextSurface}
          stroke={contextSurface}
          strokeWidth={MASK_BORDER}
          strokeLinejoin="round"
        />
      )}
      <path d={spec.path} fill={symbolColor} />
    </Box>
  );

  return (
    <Box
      {...props}
      sx={[
        {
          display: 'inline-flex',
          alignItems: 'center',
          justifyContent: 'center',
          boxSizing: 'border-box',
          flexShrink: 0,
          width: isSmall ? 22 : 32,
          height: isSmall ? 22 : 32,
          ...(isSmall
            ? {}
            : {
                borderRadius: '50%',
                // severity surface composited over the opaque context base
                background: `linear-gradient(0deg, ${palette.surface}, ${palette.surface}), ${contextSurface}`,
                border: `1px solid ${theme.palette.divider}`,
              }),
        },
        ...(Array.isArray(sx) ? sx : [sx]),
      ]}
    >
      {symbol}
    </Box>
  );
};

export default StatusIcon;
