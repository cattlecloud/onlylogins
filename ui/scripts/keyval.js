// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

// keyval.js implements client-side Copy / Show / Hide behavior for keyvals.
//
//   Copy: writes the plaintext value to the clipboard and briefly swaps the
//         copy icon for a checkmark on success.
//
//   Show: replaces the masked display (•••) in the value element with the
//         plaintext, and swaps the show icon for the hide icon.
//
//   Hide: replaces the plaintext value with the masked bullets, and swaps
//         the hide icon back to the show icon.
(function () {
  'use strict';

  var COPY_FEEDBACK_MS = 1500;
  var COPY_ICON = '/ui/images/copy.svg';
  var CHECK_ICON = '/ui/images/check.svg';
  var SHOW_ICON = '/ui/images/show.svg';
  var HIDE_ICON = '/ui/images/hide.svg';

  function setImg(button, src, alt) {
    var img = button.querySelector('img');
    if (img) {
      img.src = src;
      img.alt = alt;
    }
  }

  function withRevert(button, newTitle, ms) {
    var previous = button.title;
    button.title = newTitle;
    setTimeout(function () {
      button.title = previous;
    }, ms);
  }

  function setIcon(button, src, alt, title) {
    setImg(button, src, alt);
    button.title = title;
  }

  function flashCheck(button) {
    setImg(button, CHECK_ICON, 'Copied!');
    setTimeout(function () {
      setImg(button, COPY_ICON, 'Copy');
    }, COPY_FEEDBACK_MS);
  }

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

    // Clicks land on the icon, so resolve the owning button.
    var copyBtn = target.closest('.keyval-copy-btn');
    var showBtn = target.closest('.keyval-show-btn');
    var hideBtn = target.closest('.keyval-hide-btn');

    // Copy: write plaintext value to clipboard, then flash a checkmark.
    if (copyBtn) {
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(plaintext).then(
          function () { flashCheck(copyBtn); },
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
      showBtn.classList.remove('keyval-show-btn');
      showBtn.classList.add('keyval-hide-btn');
      return;
    }

    // Hide: swap plaintext -> dots, flip icon and tooltip back to Show.
    if (hideBtn) {
      valueEl.textContent = maskFor(plaintext);
      setIcon(hideBtn, SHOW_ICON, 'Show', 'Show');
      hideBtn.classList.remove('keyval-hide-btn');
      hideBtn.classList.add('keyval-show-btn');
      return;
    }
  });
})();
