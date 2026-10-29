import type { Meta, StoryObj } from '@storybook/react';
import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';
import StarBorderOutlined from '@mui/icons-material/StarBorderOutlined';

const meta = {
  title: 'Foundations/Surfaces',
  component: Paper,
  tags: ['autodocs', 'experimental'],
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        component:
          '`<Paper variant="dashed">` outlines a region to point out, such as an empty state that says how to fill it: a 2px dashed line in the divider color, 6px dashes 4px apart, with the theme\'s medium radius and no fill or shadow. The line takes the first 2px of the region, so pad the content 2px more than inside a solid border.',
      },
    },
  },
} satisfies Meta<typeof Paper>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Dashed: Story = {
  args: { variant: 'dashed' },
  render: (args) => (
    <Paper
      {...args}
      sx={{ display: 'flex', alignItems: 'flex-start', gap: 1, p: '10px', maxWidth: 224 }}
    >
      <StarBorderOutlined sx={{ flexShrink: 0, my: '-1px', color: 'text.secondary' }} />
      <Typography variant="caption" color="text.secondary">
        Click the star on apps or dashboards to add them here.
      </Typography>
    </Paper>
  ),
};
