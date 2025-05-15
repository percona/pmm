export const applyCustomStyles = () => {
  console.log('applying custom styles');

  const style = document.createElement('style');
  style.innerText = `
              #mega-menu-toggle,
              header > div:first-child > div:nth-child(2) {
                display: none;
              }

              header {
                padding: 8px;
                flex-direction: row !important;
                border-bottom: 1px solid rgba(36, 41, 46, 0.12)
              }

              header > div:first-child {
                flex: 1;
              }

              header > div {
                border-bottom: 0 !important;
              }

              [class*="canvas-wrapper"] > div {
                top: 57px;
              }
              `;
  document.head.appendChild(style);
};
