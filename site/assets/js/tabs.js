// Tab persistence: save and restore tab preferences via localStorage.
// Syncs all tab groups with the same name on the page.
(function () {
  // Restore saved preferences on load.
  document.querySelectorAll('.tabs').forEach(function (el) {
    var group = el.dataset.tabGroup;
    var idx = localStorage.getItem('tab-' + group);
    if (idx !== null) {
      var radio = el.querySelectorAll('.tabs-radio')[parseInt(idx, 10)];
      if (radio) radio.checked = true;
    }
  });

  // On change, persist and sync same-group tabs.
  document.addEventListener('change', function (e) {
    if (!e.target.classList.contains('tabs-radio')) return;
    var el = e.target.closest('.tabs');
    var group = el.dataset.tabGroup;
    var radios = el.querySelectorAll('.tabs-radio');
    var idx = Array.from(radios).indexOf(e.target);
    localStorage.setItem('tab-' + group, idx);
    document.querySelectorAll('.tabs[data-tab-group="' + group + '"]').forEach(function (other) {
      if (other !== el) {
        var r = other.querySelectorAll('.tabs-radio')[idx];
        if (r) r.checked = true;
      }
    });
  });
})();
