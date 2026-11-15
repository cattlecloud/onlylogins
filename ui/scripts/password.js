// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

// password.js implements client-side Copy / Show / Hide behavior for passwords
//
//   Copy: writes the plaintext to the clipboard via navigator.clipboard.writeText
//         and briefly swaps the copy icon for a checkmark on success.
//
//   Show: replaces the masked display (•••) in the password element
//         with the plaintext, and swaps the show icon for the hide icon.
//
//   Hide: replaces the plain text password with the masked bullets, and
//         swaps the hide icon back to the show icon.
(function () {
  'use strict';

  var COPY_FEEDBACK_MS = 1500;
  var COPY_ICON = '/ui/images/copy.svg';
  var CHECK_ICON = '/ui/images/check.svg';
  var SHOW_ICON = '/ui/images/show.svg';
  var HIDE_ICON = '/ui/images/hide.svg';

  function withRevert(button, newTitle, ms) {
    var previous = button.title;
    button.title = newTitle;
    setTimeout(function () {
      button.title = previous;
    }, ms);
  }

  function setImg(button, src, alt) {
    var img = button.querySelector('img');
    if (img) {
      img.src = src;
      img.alt = alt;
    }
  }

  function setIcon(button, src, alt, title) {
    setImg(button, src, alt);
    button.title = title;
  }

  function flashCheck(button, ms) {
    setImg(button, CHECK_ICON, 'Copied!');
    setTimeout(function () {
      setImg(button, COPY_ICON, 'Copy');
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

    // Clicks land on the icon, so resolve the owning button.
    var copyBtn = target.closest('.password-copy-btn');
    var showBtn = target.closest('.password-show-btn');
    var hideBtn = target.closest('.password-hide-btn');

    // Copy: write plaintext to system clipboard, then briefly swap the
    // copy icon for a checkmark to confirm success.
    if (copyBtn) {
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(plaintext).then(
          function () { flashCheck(copyBtn, COPY_FEEDBACK_MS); },
          function () { withRevert(copyBtn, 'Copy failed', COPY_FEEDBACK_MS); }
        );
      } else {
        withRevert(copyBtn, 'Copy unsupported', COPY_FEEDBACK_MS);
      }
      return;
    }

    // Show: swap dots -> plaintext, flip icon and tooltip to Hide.
    if (showBtn) {
      valueEl.textContent = plaintext;
      setIcon(showBtn, HIDE_ICON, 'Hide', 'Hide');
      showBtn.classList.remove('password-show-btn');
      showBtn.classList.add('password-hide-btn');
      return;
    }

    // Hide: swap plaintext -> dots, flip icon and tooltip back to Show.
    if (hideBtn) {
      valueEl.textContent = maskFor(plaintext);
      setIcon(hideBtn, SHOW_ICON, 'Show', 'Show');
      hideBtn.classList.remove('password-hide-btn');
      hideBtn.classList.add('password-show-btn');
      return;
    }
  });
})();
