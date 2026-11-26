import { render } from '@testing-library/react';
import { ReleaseNotes } from './ReleaseNotes';

describe('ReleaseNotes', () => {
  it("isn't shown if it's empty", () => {
    const { container } = render(<ReleaseNotes content="" />);

    expect(container).toBeEmptyDOMElement();
  });
});
