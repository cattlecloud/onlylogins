// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

// document.js implements the document-page-specific client-side interactions:
//
//   Note: reveals a note's content in place. This is intentionally one-way
//         (no Hide, no Copy).
//
//   Secret input: password fields are masked with CSS instead of using
//         type="password", so that the browser never treats the value as a
//         credential for this site. The toggle reveals what was typed, and
//         the generate button fills the field with a random password.
//
// Copy / Show / Hide of stored secret rows is handled by secrets.js, shared
// with the folder page.
(function () {
  'use strict';

  const SHOW_ICON = '/ui/images/show.svg';
  const HIDE_ICON = '/ui/images/hide.svg';

  // Password generation. Ambiguous characters (l, I, O, 0, 1) are left out
  // so a generated password is easy to read back and type somewhere else.
  const PASSWORD_LENGTH = 20;
  const PASSWORD_LOWER = 'abcdefghijkmnopqrstuvwxyz';
  const PASSWORD_UPPER = 'ABCDEFGHJKLMNPQRSTUVWXYZ';
  const PASSWORD_DIGITS = '23456789';
  const PASSWORD_SYMBOLS = '!@#$%^&*-_=+?';

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
    const img = toggle.querySelector('img');
    if (img) {
      img.src = revealed ? HIDE_ICON : SHOW_ICON;
      img.alt = revealed ? 'Hide' : 'Show';
    }
    toggle.title = revealed ? 'Hide' : 'Show';
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
    handleNoteReveal(target);
  });
})();
