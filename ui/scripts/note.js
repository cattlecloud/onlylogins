// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

// note.js implements the click-to-reveal behavior for notes:
//
// This is intentionally one-way (no Hide, no Copy).
(function () {
  'use strict';

  document.addEventListener('click', function (event) {
    var target = event.target;
    if (!(target instanceof Element)) {
      return;
    }

    if (!target.matches('.note-reveal-btn')) {
      return;
    }

    var area = target.closest('.note-area');
    if (!area) {
      return;
    }

    var panel = area.querySelector('.note-panel');
    if (!panel) {
      return;
    }

    var content = panel.querySelector('.note-content');
    if (!content) {
      return;
    }

    var plaintext = panel.getAttribute('data-note') || '';
    content.textContent = plaintext;
    content.hidden = false;
    target.hidden = true;

    var actions = area.querySelector('.note-actions');
    if (actions) {
      actions.hidden = false;
    }
  });
})();
