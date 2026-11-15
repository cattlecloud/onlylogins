// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

// keyval.js implements the client-side how-hide behavior for keyvals.
(function () {
  'use strict';

  document.addEventListener('click', function (event) {
    var target = event.target;
    if (!(target instanceof Element)) {
      return;
    }

    var row = target.closest('.keyval-row');
    if (!row) {
      return;
    }

    var valueEl = row.querySelector('.keyval-value');
    if (!valueEl) {
      return;
    }
    var plaintext = row.getAttribute('data-value') || '';

    // show: swap dots -> plaintext, flip button to Hide.
    if (target.matches('.keyval-show-btn')) {
      valueEl.textContent = plaintext;
      target.textContent = 'Hide';
      target.title = 'Hide';
      target.classList.remove('keyval-show-btn');
      target.classList.add('keyval-hide-btn');
      return;
    }

    // hide: swap plaintext -> dots, flip button back to Show.
    if (target.matches('.keyval-hide-btn')) {
      valueEl.textContent = maskFor(plaintext);
      target.textContent = 'Show';
      target.title = 'Show';
      target.classList.remove('keyval-hide-btn');
      target.classList.add('keyval-show-btn');
      return;
    }
  });
})();
