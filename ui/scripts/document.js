// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

// document.js implements the client-side interactions on the document page:
//
//   Copy: writes the row's plaintext value (stored in a data-* attribute) to
//         the system clipboard, briefly swapping the copy icon for a checkmark
//         on success. Applies to weblinks, usernames, passwords, and keyvals.
//
//   Show: replaces the masked display (•••) with the plaintext, and swaps the
//         show icon for the hide icon. Applies to passwords and keyvals.
//
//   Hide: replaces the plaintext with the masked bullets, and swaps the hide
//         icon back to the show icon.
//
//   Note: reveals a note's content in place. This is intentionally one-way
//         (no Hide, no Copy).
(function () {
  'use strict';

  var COPY_FEEDBACK_MS = 1500;
  var COPY_ICON = '/ui/images/copy.svg';
  var CHECK_ICON = '/ui/images/check.svg';
  var SHOW_ICON = '/ui/images/show.svg';
  var HIDE_ICON = '/ui/images/hide.svg';
  var MASK_MAX = 16;

  // Secret fields support Copy / Show / Hide. Passwords and keyvals share the
  // same structure, so each is described by the class names and data attribute
  // that differ between them.
  var SECRET_FIELDS = [
    {
      row: 'password-row',
      value: 'password-value',
      data: 'data-password',
      copy: 'password-copy-btn',
      show: 'password-show-btn',
      hide: 'password-hide-btn'
    },
    {
      row: 'keyval-row',
      value: 'keyval-value',
      data: 'data-value',
      copy: 'keyval-copy-btn',
      show: 'keyval-show-btn',
      hide: 'keyval-hide-btn'
    }
  ];

  function maskFor(plaintext) {
    var count = Array.from(plaintext).length;
    if (count > MASK_MAX) {
      count = MASK_MAX;
    }
    return '\u2022'.repeat(count);
  }

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

  function flashCheck(button, ms) {
    setImg(button, CHECK_ICON, 'Copied!');
    setTimeout(function () {
      setImg(button, COPY_ICON, 'Copy');
    }, ms);
  }

  function copyToClipboard(button, value, ms) {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(value).then(
        function () { flashCheck(button, ms); },
        function () { withRevert(button, 'Copy failed', ms); }
      );
    } else {
      withRevert(button, 'Copy unsupported', ms);
    }
  }

  // Weblinks and usernames only support Copy, and their rows share the same
  // data-value attribute, so they are handled together.
  function handleCopyOnly(target) {
    // Clicks land on the icon, so resolve the owning button.
    var copyBtn = target.closest('.weblink-copy-btn, .username-copy-btn');
    if (!copyBtn) {
      return false;
    }

    var row = copyBtn.closest('.weblink-row, .username-row');
    if (!row) {
      return false;
    }

    var value = row.getAttribute('data-value') || '';
    if (!value) {
      return false;
    }

    copyToClipboard(copyBtn, value, COPY_FEEDBACK_MS);
    return true;
  }

  function handleSecret(field, target) {
    var row = target.closest('.' + field.row);
    if (!row) {
      return false;
    }

    var valueEl = row.querySelector('.' + field.value);
    if (!valueEl) {
      return false;
    }
    var plaintext = row.getAttribute(field.data) || '';

    // Clicks land on the icon, so resolve the owning button.
    var copyBtn = target.closest('.' + field.copy);
    var showBtn = target.closest('.' + field.show);
    var hideBtn = target.closest('.' + field.hide);

    // Copy: write plaintext to clipboard, then flash a checkmark.
    if (copyBtn) {
      copyToClipboard(copyBtn, plaintext, COPY_FEEDBACK_MS);
      return true;
    }

    // Show: swap dots -> plaintext, flip icon and tooltip to Hide.
    if (showBtn) {
      valueEl.textContent = plaintext;
      setIcon(showBtn, HIDE_ICON, 'Hide', 'Hide');
      showBtn.classList.remove(field.show);
      showBtn.classList.add(field.hide);
      return true;
    }

    // Hide: swap plaintext -> dots, flip icon and tooltip back to Show.
    if (hideBtn) {
      valueEl.textContent = maskFor(plaintext);
      setIcon(hideBtn, SHOW_ICON, 'Show', 'Show');
      hideBtn.classList.remove(field.hide);
      hideBtn.classList.add(field.show);
      return true;
    }

    return false;
  }

  function handleNoteReveal(target) {
    if (!target.matches('.note-reveal-btn')) {
      return false;
    }

    var area = target.closest('.note-area');
    if (!area) {
      return false;
    }

    var panel = area.querySelector('.note-panel');
    if (!panel) {
      return false;
    }

    var content = panel.querySelector('.note-content');
    if (!content) {
      return false;
    }

    var plaintext = panel.getAttribute('data-note') || '';
    content.textContent = plaintext;
    content.hidden = false;
    target.hidden = true;

    var actions = area.querySelector('.note-actions');
    if (actions) {
      actions.hidden = false;
    }
    return true;
  }

  document.addEventListener('click', function (event) {
    var target = event.target;
    if (!(target instanceof Element)) {
      return;
    }

    if (handleCopyOnly(target)) {
      return;
    }
    for (var i = 0; i < SECRET_FIELDS.length; i++) {
      if (handleSecret(SECRET_FIELDS[i], target)) {
        return;
      }
    }
    handleNoteReveal(target);
  });
})();
