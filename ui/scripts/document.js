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
//
//   Secret input: password fields are masked with CSS instead of using
//         type="password", so that the browser never treats the value as a
//         credential for this site. The toggle reveals what was typed, and
//         the generate button fills the field with a random password.
(function () {
  'use strict';

  const COPY_FEEDBACK_MS = 1500;
  const COPY_ICON = '/ui/images/copy.svg';
  const CHECK_ICON = '/ui/images/check.svg';
  const SHOW_ICON = '/ui/images/show.svg';
  const HIDE_ICON = '/ui/images/hide.svg';
  const MASK_MAX = 16;

  // Password generation. Ambiguous characters (l, I, O, 0, 1) are left out
  // so a generated password is easy to read back and type somewhere else.
  const PASSWORD_LENGTH = 20;
  const PASSWORD_LOWER = 'abcdefghijkmnopqrstuvwxyz';
  const PASSWORD_UPPER = 'ABCDEFGHJKLMNPQRSTUVWXYZ';
  const PASSWORD_DIGITS = '23456789';
  const PASSWORD_SYMBOLS = '!@#$%^&*-_=+?';

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

  // Returns a random integer in [0, max). Values are drawn from the
  // cryptographic random source and rejected when they would introduce a
  // modulo bias.
  const randomInt = (max) => {
    const buffer = new Uint32Array(1);
    const limit = Math.floor(0x100000000 / max) * max;
    let value;
    do {
      window.crypto.getRandomValues(buffer);
      value = buffer[0];
    } while (value >= limit);
    return value % max;
  };

  const randomChar = (charset) => charset[randomInt(charset.length)];

  const shuffle = (chars) => {
    for (let i = chars.length - 1; i > 0; i--) {
      const j = randomInt(i + 1);
      const swap = chars[i];
      chars[i] = chars[j];
      chars[j] = swap;
    }
    return chars;
  };

  // generatePassword builds a random password that always contains at least
  // one lowercase letter, one uppercase letter, one digit, and one symbol.
  const generatePassword = () => {
    const classes = [PASSWORD_LOWER, PASSWORD_UPPER, PASSWORD_DIGITS, PASSWORD_SYMBOLS];
    const required = classes.map(randomChar);
    const all = classes.join('');
    const rest = Array.from({ length: PASSWORD_LENGTH - required.length }, () => randomChar(all));
    return shuffle([...required, ...rest]).join('');
  };

  // secretInputFor finds the masked input belonging to the button's form.
  const secretInputFor = (button) => button.closest('form')?.querySelector('.secret-input') ?? null;

  // Toggle: show or hide the contents of a masked secret input.
  const handleSecretToggle = (target) => {
    const toggle = target.closest('.secret-toggle-btn');
    if (!toggle) {
      return false;
    }

    const input = secretInputFor(toggle);
    if (!input) {
      return false;
    }

    const revealed = input.classList.toggle('revealed');
    setIcon(toggle, revealed ? HIDE_ICON : SHOW_ICON, revealed ? 'Hide' : 'Show', revealed ? 'Hide' : 'Show');
    toggle.setAttribute('aria-label', revealed ? 'Hide password' : 'Show password');
    return true;
  };

  // Generate: fill the masked secret input with a random password.
  const handleSecretGenerate = (target) => {
    const button = target.closest('.secret-generate-btn');
    if (!button) {
      return false;
    }

    const input = secretInputFor(button);
    if (!input) {
      return false;
    }

    input.value = generatePassword();
    input.focus();
    input.select();
    return true;
  };

  const handleNoteReveal = (target) => {
    if (!target.matches('.note-reveal-btn')) {
      return false;
    }

    const area = target.closest('.note-area');
    if (!area) {
      return false;
    }

    const panel = area.querySelector('.note-panel');
    if (!panel) {
      return false;
    }

    const content = panel.querySelector('.note-content');
    if (!content) {
      return false;
    }

    const plaintext = panel.getAttribute('data-note') || '';
    content.textContent = plaintext;
    content.hidden = false;
    target.hidden = true;

    const actions = area.querySelector('.note-actions');
    if (actions) {
      actions.hidden = false;
    }
    return true;
  };

  document.addEventListener('click', (event) => {
    const target = event.target;
    if (!(target instanceof Element)) {
      return;
    }

    if (handleSecretToggle(target)) {
      return;
    }
    if (handleSecretGenerate(target)) {
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
    handleNoteReveal(target);
  });
})();
