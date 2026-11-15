// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

// copy.js implements click-to-copy for weblinks and usernames.
//
//   Copy: writes the row's plaintext value (stored in data-value) to the
//         system clipboard, briefly swapping the copy icon for a checkmark
//         on success.
(function () {
  'use strict';

  var COPY_FEEDBACK_MS = 1500;
  var COPY_ICON = '/ui/images/copy.svg';
  var CHECK_ICON = '/ui/images/check.svg';

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

    // Clicks land on the icon, so resolve the owning button.
    var copyBtn = target.closest('.weblink-copy-btn, .username-copy-btn');
    if (!copyBtn) {
      return;
    }

    var row = copyBtn.closest('.weblink-row, .username-row');
    if (!row) {
      return;
    }

    var value = row.getAttribute('data-value') || '';
    if (!value) {
      return;
    }

    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(value).then(
        function () { flashCheck(copyBtn); },
        function () { withRevert(copyBtn, 'Copy failed', COPY_FEEDBACK_MS); }
      );
    } else {
      withRevert(copyBtn, 'Copy unsupported', COPY_FEEDBACK_MS);
    }
  });
})();
