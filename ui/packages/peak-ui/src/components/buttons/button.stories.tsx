import { type Meta, type StoryObj } from '@storybook/react';
import * as DocBlock from '@storybook/addon-docs/blocks';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import type { ButtonProps } from '@mui/material/Button';
import AddOutlined from '@mui/icons-material/AddOutlined';
import ArrowBackOutlined from '@mui/icons-material/ArrowBackOutlined';
import ArrowForwardOutlined from '@mui/icons-material/ArrowForwardOutlined';
import DeleteOutlineOutlined from '@mui/icons-material/DeleteOutlineOutlined';

const VARIANTS = ['contained', 'outlined', 'text'] as const;
const SIZES = ['large', 'medium', 'small'] as const;

const meta = {
  title: 'Inputs/Button',
  component: Button,
  tags: ['autodocs', 'stable'],
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: [
          "All buttons come fully themed by rendering MUI's `Button` inside `ThemeContextProvider`. There is no separate wrapper component. Pick variants by emphasis:",
          '',
          '| Variant | Emphasis | Use for |',
          '| --- | --- | --- |',
          '| `contained` | High | The single primary action of a view. Distinguished by fill. Prefer to use one per page. |',
          '| `outlined` | Medium | Still important actions but lower-emphasis vs. contained, or a higher-emphasis alternative to text. Use sparingly. |',
          '| `text` | Low | Less-pronounced actions, e.g. in dialogs and cards, where emphasis should stay on the content. Use for secondary actions. When in doubt, use this.|',
          '',
          '#### Design system reminder: Keep it consistent — No stray buttons',
          '',
          "- **Do** import via `import Button from '@mui/material/Button'` (path import) and stick to the documented `variant`, `size`, `disabled`, `startIcon`/`endIcon`, and `fullWidth` props.",
          '- **Do** reach for the composed buttons before hand-rolling your own: [Icon Button](?path=/docs/inputs-icon-button--docs) for icon-only actions, [Copy to Clipboard](?path=/docs/inputs-copy-to-clipboard--docs), and `MenuButton` for a button that opens a dropdown.',
          "- **Don't** restyle buttons through `sx`/`styled` — colors, radius, borders, padding, typography, and text transform all come from the theme. Layout-only `sx` (margins, alignment, width) is fine.",
          "- **Don't** pass palette `color`s. The design kit defines buttons in the primary (brand) color only; other colors are not part of the system.",
          '- **Try to** not use a button as a toggle to alter GUI state. Contained buttons should have **no selected state by design**. Use buttons to change the user-experience flow.',
        ].join('\n'),
      },
      page: () => (
        <>
          <DocBlock.Title />
          <DocBlock.Subtitle />
          <DocBlock.Description />
          <DocBlock.Primary />
          <DocBlock.Controls />
          <DocBlock.Stories />
        </>
      ),
    },
  },
  args: {
    variant: 'contained',
    size: 'large',
    children: 'Label',
  },
  argTypes: {
    variant: {
      options: VARIANTS,
      control: { type: 'inline-radio' },
      description: 'Emphasis level: `contained` high, `outlined` medium, `text` low.',
      table: { defaultValue: { summary: "'text'" } },
    },
    size: {
      options: SIZES,
      control: { type: 'inline-radio' },
      description: 'Design-kit heights: large 40 px, medium 32 px, small 24 px.',
      table: { defaultValue: { summary: "'medium'" } },
    },
    disabled: {
      control: 'boolean',
      description: 'Disables the button and applies the muted palette.',
      table: { defaultValue: { summary: 'false' } },
    },
    children: {
      control: 'text',
      description: 'The button label.',
    },
    startIcon: { control: false, description: 'Icon rendered before the label.' },
    endIcon: { control: false, description: 'Icon rendered after the label.' },
  },
} satisfies Meta<ButtonProps>;

export default meta;
type Story = StoryObj<ButtonProps>;

export const Playground: Story = {};

export const Variants: Story = {
  parameters: {
    docs: {
      description: {
        story:
          'The three approved variants, from high to low emphasis. A view should have at most one contained button; pair it with outlined or text buttons for secondary actions.',
      },
    },
  },
  render: ({ size }) => (
    <Stack direction="row" alignItems="center" gap={2}>
      {VARIANTS.map((variant) => (
        <Stack key={variant} alignItems="center" gap={1}>
          <Button variant={variant} size={size}>
            Label
          </Button>
          <Typography variant="caption" color="text.secondary">
            {variant}
          </Typography>
        </Stack>
      ))}
    </Stack>
  ),
};

export const Sizes: Story = {
  parameters: {
    docs: {
      description: {
        story:
          'Heights match the design kit (40 → 32 → 24 px) via `minHeight` and padding. Label size drops from 15 px (large) to 13 px (medium, small). Prefer `large` for page-level actions and `small` in dense surfaces like tables and cards.',
      },
    },
  },
  render: () => (
    <Stack gap={2}>
      {SIZES.map((size) => (
        <Stack key={size} direction="row" alignItems="center" gap={2}>
          <Typography variant="caption" color="text.secondary" sx={{ width: 56 }}>
            {size}
          </Typography>
          {VARIANTS.map((variant) => (
            <Button key={variant} variant={variant} size={size}>
              Label
            </Button>
          ))}
        </Stack>
      ))}
    </Stack>
  ),
};

export const Hovered: Story = {
  parameters: {
    pseudo: { hover: true },
    docs: {
      description: {
        story: 'Hover feedback comes from primary state tokens.',
      },
    },
  },
  render: ({ size }) => (
    <Stack direction="row" alignItems="center" gap={2}>
      {VARIANTS.map((variant) => (
        <Button key={variant} variant={variant} size={size}>
          Label
        </Button>
      ))}
    </Stack>
  ),
};

export const Disabled: Story = {
  parameters: {
    docs: {
      description: {
        story:
          "Disabled buttons keep their shape, mute the label to `text.disabled`, and (for contained) fill with `action.disabled`. Set the `disabled` prop — don't fade buttons with opacity.",
      },
    },
  },
  render: ({ size }) => (
    <Stack direction="row" alignItems="center" gap={2}>
      {VARIANTS.map((variant) => (
        <Button key={variant} variant={variant} size={size} disabled>
          Label
        </Button>
      ))}
    </Stack>
  ),
};

export const WithIcon: Story = {
  name: 'With icon',
  parameters: {
    docs: {
      description: {
        story:
          'Use `startIcon`/`endIcon` with custom Peak Design icons or Material icons. Icon size matches Icon Button (24 / 20 / 16). Contained and outlined use a 12px layout slot so icons pull toward the pill edge; text buttons stay flush with padding. Gap to the label is 4px.',
      },
    },
  },
  render: ({ variant, size }) => (
    <Stack direction="row" alignItems="center" gap={2}>
      <Button variant={variant} size={size} startIcon={<ArrowBackOutlined />}>
        Back
      </Button>
      <Button variant={variant} size={size} endIcon={<ArrowForwardOutlined />}>
        Next
      </Button>
      <Button variant={variant} size={size} startIcon={<AddOutlined />}>
        Add new
      </Button>
    </Stack>
  ),
};

export const ActionPairing: Story = {
  name: 'Action pairing example',
  parameters: {
    docs: {
      description: {
        story:
          'An example that could fit a dialog/form footer: the primary action is contained, the dismissing action is text, and a destructive-but-secondary action sits apart as outlined. Emphasis — not color — communicates the hierarchy.',
      },
    },
  },
  render: ({ size }) => (
    <Stack direction="row" alignItems="center" gap={1} sx={{ width: 480 }}>
      <Button variant="outlined" size={size} startIcon={<DeleteOutlineOutlined />}>
        Delete
      </Button>
      <Stack direction="row" gap={1} sx={{ ml: 'auto' }}>
        <Button variant="text" size={size}>
          Cancel
        </Button>
        <Button variant="contained" size={size}>
          Save changes
        </Button>
      </Stack>
    </Stack>
  ),
};
