// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

// secrets.js implements the client-side interactions for revealing and copying
// stored secret values. It is shared between the document and folder pages.
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
(function () {
  'use strict';

  const COPY_FEEDBACK_MS = 1500;
  const COPY_ICON = '/ui/images/copy.svg';
  const CHECK_ICON = '/ui/images/check.svg';
  const SHOW_ICON = '/ui/images/show.svg';
  const HIDE_ICON = '/ui/images/hide.svg';
  const MASK_MAX = 16;

  // Secret fields support Copy / Show / Hide. Passwords and keyvals share the
  // same structure, so each is described by the class names and data attribute
  // that differ between them.
  const SECRET_FIELDS = [
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

  const maskFor = (plaintext) => {
    let count = [...plaintext].length;
    if (count > MASK_MAX) {
      count = MASK_MAX;
    }
    return '\u2022'.repeat(count);
  };

  const setImg = (button, src, alt) => {
    const img = button.querySelector('img');
    if (img) {
      img.src = src;
      img.alt = alt;
    }
  };

  const withRevert = (button, newTitle, ms) => {
    const previous = button.title;
    button.title = newTitle;
    setTimeout(() => {
      button.title = previous;
    }, ms);
  };

  const setIcon = (button, src, alt, title) => {
    setImg(button, src, alt);
    button.title = title;
  };

  const flashCheck = (button, ms) => {
    setImg(button, CHECK_ICON, 'Copied!');
    setTimeout(() => {
      setImg(button, COPY_ICON, 'Copy');
    }, ms);
  };

  const copyToClipboard = (button, value, ms) => {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(value).then(
        () => flashCheck(button, ms),
        () => withRevert(button, 'Copy failed', ms)
      );
    } else {
      withRevert(button, 'Copy unsupported', ms);
    }
  };

  // Weblinks and usernames only support Copy, and their rows share the same
  // data-value attribute, so they are handled together.
  const handleCopyOnly = (target) => {
    // Clicks land on the icon, so resolve the owning button.
    const copyBtn = target.closest('.weblink-copy-btn, .username-copy-btn');
    if (!copyBtn) {
      return false;
    }

    const row = copyBtn.closest('.weblink-row, .username-row');
    if (!row) {
      return false;
    }

    const value = row.getAttribute('data-value') || '';
    if (!value) {
      return false;
    }

    copyToClipboard(copyBtn, value, COPY_FEEDBACK_MS);
    return true;
  };

  const handleSecret = (field, target) => {
    const row = target.closest('.' + field.row);
    if (!row) {
      return false;
    }

    const valueEl = row.querySelector('.' + field.value);
    if (!valueEl) {
      return false;
    }
    const plaintext = row.getAttribute(field.data) || '';

    // Clicks land on the icon, so resolve the owning button.
    const copyBtn = target.closest('.' + field.copy);
    const showBtn = target.closest('.' + field.show);
    const hideBtn = target.closest('.' + field.hide);

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
  };

  document.addEventListener('click', (event) => {
    const target = event.target;
    if (!(target instanceof Element)) {
      return;
    }

    if (handleCopyOnly(target)) {
      return;
    }
    for (const field of SECRET_FIELDS) {
      if (handleSecret(field, target)) {
        return;
      }
    }
  });
})();
