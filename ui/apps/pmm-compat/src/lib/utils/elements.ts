export const waitForElement = async (selector: string, timeout = 5000) =>
  new Promise<HTMLElement | null>((resolve) => {
    const startTime = Date.now();
    const check = () => {
      const element = document.querySelector(selector);
      if (element) {
        resolve(element as HTMLElement);
      } else if (Date.now() - startTime < timeout) {
        requestAnimationFrame(check);
      } else {
        resolve(null);
      }
    };
    check();
  });
