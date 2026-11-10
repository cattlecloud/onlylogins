// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

// share.js implements the click-to-copy behavior for share-link buttons.
//
//   Copy: writes the share URL to the system clipboard via
//         navigator.clipboard.writeText, then briefly flips the button into
//         a `.copied` state to confirm the action.
(function () {
  'use strict';

  document.addEventListener('click', function (event) {
    var target = event.target;
    if (!(target instanceof Element)) {
      return;
    }
    if (!target.matches('.copy-btn')) {
      return;
    }

    // the share URL is embedded directly on the input adjacent to the button
    var article = target.closest('article');
    if (!article) {
      return;
    }
    var input = article.querySelector('input[type="text"]');
    if (!input) {
      return;
    }

    var address = input.value || '';
    if (!address) {
      return;
    }

    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(address).then(
        function () {
          target.classList.add('copied');
          setTimeout(function () {
            target.classList.remove('copied');
          }, 2000);
        },
        function () {
          // ignore failures; the URL is still visible to copy manually
        }
      );
    }
  });

  // also support inline onclick="copylink(...)" handlers (used by the
  // HTMX-swapped share creation result)
  window.copylink = function (element, timeout, address) {
    if (!(element instanceof Element)) {
      return;
    }
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(address).then(function () {
        element.classList.add('copied');
        setTimeout(function () {
          element.classList.remove('copied');
        }, timeout);
      });
    }
  };
})();
