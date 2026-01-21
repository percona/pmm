import type { Meta, StoryObj } from '@storybook/react';
import { useForm } from 'react-hook-form';
import { FormProvider } from 'react-hook-form';
import * as DocBlock from '@storybook/blocks';
import TextInput from './text';
import { TextInputProps } from './text.types';
import { useState } from 'react';
import { IconButton, InputAdornment } from '@mui/material';
import { SearchOutlined, Visibility, VisibilityOff } from '@mui/icons-material';

type ExpandedTextInputProps = TextInputProps & {
  readOnly?: boolean;
  type?: string;
  maxLength?: number;
  placeholder?: string;
  error?: boolean;
  minRows?: number;
  maxRows?: number;
  size?: 'small' | 'medium';
  multiline?: boolean;
  disabled?: boolean;
  helperText?: string;
};

const meta = {
  title: 'Inputs/TextInput',
  component: TextInput,
  tags: ['autodocs'],
  parameters: {
    layout: 'centered',
    docs: {
      toc: true,
      page: () => (
        <>
          <DocBlock.Title />
          <DocBlock.Subtitle />
          <DocBlock.Description />
          <DocBlock.Source
            code={`
<TextInput
  name="TextInput"
  label="Label"
  isRequired={false}
  textFieldProps={{
    placeholder: "Placeholder",
    helperText: "Helper text",
    size: "small",
    type: "email",
    inputProps: {
      readOnly: false,
      maxLength: 16,
    },
  }}
/>
            `}
          />
          <DocBlock.Primary />
          <DocBlock.Controls />
          <DocBlock.Stories includePrimary={false} />
        </>
      ),
    },
  },
  argTypes: {
    disabled: {
      type: 'boolean',
      defaultValue: 'false',
      description: '`boolean`',
    },
    error: {
      type: 'boolean',
      defaultValue: 'false',
      description: '`boolean`',
    },
    isRequired: {
      type: 'boolean',
      defaultValue: 'false',
      description: '`boolean`',
    },
    multiline: {
      type: 'boolean',
      defaultValue: 'false',
      description: '`boolean`',
    },
    readOnly: {
      type: 'boolean',
      defaultValue: 'false',
      description: '`boolean`',
    },
    label: {
      type: 'string',
      description: '`string`',
    },
    maxLength: {
      type: 'number',
      description: '`number`',
    },
    maxRows: {
      type: 'number',
      description: '`number`',
    },
    minRows: {
      type: 'number',
      description: '`number`',
    },
    name: {
      type: 'string',
      description: '`string`',
    },
    placeholder: {
      type: 'string',
      description: '`string`',
    },
    helperText: {
      type: 'string',
      description: '`string`',
    },
    size: {
      options: ['small', 'medium'],
      control: { type: 'radio' },
      defaultValue: 'small',
      description: '`string`',
    },
    type: {
      options: ['email', 'password', 'number', 'text', 'tel', 'url'],
      control: 'select',
      description: '`string`',
    },
  },
  render: function Render({
    name,
    label,
    placeholder,
    readOnly,
    size,
    minRows,
    maxRows,
    multiline,
    type,
    maxLength,
    error,
    disabled,
    isRequired,
    helperText,
    ...args
  }) {
    const methods = useForm();
    if (error) {
      methods.setError(name, { message: 'Please insert a valid value' });
    } else {
      methods.clearErrors(name);
    }

    return (
      <FormProvider {...methods}>
        <TextInput
          {...args}
          name={name}
          label={label}
          isRequired={isRequired}
          textFieldProps={{
            disabled,
            error,
            placeholder,
            size,
            multiline,
            minRows,
            maxRows,
            type,
            helperText,
            inputProps: {
              readOnly,
              maxLength,
            },
          }}
        />
      </FormProvider>
    );
  },
} satisfies Meta<ExpandedTextInputProps>;

export default meta;
type Story = StoryObj<ExpandedTextInputProps>;

export const Custom: Story = {
  parameters: {
    docs: {
      source: {
        code: null,
      },
    },
  },
  args: {
    name: 'Custom',
    label: 'Label',
    placeholder: 'Placeholder',
    helperText: 'Helper text',
    maxLength: 16,
    size: 'small',
    type: 'text',
    error: false,
    disabled: false,
    readOnly: false,
    isRequired: false,
    multiline: false,
    minRows: 5,
    maxRows: 10,
  },
};

export const Small: Story = {
  parameters: {
    docs: {
      source: {
        code: `
          <TextInput
  name="TextInput"
  label="Label"
  isRequired={false}
  textFieldProps={{
    disabled: false,
    error: false,
    placeholder: "Placeholder",
    size: "small",
    type: "text",
    inputProps: {
      readOnly: false,
      maxLength: 16,
    },
  }}
/>
        `,
      },
    },
  },
  args: {
    name: 'Small',
    label: 'Label',
    placeholder: 'Placeholder',
    size: 'small',
    maxLength: 8,
    type: 'text',
    error: false,
    readOnly: false,
    isRequired: false,
    disabled: false,
  },
};

export const Medium: Story = {
  parameters: {
    docs: {
      source: {
        code: `
          <TextInput
  name="TextInput"
  label="Label"
  isRequired={false}
  textFieldProps={{
    disabled: false,
    error: false,
    placeholder: "Placeholder",
    size: "medium",
    type: "text",
    inputProps: {
      readOnly: false,
      maxLength: 16,
    },
  }}
/>
        `,
      },
    },
  },
  args: {
    name: 'Medium',
    label: 'Label',
    placeholder: 'Placeholder',
    maxLength: 8,
    size: 'medium',
    type: 'text',
    error: false,
    readOnly: false,
    isRequired: false,
    disabled: false,
  },
};

export const Disabled: Story = {
  parameters: {
    docs: {
      source: {
        code: `
          <TextInput
  name="TextInput"
  label="Label"
  isRequired={false}
  textFieldProps={{
    disabled: true,
    error: false,
    placeholder: "Placeholder",
    size: "small",
    type: "text",
    inputProps: {
      readOnly: false,
      maxLength: 16,
    },
  }}
/>
        `,
      },
    },
  },
  args: {
    name: 'Disabled',
    label: 'Label',
    placeholder: 'Placeholder',
    maxLength: 8,
    size: 'small',
    type: 'text',
    error: false,
    readOnly: false,
    isRequired: false,
    disabled: true,
  },
};

export const Required: Story = {
  parameters: {
    docs: {
      source: {
        code: `
          <TextInput
  name="TextInput"
  label="Label"
  isRequired={true}
  textFieldProps={{
    disabled: false,
    error: false,
    placeholder: "Placeholder",
    size: "small",
    type: "text",
    inputProps: {
      readOnly: false,
      maxLength: 16,
    },
  }}
/>
        `,
      },
    },
  },
  args: {
    name: 'Required',
    label: 'Label',
    placeholder: 'Placeholder',
    maxLength: 8,
    size: 'small',
    type: 'text',
    error: false,
    readOnly: false,
    isRequired: true,
    disabled: false,
  },
};

export const Error: Story = {
  parameters: {
    docs: {
      source: {
        code: `
          <TextInput
  name="TextInput"
  label="Label"
  isRequired={false}
  textFieldProps={{
    disabled: false,
    error: true,
    placeholder: "Placeholder",
    size: "small",
    type: "text",
    inputProps: {
      readOnly: false,
      maxLength: 16,
    },
  }}
/>
        `,
      },
    },
  },
  args: {
    name: 'Error',
    label: 'Label',
    placeholder: 'Placeholder',
    maxLength: 8,
    size: 'small',
    type: 'text',
    error: true,
    readOnly: false,
    isRequired: false,
    disabled: false,
  },
};

export const NumberType: Story = {
  parameters: {
    docs: {
      source: {
        code: `
          <TextInput
  name="TextInput"
  label="Label"
  isRequired={false}
  textFieldProps={{
    disabled: false,
    error: false,
    placeholder: "Placeholder",
    size: "small",
    type: "number",
    inputProps: {
      readOnly: false,
      maxLength: 16,
    },
  }}
/>
        `,
      },
    },
  },
  args: {
    name: 'NumberType',
    label: 'Label',
    placeholder: 'Placeholder',
    maxLength: 8,
    size: 'small',
    type: 'number',
    error: false,
    readOnly: false,
    isRequired: false,
    disabled: false,
  },
};

export const Password: Story = {
  parameters: {
    docs: {
      source: {
        code: `
const methods = useForm();

const [showPassword, setShowPassword] = useState(false);
const handleClickShowPassword = () => setShowPassword((show) => !show);

return (
  <FormProvider {...methods}>
    <TextInput
      name="name"
      label="label"
      isRequired={false}
      textFieldProps={{
        InputProps: {
          endAdornment: (
            <InputAdornment position="end">
              <IconButton onClick={handleClickShowPassword} edge="end">
                {showPassword ? <VisibilityOff /> : <Visibility />}
              </IconButton>
            </InputAdornment>
          ),
        },
        type: showPassword ? "text" : "password",
      }}
    />
  </FormProvider>
);
        `,
      },
    },
  },
  args: {
    name: 'Password',
    label: 'Label',
    placeholder: 'Placeholder',
    maxLength: 8,
    size: 'small',
    error: false,
    readOnly: false,
    isRequired: false,
    disabled: false,
  },
  render: function Password({
    name,
    label,
    placeholder,
    maxLength,
    size,
    error,
    readOnly,
    isRequired,
    disabled,
    helperText,
  }) {
    const methods = useForm();

    const [showPassword, setShowPassword] = useState(false);
    const handleClickShowPassword = () => setShowPassword((show) => !show);

    if (error) {
      methods.setError(name, { message: 'Please insert a valid value' });
    } else {
      methods.clearErrors(name);
    }

    return (
      <FormProvider {...methods}>
        <TextInput
          isRequired={isRequired}
          name={name}
          label={label}
          textFieldProps={{
            InputProps: {
              endAdornment: (
                <InputAdornment position="end">
                  <IconButton onClick={handleClickShowPassword} edge="end">
                    {showPassword ? <VisibilityOff /> : <Visibility />}
                  </IconButton>
                </InputAdornment>
              ),
            },
            inputProps: {
              maxLength,
              readOnly,
            },
            helperText,
            disabled,
            error,
            placeholder,
            size,
            type: showPassword ? 'text' : 'password',
          }}
        />
      </FormProvider>
    );
  },
};

export const Search: Story = {
  parameters: {
    docs: {
      source: {
        code: `
<TextInput
  name="name"
  label="label"
  isRequired={false}
  textFieldProps={{
    InputProps: {
      startAdornment: (
        <InputAdornment position="start">
          <SearchOutlined />
        </InputAdornment>
      ),
    },
    type: "text",
  }}
/>
        `,
      },
    },
  },
  args: {
    name: 'Search',
    label: 'Label',
    placeholder: 'Placeholder',
    maxLength: 8,
    size: 'small',
    error: false,
    readOnly: false,
    isRequired: false,
    disabled: false,
    type: 'text',
  },
  render: function Search({
    name,
    label,
    placeholder,
    maxLength,
    size,
    error,
    readOnly,
    isRequired,
    disabled,
    type,
    helperText,
  }) {
    const methods = useForm();

    if (error) {
      methods.setError(name, { message: 'Please insert a valid value' });
    } else {
      methods.clearErrors(name);
    }

    return (
      <FormProvider {...methods}>
        <TextInput
          isRequired={isRequired}
          name={name}
          label={label}
          textFieldProps={{
            InputProps: {
              startAdornment: (
                <InputAdornment position="start">
                  <SearchOutlined />
                </InputAdornment>
              ),
            },
            inputProps: {
              maxLength,
              readOnly,
            },
            helperText,
            disabled,
            error,
            placeholder,
            size,
            type,
          }}
        />
      </FormProvider>
    );
  },
};

export const TextArea: Story = {
  parameters: {
    docs: {
      source: {
        code: `
          <TextInput
  name="TextInput"
  label="Label"
  isRequired={false}
  textFieldProps={{
    disabled: false,
    error: false,
    placeholder: "Placeholder",
    size: "small",
    type: "text",
    multiline: true,
    minRows: 3,
    maxRows: 5,
    inputProps: {
      readOnly: false,
      maxLength: 16,
    },
  }}
/>
        `,
      },
    },
  },
  args: {
    name: 'TextArea',
    label: 'Label',
    placeholder: 'Placeholder',
    maxLength: 255,
    size: 'small',
    error: false,
    readOnly: false,
    isRequired: false,
    multiline: true,
    minRows: 3,
    maxRows: 5,
  },
};
