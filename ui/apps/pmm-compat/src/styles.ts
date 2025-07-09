import { LOCATORS } from 'lib/constants';

export const applyCustomStyles = () => {
  const style = document.createElement('style');

  // Hide toolbar elements
  style.innerText = `
    ${LOCATORS.menuToggle},
    ${LOCATORS.helpButton},
    ${LOCATORS.searchButton},
    ${LOCATORS.profileButton} {
      display: none;
    }

    ${LOCATORS.commandPaletteTrigger},
    ${LOCATORS.searchButton} {
      visibility: hidden;
      order: -1;
    }
  `;

  document.head.appendChild(style);
};
