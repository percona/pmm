import { act, fireEvent, render, screen } from '@testing-library/react';
import { PMM_DOCS_INVENTORY_SERVICES_URL } from 'lib/constants';
import { Messages } from '../Services.messages';
import { AddressHeader } from './AddressHeader';

describe('AddressHeader', () => {
  it('shows the column name with an info icon', () => {
    render(<AddressHeader />);

    expect(screen.getByText('Address')).toBeInTheDocument();
    expect(screen.getByTestId('address-info-icon')).toBeInTheDocument();
  });

  it('explains the address with a docs link on hover', async () => {
    render(<AddressHeader />);

    fireEvent.mouseOver(screen.getByTestId('address-info-icon'));

    const tooltip = await screen.findByRole('tooltip');
    expect(tooltip).toHaveTextContent(Messages.addressInfo);
    const link = screen.getByTestId('address-info-docs-link');
    expect(link).toHaveAttribute('href', PMM_DOCS_INVENTORY_SERVICES_URL);
    expect(link).toHaveAttribute('target', '_blank');
  });

  it('explains the address when the icon gets keyboard focus', async () => {
    render(<AddressHeader />);

    const icon = screen.getByTestId('address-info-icon');
    fireEvent.keyDown(document.body, { key: 'Tab' });
    act(() => icon.focus());

    expect(icon).toHaveFocus();
    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      Messages.addressInfo
    );
  });
});
