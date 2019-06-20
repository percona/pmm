setTimeout(() => {
  makeSelect();
}, 500);

function makeSelect() {
  const custom_select = document.getElementById('custom_select');
  const select_active_option = custom_select.getElementsByClassName('select-active-text')[0];
  const custom_select_list = document.getElementById('custom_select_list');

  select_active_option.innerHTML = window.location.href.includes('2.x') ?
    custom_select_list.getElementsByClassName('custom-select__option')[1].innerHTML :
    custom_select_list.getElementsByClassName('custom-select__option')[0].innerHTML;

  document.addEventListener('click', event => {
    if (event.target.parentElement.id === 'custom_select' || event.target.id === 'custom_select') {
      custom_select_list.classList.toggle('select-hidden')
    }

    if (Array.from(event.target.classList).includes('custom-select__option')) {
      select_active_option.innerHTML = event.target.innerHTML;
    }

    if (event.target.id !== 'custom_select' && event.target.parentElement.id !== 'custom_select') {
      custom_select_list.classList.add('select-hidden')
    }

  });
}