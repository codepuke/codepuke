// Admin-only enhancement, loaded only by the admin shell; the public site
// keeps its two-element inventory (design-system.md section 6).
//
// <admin-editor> wraps the editor page and adds the live half of the three
// board states over the already-working form: input marks the state chip
// unsaved and enables save, and the update-preview button becomes a fetch
// round trip to /admin/preview that dims the last render in place while the
// server renders. Without JS the same button posts the whole form and the
// server sends the page back with a fresh preview.
customElements.define("admin-editor", class extends HTMLElement {
  connectedCallback() {
    this.form = this.querySelector("form#editor");
    this.state = this.querySelector("[data-state]");
    this.save = this.querySelector("[data-save]");
    this.pane = this.querySelector("[data-preview-pane]");
    this.status = this.querySelector("[data-preview-status]");
    this.button = this.querySelector("[data-preview-btn]");
    if (!this.form || !this.pane) return;

    this.form.addEventListener("input", () => this.markUnsaved());
    this.button.addEventListener("click", (e) => {
      e.preventDefault();
      this.roundTrip();
    });
  }

  markUnsaved() {
    this.state.textContent = "unsaved changes";
    this.state.classList.remove("saved");
    this.state.classList.add("unsaved");
    this.save.disabled = false;
  }

  async roundTrip() {
    this.status.textContent = "preview // rendering on the server…";
    this.button.disabled = true;
    this.pane.classList.add("in-flight");
    try {
      const res = await fetch("/admin/preview", {
        method: "POST",
        body: new URLSearchParams({ body_md: this.form.elements.body_md.value }),
      });
      if (!res.ok) throw new Error(`render failed (${res.status})`);
      this.pane.innerHTML = await res.text();
      this.status.textContent = "preview // rendered just now";
    } catch (err) {
      this.status.textContent = "preview // " + err.message;
    } finally {
      this.button.disabled = false;
      this.pane.classList.remove("in-flight");
    }
  }
});
