import { Component, PropsWithChildren, ReactNode } from 'react';

interface IconBoundaryProps extends PropsWithChildren {
  fallback: ReactNode;
}

interface IconBoundaryState {
  failed: boolean;
}

/**
 * Keeps a missing icon from taking the application down with it. Icons are the only
 * chunks this app loads lazily, so an upgrade that deletes the one a page asks for
 * rejects its import, and React unwinds a render error with no boundary above it all
 * the way to the root, blanking the page. Falling back leaves a gap where the icon
 * was, which is what lets the version watcher still offer a reload.
 */
export class IconBoundary extends Component<
  IconBoundaryProps,
  IconBoundaryState
> {
  state: IconBoundaryState = { failed: false };

  static getDerivedStateFromError(): IconBoundaryState {
    return { failed: true };
  }

  render() {
    return this.state.failed ? this.props.fallback : this.props.children;
  }
}
