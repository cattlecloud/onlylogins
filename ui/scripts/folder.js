// folder.js implements the "example chip" behaviour on the new-document form.
//
// When a user clicks one of the .example-chip buttons, this script copies the
// chip's data-example value into the #document-label input, focuses it, and
// selects the text so the user can immediately edit it (or just hit Create).
//
// No network requests, no extra UI - purely a small UX hint that turns
// abstract placeholders into something concrete.
(function () {
  'use strict';

  document.addEventListener('click', function (event) {
    var target = event.target;
    if (!(target instanceof Element)) {
      return;
    }
    if (!target.matches('.example-chip')) {
      return;
    }

    var input = document.getElementById('document-label');
    if (!input) {
      return;
    }

    var value = target.getAttribute('data-example') || target.textContent || '';
    input.value = value;
    input.focus();
    input.select();
  });
})();
