// Runs before first paint so the saved theme never flashes. Kept as a separate
// file rather than an inline script so the CSP can forbid inline script.
(function () {
  try {
    var saved = localStorage.getItem('pl-theme')
    var theme = saved === 'light' || saved === 'dark' ? saved : 'dark'
    document.documentElement.setAttribute('data-theme', theme)
    var meta = document.getElementById('meta-theme-color')
    if (meta) meta.setAttribute('content', theme === 'light' ? '#f9f7f3' : '#090a0d')
  } catch (e) {}
})()
