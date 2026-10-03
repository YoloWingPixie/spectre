(function () {
  var root = document.documentElement;
  var key = 'spectre.theme';
  function preference() {
    try {
      var cookie = document.cookie.match(/(?:^|;\s*)spectre\.theme=(light|dark)(?:;|$)/);
      if (cookie) return cookie[1];
      var stored = localStorage.getItem(key) || localStorage.getItem('specview.theme');
      if (stored === 'light' || stored === 'dark') return stored;
    } catch (e) {}
    return null;
  }
  function apply(theme) {
    if (theme) root.setAttribute('data-theme', theme);
  }
  apply(preference());
  document.addEventListener('DOMContentLoaded', function () {
    var button = document.getElementById('theme');
    if (button) button.addEventListener('click', function () {
      var current = root.getAttribute('data-theme');
      var dark = current ? current === 'dark' : matchMedia('(prefers-color-scheme: dark)').matches;
      var theme = dark ? 'light' : 'dark';
      apply(theme);
      try { localStorage.setItem(key, theme); } catch (e) {}
      try { document.cookie = key + '=' + theme + '; Path=/; Max-Age=31536000; SameSite=Lax'; } catch (e) {}
    });
  });
  window.addEventListener('storage', function (event) {
    if (event.key === key) apply(preference());
  });
  window.addEventListener('focus', function () { apply(preference()); });
})();
