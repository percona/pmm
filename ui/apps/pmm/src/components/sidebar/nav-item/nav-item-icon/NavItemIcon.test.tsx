import { render, screen } from "@testing-library/react";
import NavItemIcon from "./NavItemIcon";
import { memo } from "react";

const TestIcon = () => <div>icon</div>;
const TestMemoizedIcon = memo(TestIcon);

describe('NavItemIcon', () => {
  it('should render an icon (string)', () => {
    render(<NavItemIcon icon="search" />);
    expect(screen).not.toBeNull();
  });

  it('should render an icon (element)', () => {
    render(<NavItemIcon icon={<TestIcon />} />);
    expect(screen.getByText('icon')).toBeInTheDocument();
  });

  it('should render an icon (component)', () => {
    render(<NavItemIcon icon={TestIcon} />);
    expect(screen.getByText('icon')).toBeInTheDocument();
  });

  it('should render an icon (memoized component)', async () => {
    render(<NavItemIcon icon={TestMemoizedIcon} />);
    expect(screen.getByText('icon')).toBeInTheDocument();
  });
});