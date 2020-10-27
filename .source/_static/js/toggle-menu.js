const styleDomEl = document.createElement('style');
styleDomEl.innerHTML =
  '.sphinxsidebarwrapper {' +
  'display: none;' +
  'padding-right: 0 !important;' +
  '}' +
  '.sphinxsidebarwrapper ul {' +
  'list-style: none;' +
  '}' +
  '.sphinxsidebarwrapper > ul {' +
  'padding-left: 0;' +
  '}' +
  '.sphinxsidebarwrapper > ul > li {' +
  'padding: 0 0 10px 0;' +
  'margin: 0;' +
  '}' +
  '.custom-button {' +
  'cursor: pointer;' +
  'display: inline-flex;' +
  'justify-content: center;' +
  'align-items: center;' +
  'width: 10px;' +
  'margin-right: 5px;' +
  'margin-bottom: 0;' +
  'font-size: 18px;' +
  'font-weight: 400;' +
  'border: none;' +
  'background-color: transparent;' +
  'outline: none;' +
  '}' +
  '.custom-button ~ ul {' +
  'display: none;' +
  '}' +
  '.custom-button--main-active {' +
  'background-color: #e3e3e3' +
  '}' +
  '.custom-button.custom-button--active ~ ul {' +
  'display: block;' +
  '}' +
  '.custom-button:before {' +
  'content: \'+\';' +
  '}' +
  '.custom-button.custom-button--active:before {' +
  'content: \'-\';' +
  '}';
document.head.appendChild(styleDomEl);

setTimeout(() => {
  const asideMenu = document.getElementsByClassName('sphinxsidebarwrapper')[0];
  hideSubMenus();
  asideMenu.style.display = 'block';
}, 500);

function hideSubMenus() {
  const asideMenu = document.getElementsByClassName('sphinxsidebarwrapper')[0];
  const activeCheckboxClass = 'custom-button--active';
  const activeBackgroundClass = 'custom-button--main-active';
  const links = Array.from(asideMenu.getElementsByTagName('a'));
  const accordionLinks = links.filter(links => links.nextElementSibling && links.nextElementSibling.localName === 'ul');
  const simpleLinks = links.filter(links => !links.nextElementSibling && links.parentElement.localName === 'li');

  simpleLinks.forEach(simpleLink => {
    simpleLink.parentElement.style.listStyleType = 'disc';
    simpleLink.parentElement.style.marginLeft = '20px';
  });

  accordionLinks.forEach((link, index) => {
    insertButton(link, index);
  });

  const buttons = Array.from(document.getElementsByClassName('custom-button'));

  buttons.forEach(button => button.addEventListener('click', event => {
    event.preventDefault();
    const current = event.currentTarget;
    const parent = current.parentElement;
    const isMain = Array.from(parent.classList).includes('toctree-l1');
    const isMainActive = Array.from(parent.classList).includes(activeBackgroundClass);
    const targetClassList = Array.from(current.classList);

    toggleElement(targetClassList.includes(activeCheckboxClass), current, activeCheckboxClass);
    if (isMain) {
      toggleElement(isMainActive, parent, activeBackgroundClass);
    }
  }));

  asideMenu.parentNode.insertBefore(styleDomEl, asideMenu);
}

function toggleElement(condition, item, className) {
  const isButton = item.localName === 'button';

  if (!condition) {
    const previousActive = Array.from(item.parentElement.parentElement.getElementsByClassName('list-item--active'));
    if (isButton) {
      localStorage.setItem(item.id, 'true');

      if (previousActive.length) {
        previousActive.forEach(previous => {

          const previousActiveButtons = Array.from(previous.getElementsByClassName('custom-button--active'));
          removeClass(previous, ['list-item--active', 'custom-button--main-active']);

          if (previousActiveButtons.length) {
            previousActiveButtons.forEach(previousButton => {

              removeClass(previousButton, 'custom-button--active');
              localStorage.removeItem(previousButton.id);
            });
          }
        })
      }
    }

    addClass(item, className);
    addClass(item.parentElement, 'list-item--active');
  } else {
    removeClass(item, className);
    removeClass(item.parentElement, 'list-item--active');

    if (isButton) {
      localStorage.removeItem(item.id);
    }
  }
}

function addClass(item, classes) {
  item.classList.add(...Array.isArray(classes) ? classes : [classes]);
}

function removeClass(item, classes) {
  item.classList.remove(...Array.isArray(classes) ? classes : [classes]);
}

function insertButton(element, id) {
  const button = document.createElement('button');
  const isMain = Array.from(element.parentElement.classList).includes('toctree-l1');
  button.id = id;
  addClass(button, 'custom-button');
  if (localStorage.getItem(id)) {
    addClass(button, 'custom-button--active');
    addClass(element.parentElement, 'list-item--active');
    if (isMain) {
      addClass(element.parentElement, 'custom-button--main-active');
    }
  }
  element.insertAdjacentElement('beforebegin', button);
}
