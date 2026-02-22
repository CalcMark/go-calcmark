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

// Follow OS theme changes only when user hasn't manually chosen
window
  .matchMedia("(prefers-color-scheme: dark)")
  .addEventListener("change", (e) => {
    if (!getStoredTheme()) {
      applyTheme(e.matches ? DARK : LIGHT);
    }
  });
