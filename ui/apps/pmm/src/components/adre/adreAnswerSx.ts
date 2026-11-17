/** The Assistant's dark palette, shared by the pages that read like it. */
export const adreColors = {
  page: '#121114',
  panel: '#17161A',
  card: '#1F1E23',
  border: 'rgba(255,255,255,0.08)',
  divider: 'rgba(255,255,255,0.06)',
  text: '#F4F3F6',
  body: '#E9E7EE',
  secondary: '#A9A7B0',
  muted: '#8E8C95',
  accent: '#B6B2FF',
  accentText: '#CFCBFF',
  leadBg: 'rgba(182,178,255,0.08)',
  leadBorder: 'rgba(182,178,255,0.2)',
} as const;

export const headingFont = 'Poppins, sans-serif';

// An answer reads as a document: headings in Poppins and numbered steps.
export const answerBodySx = {
  minWidth: 0,
  fontSize: 15,
  lineHeight: 1.65,
  color: adreColors.body,
  overflowWrap: 'anywhere',
  wordBreak: 'break-word',
  '& img': { maxWidth: '100%', height: 'auto', display: 'block' },
  '& h1, & h2, & h3, & h4, & h5, & h6': {
    fontFamily: headingFont,
    fontWeight: 600,
    color: adreColors.text,
    lineHeight: 1.4,
    mt: 3,
    mb: 1,
  },
  '& h1': { fontSize: 20 },
  '& h2': { fontSize: 18 },
  '& h3, & h4, & h5, & h6': { fontSize: 16 },
  '& > :first-child': { mt: 0 },
  '& strong': { color: adreColors.text },
  '& ul': { pl: 2.75, my: 1 },
  '& li': { my: 0.75 },
  '& ol': { listStyle: 'none', pl: 0, my: 1.5 },
  '& ol > li': { position: 'relative', pl: '40px', my: 2 },
  '& ol > li::before': {
    content: 'attr(data-n)',
    position: 'absolute',
    left: 0,
    top: 0,
    width: 26,
    height: 26,
    borderRadius: '50%',
    bgcolor: '#26252B',
    color: adreColors.accentText,
    fontSize: 13,
    fontWeight: 500,
    lineHeight: '26px',
    textAlign: 'center',
  },
  '& blockquote': {
    m: 0,
    my: 1.5,
    pl: 2,
    color: adreColors.secondary,
    borderLeft: '3px solid rgba(255,255,255,0.12)',
  },
} as const;

/** An Assistant answer, which also highlights its lead. */
export const answerSx = {
  ...answerBodySx,
  // The lead: the opening paragraph, or the one under an opening heading such as Summary.
  '& > .adre-md-p:first-child, & > :is(h1, h2, h3):first-child + .adre-md-p': {
    px: 2.25,
    py: 2,
    mb: 2.5,
    borderRadius: '14px',
    bgcolor: adreColors.leadBg,
    border: `1px solid ${adreColors.leadBorder}`,
    color: '#ECEAF1',
  },
} as const;
