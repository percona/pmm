import type { Meta, StoryObj } from '@storybook/react';
import { Typography, type TypographyProps } from '@mui/material';

const meta = {
  title: 'Foundations/Typography',
  parameters: {
    layout: 'padded',
  },
} satisfies Meta;

export default meta;
type Story = StoryObj<TypographyProps>;

const labelStyle = {
  display: 'block',
  fontSize: '0.875rem',
  fontFamily: '"Roboto Mono", monospace',
  opacity: 0.5,
  marginBottom: '0.25rem',
};

const stackStyle = {
  display: 'flex',
  flexDirection: 'column' as const,
  gap: '2rem',
};

/* ── Headings ─────────────────────────────────────── */

const headingVariants = ['h1', 'h2', 'h3', 'h4', 'h5', 'h6'] as const;

const headingSamples: Record<(typeof headingVariants)[number], string> = {
  h1: 'Critical Thinking',
  h2: 'Education Cultivates Empathy',
  h3: 'Knowledge Empowers Us To Shape Our Destiny',
  h4: 'Investment In Knowledge Consistently Pays The Best Interest',
  h5: 'Reading Unlocks Doors To New Worlds And Perspectives',
  h6: 'Learning Happens When We Are Engaged And Excited',
};

export const HeadingsShowcase: Story = {
  tags: ['!dev'],
  render: () => (
    <div style={stackStyle}>
      {headingVariants.map((v) => (
        <div key={v}>
          <span style={labelStyle}>{v}</span>
          <Typography variant={v}>{headingSamples[v]}</Typography>
        </div>
      ))}
    </div>
  ),
};

export const Headings: Story = {
  args: {
    variant: 'h1',
    children: headingSamples.h1,
  },
  argTypes: {
    variant: {
      control: 'select',
      options: [...headingVariants],
    },
    children: { control: 'text' },
  },
  render: (args) => (
    <Typography variant={args.variant}>{args.children}</Typography>
  ),
};

/* ── Titling ──────────────────────────────────────── */

const titlingVariants = ['subHead1', 'subHead2', 'overline', 'sectionHeading'] as const;

const titlingSamples: Record<(typeof titlingVariants)[number], string> = {
  subHead1: 'Education is the passport to the future and the key to unlocking the world. Students thrive when teachers actively inspire and challenge them.',
  subHead2: 'Critical thinking is the essential compass for navigating a complex world. Every focused lesson learned is a significant step toward personal growth. Technology provides new, borderless avenues for global learning and connection.',
  overline: 'Critical Thinking',
  sectionHeading: 'Education Is The Passport',
};

export const TitlingShowcase: Story = {
  tags: ['!dev'],
  render: () => (
    <div style={stackStyle}>
      {titlingVariants.map((v) => (
        <div key={v}>
          <span style={labelStyle}>{v}</span>
          <Typography variant={v}>{titlingSamples[v]}</Typography>
        </div>
      ))}
    </div>
  ),
};

export const Titling: Story = {
  args: {
    variant: 'subHead1' as TypographyProps['variant'],
    children: titlingSamples.subHead1,
  },
  argTypes: {
    variant: {
      control: 'select',
      options: [...titlingVariants],
    },
    children: { control: 'text' },
  },
  render: (args) => (
    <Typography variant={args.variant}>{args.children}</Typography>
  ),
};

/* ── Body ─────────────────────────────────────────── */

const bodyVariants = ['body1', 'body2', 'caption'] as const;

const bodySample =
  'An investment in knowledge consistently pays the best interest. The thoughtful pursuit of wisdom is a challenging, lifelong adventure. Every focused lesson learned is a significant step toward personal growth. Knowledge empowers individuals to confidently shape their own destiny. The thoughtful pursuit of wisdom is a challenging, lifelong adventure. A holistic education cultivates vital empathy and deep understanding. Learning is a continuous, lifelong journey, not a static destination.';

export const BodyShowcase: Story = {
  tags: ['!dev'],
  render: () => (
    <div style={stackStyle}>
      {bodyVariants.map((v) => (
        <div key={v}>
          <span style={labelStyle}>{v}</span>
          <Typography variant={v}>{bodySample}</Typography>
        </div>
      ))}
    </div>
  ),
};

export const Body: Story = {
  name: 'Body styles',
  args: {
    variant: 'body1',
    children: bodySample,
  },
  argTypes: {
    variant: {
      control: 'select',
      options: [...bodyVariants],
    },
    children: { control: 'text' },
  },
  render: (args) => (
    <Typography variant={args.variant}>{args.children}</Typography>
  ),
};

/* ── Component text styles ────────────────────────── */

const componentVariants = ['button', 'menuText', 'inputText', 'inputLabel', 'helperText'] as const;

const componentSample = 'Select database';

export const ComponentStylesShowcase: Story = {
  tags: ['!dev'],
  render: () => (
    <div style={stackStyle}>
      {componentVariants.map((v) => (
        <div key={v}>
          <span style={labelStyle}>{v}</span>
          <Typography variant={v}>{componentSample}</Typography>
        </div>
      ))}
    </div>
  ),
};

export const ComponentStyles: Story = {
  name: 'Component text styles',
  args: {
    variant: 'button',
    children: componentSample,
  },
  argTypes: {
    variant: {
      control: 'select',
      options: [...componentVariants],
    },
    children: { control: 'text' },
  },
  render: (args) => (
    <Typography variant={args.variant}>{args.children}</Typography>
  ),
};
