export const applyCustomStyles = () => {
  console.log('applying custom styles');

  const style = document.createElement('style');
  style.innerText = `
                  #mega-menu-toggle,
              header > div:first-child,
              header > div:nth-child(2) > div:first-of-type,
              header div[class*=NavToolbar-actions] > div:last-of-type,
              button[title="Toggle top search bar"]
               {
                display: none;
              }`;
  document.head.appendChild(style);
};
