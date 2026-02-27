import type { Meta, StoryObj } from '@storybook/react';
import { Typography, TypographyProps } from '@mui/material';

const meta = {
  title: 'Foundations/Typography',
  parameters: {
    layout: 'centered',
  },
} satisfies Meta;

export default meta;
type Story = StoryObj<TypographyProps>;

export const H1: Story = {
  parameters: {
    docs: {
      source: {
        code: '<Typography variant="h1">Disregard and contempt</Typography>',
      },
    },
  },

  render: function Render() {
    return (
      <>
        <Typography variant='h1'>
          Disregard and contempt
        </Typography>
      </>
    );
  },
};

export const H2: Story = {
  parameters: {
    docs: {
      source: {
        code: '<Typography variant="h2">Whereas disregard and contempt for human rights</Typography>',
      },
    },
  },

  render: function Render() {
    return (
      <>
        <Typography variant='h2'>
          Whereas disregard and contempt for human rights
        </Typography>
      </>
    );
  },
};

export const H3: Story = {
  parameters: {
    docs: {
      source: {
        code: '<Typography variant="h3">Whereas disregard and contempt for human rights have resulted</Typography>',
      },
    },
  },

  render: function Render() {
    return (
      <>
        <Typography variant='h3'>
          Whereas disregard and contempt for human rights have resulted
        </Typography>
      </>
    );
  },
};

export const H4: Story = {
  parameters: {
    docs: {
      source: {
        code: '<Typography variant="h4">No one shall be held in slavery or servitude; slavery and the slave trade shall be prohibited</Typography>',
      },
    },
  },

  render: function Render() {
    return (
      <>
        <Typography variant='h4'>
          No one shall be held in slavery or servitude; slavery and the slave trade shall be prohibited
        </Typography>
      </>
    );
  },
};

export const H5: Story = {
  parameters: {
    docs: {
      source: {
        code: '<Typography variant="h5">No one shall be held in slavery or servitude; slavery and the slave trade shall be prohibited in all their forms</Typography>',
      },
    },
  },

  render: function Render() {
    return (
      <>
        <Typography variant='h5'>
          No one shall be held in slavery or servitude; slavery and the slave trade shall be prohibited in all their forms
        </Typography>
      </>
    );
  },
};

export const H6: Story = {
  parameters: {
    docs: {
      source: {
        code: '<Typography variant="h6">Everyone has the right to an effective remedy by the competent national tribunals for acts violating the fundamental rights granted him by the constitution or by law</Typography>',
      },
    },
  },

  render: function Render() {
    return (
      <>
        <Typography variant='h6'>
          Everyone has the right to an effective remedy by the competent national tribunals for acts violating the fundamental rights granted him by the constitution or by law
        </Typography>
      </>
    );
  },
};

export const Subhead1: Story = {
  parameters: {
    docs: {
      source: {
        code: '<Typography variant="subHead1">Everyone has the right to an effective remedy by the competent national tribunals for acts violating the fundamental rights granted him by the constitution or by law.</Typography>',
      },
    },
  },

  render: function Render() {
    return (
      <>
        <Typography variant='subHead1'>
          Everyone has the right to an effective remedy by the competent national tribunals for acts violating the fundamental rights granted him by the constitution or by law.
        </Typography>
      </>
    );
  },
};

export const Subhead2: Story = {
  parameters: {
    docs: {
      source: {
        code: '<Typography variant="subHead2">No one shall be subjected to arbitrary arrest, detention or exile. Everyone is entitled in full equality to a fair and public hearing by an independent and impartial tribunal, in the determination of his rights and obligations and of any criminal charge against him.</Typography>',
      },
    },
  },

  render: function Render() {
    return (
      <>
        <Typography variant='subHead2'>
          No one shall be subjected to arbitrary arrest, detention or exile. Everyone is entitled in full equality to a fair and public hearing by an independent and impartial tribunal, in the determination of his rights and obligations and of any criminal charge against him.
        </Typography>
      </>
    );
  },
};

export const Overline: Story = {
  parameters: {
    docs: {
      source: {
        code: '<Typography variant="overline">Everyone has the right to an effective remedy</Typography>',
      },
    },
  },

  render: function Render() {
    return (
      <>
        <Typography variant='overline'>
          Everyone has the right to an effective remedy
        </Typography>
      </>
    );
  },
};

export const SectionHeading: Story = {
  parameters: {
    docs: {
      source: {
        code: '<Typography variant="sectionHeading">Everyone is entitled in full equality to a fair and public hearing.</Typography>',
      },
    },
  },

  render: function Render() {
    return (
      <>
        <Typography variant='sectionHeading'>
          Everyone is entitled in full equality to a fair and public hearing.
        </Typography>
      </>
    );
  },
};

export const Body1: Story = {
  parameters: {
    docs: {
      source: {
        code: '<Typography variant="body1">No one shall be subjected to arbitrary arrest, detention or exile. Everyone is entitled in full equality to a fair and public hearing by an independent and impartial tribunal, in the determination of his rights and obligations and of any criminal charge against him. No one shall be subjected to arbitrary interference with his privacy, family, home or correspondence, nor to attacks upon his honour and reputation. Everyone has the right to the protection of the law against such interference or attacks.</Typography>',
      },
    },
  },

  render: function Render() {
    return (
      <>
        <Typography variant='body1'>
          No one shall be subjected to arbitrary arrest, detention or exile. Everyone is entitled in full equality to a fair and public hearing by an independent and impartial tribunal, in the determination of his rights and obligations and of any criminal charge against him. No one shall be subjected to arbitrary interference with his privacy, family, home or correspondence, nor to attacks upon his honour and reputation. Everyone has the right to the protection of the law against such interference or attacks.
        </Typography>
      </>
    );
  },
};

export const Body2: Story = {
  parameters: {
    docs: {
      source: {
        code: '<Typography variant="body2">No one shall be subjected to arbitrary arrest, detention or exile. Everyone is entitled in full equality to a fair and public hearing by an independent and impartial tribunal, in the determination of his rights and obligations and of any criminal charge against him. No one shall be subjected to arbitrary interference with his privacy, family, home or correspondence, nor to attacks upon his honour and reputation. Everyone has the right to the protection of the law against such interference or attacks.</Typography>',
      },
    },
  },

  render: function Render() {
    return (
      <>
        <Typography variant='body2'>
          No one shall be subjected to arbitrary arrest, detention or exile. Everyone is entitled in full equality to a fair and public hearing by an independent and impartial tribunal, in the determination of his rights and obligations and of any criminal charge against him. No one shall be subjected to arbitrary interference with his privacy, family, home or correspondence, nor to attacks upon his honour and reputation. Everyone has the right to the protection of the law against such interference or attacks.
        </Typography>
      </>
    );
  },
};

export const Caption: Story = {
  parameters: {
    docs: {
      source: {
        code: '<Typography variant="caption">No one shall be subjected to arbitrary arrest, detention or exile. Everyone is entitled in full equality to a fair and public hearing by an independent and impartial tribunal, in the determination of his rights and obligations and of any criminal charge against him. No one shall be subjected to arbitrary interference with his privacy, family, home or correspondence, nor to attacks upon his honour and reputation. Everyone has the right to the protection of the law against such interference or attacks.</Typography>',
      },
    },
  },

  render: function Render() {
    return (
      <>
        <Typography variant='caption'>
          No one shall be subjected to arbitrary arrest, detention or exile. Everyone is entitled in full equality to a fair and public hearing by an independent and impartial tribunal, in the determination of his rights and obligations and of any criminal charge against him. No one shall be subjected to arbitrary interference with his privacy, family, home or correspondence, nor to attacks upon his honour and reputation. Everyone has the right to the protection of the law against such interference or attacks.
        </Typography>
      </>
    );
  },
};

export const Button: Story = {
  parameters: {
    docs: {
      source: {
        code: '<Typography variant="button">Whereas recognition</Typography>',
      },
    },
  },

  render: function Render() {
    return (
      <>
        <Typography variant='button'>
          Whereas recognition
        </Typography>
      </>
    );
  },
};

export const MenuText: Story = {
  parameters: {
    docs: {
      source: {
        code: '<Typography variant="menuText">Whereas recognition</Typography>',
      },
    },
  },

  render: function Render() {
    return (
      <>
        <Typography variant='menuText'>
          Whereas recognition
        </Typography>
      </>
    );
  },
};

export const InputText: Story = {
  parameters: {
    docs: {
      source: {
        code: '<Typography variant="inputText">Whereas recognition</Typography>',
      },
    },
  },

  render: function Render() {
    return (
      <>
        <Typography variant='inputText'>
          Whereas recognition
        </Typography>
      </>
    );
  },
};

export const InputLabel: Story = {
  parameters: {
    docs: {
      source: {
        code: '<Typography variant="inputLabel">Whereas recognition</Typography>',
      },
    },
  },

  render: function Render() {
    return (
      <>
        <Typography variant='inputLabel'>
          Whereas recognition
        </Typography>
      </>
    );
  },
};

export const HelperText: Story = {
  parameters: {
    docs: {
      source: {
        code: '<Typography variant="helperText">Whereas recognition</Typography>',
      },
    },
  },

  render: function Render() {
    return (
      <>
        <Typography variant='helperText'>
          Whereas recognition
        </Typography>
      </>
    );
  },
};
