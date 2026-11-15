// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

// mask.js provides a shared helper for masking secret values as bullets.
(function (global) {
  'use strict';

  var MASK_MAX = 16;

  function maskFor(plaintext) {
    var count = Array.from(plaintext).length;
    if (count > MASK_MAX) {
      count = MASK_MAX;
    }
    return '\u2022'.repeat(count);
  }

  global.maskFor = maskFor;
})(window);
