import { MenuItem } from '@mui/material';
import type { Meta, StoryObj } from '@storybook/react';
import { FormProvider, useForm } from 'react-hook-form';
import SelectInput from './select';
import * as DocBlock from '@storybook/blocks';

type CustomArgs = React.ComponentProps<typeof SelectInput> & {
  disabled?: boolean;
  error?: boolean;
  required?: boolean;
  size?: 'small' | 'medium';
  options?: string[];
  width?: string;
};

const meta = {
  title: 'Inputs/Select',
  component: SelectInput,
  tags: ['autodocs'],
  parameters: {
    layout: 'centered',
    docs: {
      source: {
        code: null,
      },
      toc: true,
      page: () => (
        <>
          <DocBlock.Title />
          <DocBlock.Subtitle />
          <DocBlock.Description />
          <DocBlock.Source
            code={`
<SelectInput
          selectFieldProps={{
            error,
          }}
          formControlProps={{
            sx: { width },
            disabled,
            required,
            size,
          }}
          name={'select'}
          label={label}
        >
          <MenuItem> Option </MenuItem>
</SelectInput>
            `}
          />
          <DocBlock.Primary />
          <DocBlock.Controls />
          <DocBlock.Stories />
        </>
      ),
    },
  },

  argTypes: {
    disabled: {
      type: 'boolean',
    },
    error: {
      type: 'boolean',
    },
    required: {
      type: 'boolean',
    },
    size: {
      options: ['small', 'medium'],
      control: { type: 'inline-radio' },
    },
    width: {
      type: 'string',
      defaultValue: '200px',
    },
  },

  render: function Render({
    disabled,
    error,
    required,
    size,
    options,
    width,
    label,
  }) {
    const methods = useForm();

    return (
      <FormProvider {...methods}>
        <SelectInput
          selectFieldProps={{
            error,
          }}
          formControlProps={{
            sx: { width },
            disabled,
            required,
            size,
          }}
          name={'select'}
          label={label}
        >
          {options?.map((item) => <MenuItem value={item}>{item}</MenuItem>)}
        </SelectInput>
      </FormProvider>
    );
  },
} satisfies Meta<CustomArgs>;
export default meta;

type Story = StoryObj<CustomArgs>;

export const Basic: Story = {
  tags: ['!autodocs'],
  args: {
    label: 'Select',
    disabled: false,
    error: false,
    required: false,
    size: 'small',
    options: ['First', 'Second', 'Third'],
    width: '200px',
  },
};

export const Small: Story = {
  parameters: {
    docs: {
      description: {
        story: `\`\`\`ts
<SelectInput formControlProps={size: "small"}>
  Options...
</SelectInput>
        \`\`\``,
      },
    },
  },
  args: {
    label: 'Small Select',
    size: 'small',
    options: ['First', 'Second', 'Third'],
    width: '200px',
  },
};

export const Medium: Story = {
  parameters: {
    docs: {
      description: {
        story: `\`\`\`ts
<SelectInput formControlProps={size: "medium"}>
  Options...
</SelectInput>
        \`\`\``,
      },
    },
  },
  args: {
    label: 'Medium Select',
    size: 'medium',
    options: ['First', 'Second', 'Third'],
    width: '200px',
  },
};

export const Disabled: Story = {
  parameters: {
    docs: {
      description: {
        story: `\`\`\`ts
<SelectInput formControlProps={disabled: true}>
  Options...
</SelectInput>
        \`\`\``,
      },
    },
  },
  args: {
    label: 'Disabled Select',
    disabled: true,
    options: ['First', 'Second', 'Third'],
    width: '200px',
  },
};

export const Error: Story = {
  parameters: {
    docs: {
      description: {
        story: `\`\`\`ts
<SelectInput formControlProps={error: true}>
  Options...
</SelectInput>
        \`\`\``,
      },
    },
  },
  args: {
    label: 'Error Select',
    error: true,
    options: ['First', 'Second', 'Third'],
    width: '200px',
  },
};

export const Hover: Story = {
  parameters: {
    pseudo: {
      hover: true,
    },
  },
  args: {
    label: 'Hover',
    options: ['First', 'Second', 'Third'],
    width: '200px',
  },
};

export const Focus: Story = {
  parameters: {
    pseudo: {
      focus: true,
    },
  },
  args: {
    label: 'Focus',
    options: ['First', 'Second', 'Third'],
    width: '200px',
  },
};
