import { FC, PropsWithChildren, useEffect, useState } from 'react';

interface Props extends PropsWithChildren {
  delay?: number;
}

const DelayedRender: FC<Props> = ({ delay, children }) => {
  const [isRendered, setIsRendered] = useState(false);

  useEffect(() => {
    const timeoutId = setTimeout(() => setIsRendered(true), delay);
    return () => clearTimeout(timeoutId);
  }, [delay]);

  return isRendered ? children : null;
};

export default DelayedRender;
