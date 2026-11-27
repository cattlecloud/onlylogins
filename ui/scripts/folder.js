// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

// folder.js implements the "example chip" behaviour on the new-document form,
// and toggles the per-document weblink and password dropdowns in the documents
// list.
(function () {
  'use strict';

  document.addEventListener('click', function (event) {
    var target = event.target;
    if (!(target instanceof Element)) {
      return;
    }
    if (!target.matches('.example-chip')) {
      return;
    }

    var input = document.getElementById('document-label');
    if (!input) {
      return;
    }

    var value = target.getAttribute('data-example') || target.textContent || '';
    input.value = value;
    input.focus();
    input.select();
  });

  // htmx lazily populates the weblink/password dropdown on click. A second
  // click on the same button clears it instead of re-fetching, so the button
  // toggles the menu open and closed.
  document.addEventListener('click', function (event) {
    var target = event.target;
    if (!(target instanceof Element)) {
      return;
    }

    var btn = target.closest('.weblink-menu-btn, .password-menu-btn');
    if (!btn) {
      return;
    }

    var li = btn.closest('li');
    if (!li) {
      return;
    }

    var menu = btn.classList.contains('weblink-menu-btn')
      ? li.querySelector('.weblink-menu')
      : li.querySelector('.password-menu');

    if (menu && menu.innerHTML.trim() !== '') {
      event.preventDefault();
      event.stopPropagation();
      menu.innerHTML = '';
    }
  }, true);
})();
