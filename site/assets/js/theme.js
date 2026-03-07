// Dark/light theme toggle with localStorage persistence.
// The inline script in head.html handles initial render to prevent FOUC.

const THEME_KEY = "theme";
const DARK = "dark";
const LIGHT = "light";

function getStoredTheme() {
  return localStorage.getItem(THEME_KEY);
}

function applyTheme(theme) {
  document.documentElement.setAttribute("data-theme", theme);
  localStorage.setItem(THEME_KEY, theme);
}

function toggleTheme() {
  const current = document.documentElement.getAttribute("data-theme");
  applyTheme(current === DARK ? LIGHT : DARK);
}

// Copy-to-clipboard for repo-file shortcode buttons.
// Each button stores the full `cm remote --http <url>` command in data-cmd.
document.addEventListener("click", (e) => {
  const btn = e.target.closest(".repo-file-copy");
  if (!btn) return;
  const cmd = btn.getAttribute("data-cmd");
  if (!cmd) return;
  navigator.clipboard.writeText(cmd).then(() => {
    btn.classList.add("copied");
    const svg = btn.innerHTML;
    btn.innerHTML =
      '<svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3.5 9l3 3 6-7"/></svg>';
    setTimeout(() => {
      btn.classList.remove("copied");
      btn.innerHTML = svg;
    }, 1500);
  });
});

// Follow OS theme changes only when user hasn't manually chosen
window
  .matchMedia("(prefers-color-scheme: dark)")
  .addEventListener("change", (e) => {
    if (!getStoredTheme()) {
      applyTheme(e.matches ? DARK : LIGHT);
    }
  });
