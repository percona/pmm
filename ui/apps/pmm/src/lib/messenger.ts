import { CrossFrameMessenger } from '@pmm/shared';

/**
 * Process-lifetime channel to the Grafana iframe. It is registered here rather
 * than from a component so listeners survive the iframe mounting, unmounting
 * and remounting; the target is resolved lazily for the same reason.
 */
const messenger = new CrossFrameMessenger('PMM')
  .setTargetOrigin(window.location.origin)
  .setTargetResolver(
    () =>
      document.querySelector<HTMLIFrameElement>('#grafana-iframe')
        ?.contentWindow
  )
  .register();

export default messenger;
