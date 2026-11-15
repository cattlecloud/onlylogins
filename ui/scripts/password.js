// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

// password.js implements client-side Copy / Show / Hide behavior for passwords
//
//   Copy: writes the plaintext to the clipboard via navigator.clipboard.writeText
//         with a brief confirmation in the button itself.
//
//   Show: replaces the masked display (•••) in the password element
//         with the plaintext.
//
//   Hide: replaces the plain text password with the masked bullets
(function () {
  'use strict';

  var COPY_FEEDBACK_MS = 1500;

  function withRevert(button, newText, ms) {
    var previous = button.textContent;
    button.textContent = newText;
    setTimeout(function () {
      button.textContent = previous;
    }, ms);
  }

  document.addEventListener('click', function (event) {
    var target = event.target;
    if (!(target instanceof Element)) {
      return;
    }

    var row = target.closest('.password-row');
    if (!row) {
      return;
    }

    var valueEl = row.querySelector('.password-value');
    if (!valueEl) {
      return;
    }
    var plaintext = row.getAttribute('data-password') || '';

    // Copy: write plaintext to system clipboard, then flash confirmation.
    if (target.matches('.password-copy-btn')) {
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(plaintext).then(
          function () { withRevert(target, 'Copied!', COPY_FEEDBACK_MS); },
          function () { withRevert(target, 'Copy failed', COPY_FEEDBACK_MS); }
        );
      } else {
        withRevert(target, 'Copy unsupported', COPY_FEEDBACK_MS);
      }
      return;
    }

    // Show: swap dots -> plaintext, flip button to Hide.
    if (target.matches('.password-show-btn')) {
      valueEl.textContent = plaintext;
      target.textContent = 'Hide';
      target.title = 'Hide';
      target.classList.remove('password-show-btn');
      target.classList.add('password-hide-btn');
      return;
    }

    // Hide: swap plaintext -> dots, flip button back to Show.
    if (target.matches('.password-hide-btn')) {
      valueEl.textContent = maskFor(plaintext);
      target.textContent = 'Show';
      target.title = 'Show';
      target.classList.remove('password-hide-btn');
      target.classList.add('password-show-btn');
      return;
    }
  });
})();
