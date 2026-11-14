// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

// folder.js implements the "example chip" behaviour on the new-document form.
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
})();
