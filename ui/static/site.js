// The complete JS inventory (design-system.md section 6): two custom
// elements that enhance markup already working without them.

// 4a. <code-tabs>: builds the tab row, toggles sections, and keeps every
// instance on the page in sync. Page context beats stored preference: a
// data-default attribute (baked in by the renderer for project docs) wins
// the initial state, and clicks on such pages are never persisted. When a
// switch changes block heights, the clicked block is scroll-anchored so the
// reader never loses their place.
customElements.define("code-tabs", class extends HTMLElement {
  connectedCallback() {
    if (this.querySelector(":scope > .ct-tabs")) return;
    const sections = [...this.querySelectorAll(":scope > section")];
    if (!sections.length) return;

    const nav = document.createElement("nav");
    nav.className = "ct-tabs";
    nav.setAttribute("role", "tablist");
    for (const s of sections) {
      const tab = document.createElement("button");
      tab.type = "button";
      tab.setAttribute("role", "tab");
      tab.dataset.lang = s.dataset.lang;
      tab.textContent = s.querySelector(".ct-label")?.textContent ?? s.dataset.lang;
      tab.addEventListener("click", () => {
        if (!this.dataset.default) {
          localStorage.setItem("codepuke:lang", s.dataset.lang);
        }
        const top = this.getBoundingClientRect().top;
        document.dispatchEvent(new CustomEvent("codepuke:lang", { detail: s.dataset.lang }));
        window.scrollBy(0, this.getBoundingClientRect().top - top);
      });
      nav.append(tab);
    }
    this.prepend(nav);

    const show = (lang) => {
      const active = sections.some((s) => s.dataset.lang === lang)
        ? lang
        : sections[0].dataset.lang;
      this.dataset.active = active;
      for (const s of sections) s.hidden = s.dataset.lang !== active;
      for (const tab of nav.children) {
        tab.setAttribute("aria-selected", String(tab.dataset.lang === active));
      }
    };
    document.addEventListener("codepuke:lang", (e) => show(e.detail));
    show(this.dataset.default || localStorage.getItem("codepuke:lang"));
  }
});

// 4c. <scroll-box>: toggles data-overflow when a line actually overflows,
// so CSS can draw the chip and fade. Without JS, native scroll remains.
customElements.define("scroll-box", class extends HTMLElement {
  connectedCallback() {
    const child = this.firstElementChild;
    if (!child) return;
    const measure = () => {
      if (child.scrollWidth > child.clientWidth) {
        this.setAttribute("data-overflow", "");
      } else {
        this.removeAttribute("data-overflow");
      }
    };
    this.resizeObserver = new ResizeObserver(measure);
    this.resizeObserver.observe(child);
    measure();
  }
  disconnectedCallback() {
    this.resizeObserver?.disconnect();
  }
});
