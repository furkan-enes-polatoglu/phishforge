// Opens an HTML string in a new browser tab as a full standalone page, using a
// blob: URL (no backend round-trip needed). Used by every "live preview" panel
// so operators can see exactly how an email/landing/training page will render
// full-screen.
export function openHtmlInNewTab(html: string) {
  const blob = new Blob([html], { type: "text/html" });
  const url = URL.createObjectURL(blob);
  window.open(url, "_blank", "noopener,noreferrer");
  // Give the new tab time to load the blob before revoking it.
  setTimeout(() => URL.revokeObjectURL(url), 60_000);
}
