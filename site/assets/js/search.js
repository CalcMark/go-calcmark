// Site search powered by Fuse.js.
// Opens a modal overlay; fetches /index.json on first use.

(function () {
  const modal = document.getElementById("search-modal");
  const input = document.getElementById("search-input");
  const results = document.getElementById("search-results");
  if (!modal || !input || !results) return;

  let fuse = null;
  let indexPromise = null;

  function loadIndex() {
    if (!indexPromise) {
      indexPromise = fetch("/index.json")
        .then((r) => r.json())
        .then((data) => {
          fuse = new Fuse(data, {
            keys: [
              { name: "title", weight: 3 },
              { name: "description", weight: 2 },
              { name: "contents", weight: 1 },
            ],
            threshold: 0.3,
            ignoreLocation: true,
            minMatchCharLength: 2,
          });
        });
    }
    return indexPromise;
  }

  function openSearch() {
    modal.setAttribute("aria-hidden", "false");
    modal.classList.add("search-modal--open");
    document.body.style.overflow = "hidden";
    input.value = "";
    results.innerHTML = "";
    input.focus();
  }

  function closeSearch() {
    modal.setAttribute("aria-hidden", "true");
    modal.classList.remove("search-modal--open");
    document.body.style.overflow = "";
  }

  function escapeHtml(str) {
    var d = document.createElement("div");
    d.textContent = str;
    return d.innerHTML;
  }

  function renderResults(hits) {
    if (!hits.length) {
      results.innerHTML =
        '<div class="search-empty">No results found.</div>';
      return;
    }
    var html = hits.slice(0, 12).map(function (hit) {
      var item = hit.item;
      var snippet = item.contents
        ? item.contents.substring(0, 120) + "\u2026"
        : item.description || "";
      return (
        '<a class="search-result" href="' +
        escapeHtml(item.permalink) +
        '">' +
        '<span class="search-result-title">' +
        escapeHtml(item.title) +
        "</span>" +
        '<span class="search-result-snippet">' +
        escapeHtml(snippet) +
        "</span>" +
        "</a>"
      );
    });
    results.innerHTML = html.join("");
  }

  // Debounced search
  var timer = null;
  input.addEventListener("input", function () {
    clearTimeout(timer);
    var query = input.value.trim();
    if (query.length < 2) {
      results.innerHTML = "";
      return;
    }
    timer = setTimeout(function () {
      loadIndex().then(function () {
        var hits = fuse.search(query);
        renderResults(hits);
      });
    }, 150);
  });

  // Close on backdrop click
  modal.addEventListener("click", function (e) {
    if (e.target === modal) closeSearch();
  });

  // Close on Escape; open on /
  document.addEventListener("keydown", function (e) {
    if (e.key === "Escape" && modal.classList.contains("search-modal--open")) {
      closeSearch();
      return;
    }
    // Open on / when not typing in an input/textarea
    if (
      e.key === "/" &&
      !modal.classList.contains("search-modal--open") &&
      !["INPUT", "TEXTAREA", "SELECT"].includes(document.activeElement.tagName) &&
      !document.activeElement.isContentEditable
    ) {
      e.preventDefault();
      openSearch();
    }
  });

  // Keyboard navigation within results
  input.addEventListener("keydown", function (e) {
    if (e.key === "ArrowDown") {
      e.preventDefault();
      var first = results.querySelector(".search-result");
      if (first) first.focus();
    }
  });

  results.addEventListener("keydown", function (e) {
    var focused = document.activeElement;
    if (!focused || !focused.classList.contains("search-result")) return;
    if (e.key === "ArrowDown") {
      e.preventDefault();
      var next = focused.nextElementSibling;
      if (next) next.focus();
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      var prev = focused.previousElementSibling;
      if (prev) prev.focus();
      else input.focus();
    }
  });

  // Wire up the nav search button
  var btn = document.getElementById("search-toggle");
  if (btn) {
    btn.addEventListener("click", function (e) {
      e.preventDefault();
      openSearch();
    });
  }
})();
