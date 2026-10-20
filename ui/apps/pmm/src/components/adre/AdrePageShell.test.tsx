import { useTheme } from '@mui/material';
import { render, screen } from '@testing-library/react';
import { TestWrapper } from 'utils/testWrapper';
import { AdrePageShell } from './AdrePageShell';
import { adreColors } from './adreAnswerSx';

const ThemeProbe = () => {
  const theme = useTheme();
  return (
    <span data-testid="probe">
      {theme.palette.mode} {theme.palette.background.paper}
    </span>
  );
};

describe('AdrePageShell', () => {
  it('heads the page and gives its content the Assistant theme', () => {
    render(
      <TestWrapper>
        <AdrePageShell
          title="AI usage"
          subtitle="Token and cost usage"
          actions={<button type="button">Export CSV</button>}
        >
          <ThemeProbe />
        </AdrePageShell>
      </TestWrapper>
    );

    expect(
      screen.getByRole('heading', { level: 1, name: 'AI usage' })
    ).toBeInTheDocument();
    expect(screen.getByText('Token and cost usage')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Export CSV' })
    ).toBeInTheDocument();
    expect(screen.getByTestId('probe')).toHaveTextContent(
      `dark ${adreColors.panel}`
    );
  });
});
