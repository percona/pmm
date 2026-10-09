import { useFirstLoginRedirect } from 'hooks/useFirstLoginRedirect';

/** Side effect only: navigates first-time users to the welcome page. Renders nothing. */
const FirstLoginRedirect = () => {
  useFirstLoginRedirect();
  return null;
};

export default FirstLoginRedirect;
