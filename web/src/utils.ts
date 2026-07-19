import QRCode from "qrcode";

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

const SAMPLE_TEXT: Record<string, string> = {
  FirstName: "Ayşe",
  LastName: "Yılmaz",
  Email: "ayse.yilmaz@ornek.com",
  Position: "Uzman",
  Department: "Finans",
};

// 1x1 transparent GIF, used so {{.TrackPixel}} doesn't render as a broken image.
const TRANSPARENT_PIXEL = "data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBTAA7";

function replaceTag(html: string, tag: string, value: string): string {
  // Tolerates optional whitespace, e.g. both {{.FirstName}} and {{ .FirstName }}.
  const re = new RegExp(`\\{\\{\\s*\\.${tag}\\s*\\}\\}`, "g");
  return html.replace(re, value);
}

// renderPreviewHtml substitutes PhishForge's merge-tags with realistic sample
// data for PREVIEW ONLY, so the operator sees a finished-looking page instead of
// raw {{.Tag}} placeholders. The real send/click-time HTML is rendered
// server-side (Go's html/template) with the actual per-target values — this is
// purely a client-side visual aid and has no effect on what gets sent or served.
export async function renderPreviewHtml(raw: string): Promise<string> {
  let html = raw;
  for (const [tag, value] of Object.entries(SAMPLE_TEXT)) {
    html = replaceTag(html, tag, value);
  }
  html = replaceTag(html, "TrackPixel", TRANSPARENT_PIXEL);
  html = replaceTag(html, "TrackURL", "#");
  html = replaceTag(html, "ReportURL", "#");
  html = replaceTag(html, "SubmitURL", "#");
  html = replaceTag(html, "AttachmentURL", "#");

  if (/\{\{\s*\.QRCodeURL\s*\}\}/.test(html)) {
    try {
      const dataUrl = await QRCode.toDataURL("https://phishforge.example/l/preview", { width: 256, margin: 1 });
      html = replaceTag(html, "QRCodeURL", dataUrl);
    } catch {
      html = replaceTag(html, "QRCodeURL", "");
    }
  }
  return html;
}
