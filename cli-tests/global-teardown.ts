import { FullConfig } from '@playwright/test';
import { cleanUpDocker } from '@helpers/containers';

const globalTeardown = async (config: FullConfig) => {
  await cleanUpDocker();
};

export default globalTeardown;
