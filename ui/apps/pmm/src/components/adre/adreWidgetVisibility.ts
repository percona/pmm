import { PMM_ASSISTANT_PATH } from 'lib/constants';

/** Whether the floating Assistant chat belongs on a page: the Assistant page and native QAN have their own. */
export const showsAdreWidget = (pathname: string) => {
  const assistantPage =
    pathname === PMM_ASSISTANT_PATH ||
    pathname.startsWith(`${PMM_ASSISTANT_PATH}/`);
  const nativeQan =
    pathname.includes('/qan') && !pathname.includes('/qan/ai-insights');
  return !assistantPage && !nativeQan;
};
