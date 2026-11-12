import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { FormProvider, useForm } from 'react-hook-form';
import TextArray from './text-array';

const Wrapper = ({ children }: { children: React.ReactNode }) => {
  const methods = useForm({
    defaultValues: {
      people: [
        {
          person: 'John',
        },
      ],
    },
  });

  return <FormProvider {...methods}>{children}</FormProvider>;
};

it('should add / remove fields', () => {
  render(
    <Wrapper>
      <TextArray fieldName="people" fieldKey="person" />
    </Wrapper>
  );

  expect(screen.getByTestId('text-input-people.0.person')).toBeInTheDocument();

  fireEvent.click(screen.getByTestId('add-text-input-button'));

  expect(screen.getByTestId('text-input-people.1.person')).toBeInTheDocument();

  fireEvent.click(screen.getByTestId('delete-text-input-0-button'));

  expect(
    screen.queryByTestId('text-input-people.1.person')
  ).not.toBeInTheDocument();
});
