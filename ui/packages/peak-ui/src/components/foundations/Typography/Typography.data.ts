import { TypographyPropsVariantOverrides } from '@mui/material';
import { Variant } from '@mui/material/styles/createTypography';
import { OverridableStringUnion } from '@mui/types';

type Typography = Array<{
  variant: OverridableStringUnion<
    Variant | 'inherit',
    TypographyPropsVariantOverrides
  >;
  text: string;
  style?: {
    fontFamily?: string;
    fontSize?: string;
    fontWeight?: string;
    lineHeight?: string;
    color?: string;
    textDecoration?: string;
  };
}>[];
export const TYPOGRAPHY: Typography = [
  [
    {
      variant: 'h1',
      text: 'Disregard and contempt',
    },
    {
      variant: 'h2',
      text: 'Whereas disregard and contempt for human rights',
    },
    {
      variant: 'h3',
      text: 'Whereas disregard and contempt for human rights have resulted',
    },
    {
      variant: 'h4',
      text: 'No one shall be held in slavery or servitude; slavery and the slave trade shall be prohibited',
    },
    {
      variant: 'h5',
      text: 'No one shall be held in slavery or servitude; slavery and the slave trade shall be prohibited in all their forms',
    },
    {
      variant: 'h6',
      text: 'Everyone has the right to an effective remedy by the competent national tribunals for acts violating the fundamental rights granted him by the constitution or by law',
    },
  ],
  [
    {
      variant: 'subHead1',
      text: 'Everyone has the right to an effective remedy by the competent national tribunals for acts violating the fundamental rights granted him by the constitution or by law.',
    },
    {
      variant: 'subHead2',
      text: 'No one shall be subjected to arbitrary arrest, detention or exile. Everyone is entitled in full equality to a fair and public hearing by an independent and impartial tribunal, in the determination of his rights and obligations and of any criminal charge against him.',
    },
    {
      variant: 'overline',
      text: 'Everyone has the right to an effective remedy',
    },
    {
      variant: 'sectionHeading',
      text: 'Everyone is entitled in full equality to a fair and public hearing.',
    },
  ],
  [
    {
      variant: 'body1',
      text: 'No one shall be subjected to arbitrary arrest, detention or exile. Everyone is entitled in full equality to a fair and public hearing by an independent and impartial tribunal, in the determination of his rights and obligations and of any criminal charge against him. No one shall be subjected to arbitrary interference with his privacy, family, home or correspondence, nor to attacks upon his honour and reputation. Everyone has the right to the protection of the law against such interference or attacks.',
    },
    {
      variant: 'body2',
      text: 'No one shall be subjected to arbitrary arrest, detention or exile. Everyone is entitled in full equality to a fair and public hearing by an independent and impartial tribunal, in the determination of his rights and obligations and of any criminal charge against him. No one shall be subjected to arbitrary interference with his privacy, family, home or correspondence, nor to attacks upon his honour and reputation. Everyone has the right to the protection of the law against such interference or attacks.',
    },
    {
      variant: 'caption',
      text: 'No one shall be subjected to arbitrary arrest, detention or exile. Everyone is entitled in full equality to a fair and public hearing by an independent and impartial tribunal, in the determination of his rights and obligations and of any criminal charge against him. No one shall be subjected to arbitrary interference with his privacy, family, home or correspondence, nor to attacks upon his honour and reputation. Everyone has the right to the protection of the law against such interference or attacks.',
    },
  ],
  [
    {
      variant: 'button',
      text: 'Whereas recognition',
      style: {
        fontFamily: 'Poppins',
        fontSize: '15px',
        fontWeight: '600',
        lineHeight: '15.94px',
        color: '#2C323E',
      },
    },
    {
      variant: 'button',
      text: 'Whereas recognition',
      style: {
        fontFamily: 'Poppins',
        fontSize: '13px',
        fontWeight: '600',
        lineHeight: '14.82px',
        color: '#2C323E',
      },
    },
    {
      variant: 'button',
      text: 'Whereas recognition',
      style: {
        fontFamily: 'Poppins',
        fontSize: '13px',
        fontWeight: '600',
        lineHeight: '14.82px',
        color: '#2C323E',
      },
    },
    {
      variant: 'menuText',
      text: 'Whereas recognition',
      style: {
        fontFamily: 'Poppins',
        fontSize: '14px',
        fontWeight: '600',
        lineHeight: '17.5px',
        color: '#2C323E',
      },
    },
    {
      variant: 'body1',
      text: 'Whereas recognition',
      style: {
        fontFamily: 'Poppins',
        fontSize: '16px',
        fontWeight: '400',
        lineHeight: '22px',
        color: '#4B5468',
        textDecoration: 'underline',
      },
    },
    {
      variant: 'body2',
      text: 'Whereas recognition',
      style: {
        fontFamily: 'Poppins',
        fontSize: '15px',
        fontWeight: '600',
        lineHeight: '15.94px',
        color: '#4B5468',
        textDecoration: 'underline',
      },
    },
  ],
  [
    {
      variant: 'button',
      text: 'Whereas recognition',
    },
    {
      variant: 'menuText',
      text: 'Whereas recognition',
    },
  ],
  [
    {
      variant: 'inputText',
      text: 'Whereas recognition',
    },
    {
      variant: 'inputLabel',
      text: 'Whereas recognition',
    },
    {
      variant: 'helperText',
      text: 'Whereas recognition',
    },
  ],
];
