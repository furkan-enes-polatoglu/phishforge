import { createContext, useContext, useMemo, useState } from "react";

export type Lang = "tr" | "en";

// Dictionary. Turkish is the primary language; English is kept as a base so more
// languages can be added later. Missing keys fall back to the key itself.
const dict: Record<string, { tr: string; en: string }> = {
  // generic
  save: { tr: "Kaydet", en: "Save" },
  create: { tr: "Oluştur", en: "Create" },
  edit: { tr: "Düzenle", en: "Edit" },
  update: { tr: "Güncelle", en: "Update" },
  delete: { tr: "Sil", en: "Delete" },
  duplicate: { tr: "Kopyala", en: "Duplicate" },
  cancel: { tr: "Vazgeç", en: "Cancel" },
  name: { tr: "Ad", en: "Name" },
  status: { tr: "Durum", en: "Status" },
  actions: { tr: "İşlemler", en: "Actions" },
  loading: { tr: "Yükleniyor…", en: "Loading…" },
  none_yet: { tr: "Henüz yok.", en: "Nothing yet." },
  existing: { tr: "Mevcut", en: "Existing" },
  saved: { tr: "Kaydedildi.", en: "Saved." },
  confirm_delete: { tr: "silinsin mi? Bu işlem geri alınamaz.", en: "delete? This cannot be undone." },
  logout: { tr: "Çıkış", en: "Log out" },

  // login
  login_subtitle: { tr: "Gelişmiş Oltalama Simülasyonu ve Güvenlik Farkındalık Platformu", en: "Advanced Phishing Simulation & Security Awareness Platform" },
  email: { tr: "E-posta", en: "Email" },
  password: { tr: "Parola", en: "Password" },
  sign_in: { tr: "Giriş yap", en: "Sign in" },
  signing_in: { tr: "Giriş yapılıyor…", en: "Signing in…" },
  language: { tr: "Dil", en: "Language" },

  // nav
  nav_dashboard: { tr: "Panel", en: "Dashboard" },
  nav_engagements: { tr: "Angajmanlar", en: "Engagements" },
  nav_assets: { tr: "Varlıklar", en: "Assets" },
  nav_training: { tr: "Eğitim", en: "Training" },
  nav_deliverability: { tr: "Teslimat", en: "Deliverability" },
  nav_settings: { tr: "Ayarlar", en: "Settings" },
  nav_audit: { tr: "Denetim", en: "Audit" },

  // dashboard
  dashboard: { tr: "Panel", en: "Dashboard" },
  stat_engagements: { tr: "Angajmanlar", en: "Engagements" },
  stat_active: { tr: "Aktif", en: "Active" },
  stat_targets: { tr: "Ulaşılan hedef", en: "Targets contacted" },
  stat_role: { tr: "Rolün", en: "Your role" },
  org_funnel: { tr: "Organizasyon hunisi (tüm kampanyalar)", en: "Organization funnel (all campaigns)" },
  authorized_only: { tr: "Yalnızca yetkili kullanım", en: "Authorized use only" },
  authorized_only_body: {
    tr: "Her kampanya; müşteriyi, yazılı yetki referansını ve tarih penceresini kaydeden bir angajman içinde çalışır. Kapsam dışı hedefler reddedilir ve tüm eylemler değiştirilemez denetim günlüğüne yazılır.",
    en: "Every campaign runs inside an engagement that records the client, a written authorization reference, and a date window. Out-of-scope targets are rejected, and all actions are written to an append-only audit log.",
  },

  // engagements
  engagements: { tr: "Angajmanlar", en: "Engagements" },
  new_engagement: { tr: "Yeni angajman (yetki kaydı)", en: "New engagement (authorization record)" },
  client_name: { tr: "Müşteri adı", en: "Client name" },
  authz_ref: { tr: "Yetki referansı", en: "Authorization reference" },
  starts: { tr: "Başlangıç", en: "Starts" },
  ends: { tr: "Bitiş", en: "Ends" },
  window: { tr: "Pencere", en: "Window" },
  open_: { tr: "Aç", en: "Open" },
  activate: { tr: "Aktifleştir", en: "Activate" },
  close_: { tr: "Kapat", en: "Close" },
  authorization: { tr: "Yetki", en: "Authorization" },

  // scope + targets
  scope_allowlist: { tr: "Kapsam (izin listesi)", en: "Scope (allowlist)" },
  scope_help: { tr: "Yalnızca bir kurala uyan hedeflere ulaşılabilir. Aktifleştirme için en az bir kural gerekir.", en: "Only targets matching a rule can be contacted. Activation requires at least one rule." },
  add: { tr: "Ekle", en: "Add" },
  no_rules: { tr: "Kural yok — angajman aktifleştirilemez.", en: "No rules — engagement cannot be activated." },
  targets: { tr: "Hedefler", en: "Targets" },
  import_scope_checked: { tr: "İçe aktar (kapsam kontrollü)", en: "Import (scope-checked)" },
  import_from_file: { tr: "Excel/CSV ile içe aktar", en: "Import from Excel/CSV" },
  import_file_help: {
    tr: "İlk satır başlık olmalı. Zorunlu: E-posta. Opsiyonel: Ad Soyad (veya Ad + Soyad ayrı sütunlarda), Departman, Pozisyon, Saat Dilimi, VIP. .xlsx veya .csv kabul edilir.",
    en: "First row must be headers. Required: Email. Optional: Full Name (or First + Last separately), Department, Position, Timezone, VIP. .xlsx or .csv accepted.",
  },
  download_template: { tr: "Şablonu indir (CSV)", en: "Download template (CSV)" },
  upload_and_import: { tr: "Yükle ve içe aktar", en: "Upload and import" },
  department: { tr: "Departman", en: "Department" },
  vip: { tr: "VIP", en: "VIP" },
  parse_errors: { tr: "satırda ayrıştırma hatası", en: "row parse errors" },
  targets_added: { tr: "hedef eklendi", en: "targets added" },
  rejected_scope: { tr: "kapsam dışı reddedildi", en: "rejected (out of scope)" },

  // campaign
  new_campaign: { tr: "Yeni kampanya", en: "New campaign" },
  email_template: { tr: "E-posta şablonu", en: "Email template" },
  landing_page: { tr: "Açılış sayfası", en: "Landing page" },
  sending_profile: { tr: "Gönderim profili", en: "Sending profile" },
  rate_per_min: { tr: "Dakikada gönderim", en: "Rate / minute" },
  schedule_optional: { tr: "Zamanlama (opsiyonel)", en: "Schedule (optional)" },
  window_start: { tr: "Pencere başlangıcı (saat)", en: "Send window start (h)" },
  window_end: { tr: "Pencere bitişi (saat)", en: "Send window end (h)" },
  warmup: { tr: "Döngü başına warm-up (0=∞)", en: "Warm-up per cycle (0=∞)" },
  jitter: { tr: "Jitter (saniye)", en: "Jitter (seconds)" },
  business_days: { tr: "Sadece iş günleri", en: "Business days only" },
  rewrite_links: { tr: "Bağlantıları takip için yeniden yaz", en: "Auto-rewrite links for tracking" },
  launch: { tr: "Başlat", en: "Launch" },
  stop: { tr: "Durdur", en: "Stop" },
  report: { tr: "Rapor", en: "Report" },
  select: { tr: "— seçin —", en: "— select —" },

  // risk
  risk_scores: { tr: "Kullanıcı risk skorları", en: "User risk scores" },
  opens: { tr: "Açılma", en: "Opens" },
  clicks: { tr: "Tıklama", en: "Clicks" },
  submits: { tr: "Gönderim", en: "Submits" },
  reports: { tr: "Bildirim", en: "Reports" },
  score: { tr: "Skor", en: "Score" },
  level: { tr: "Seviye", en: "Level" },

  // assets
  assets: { tr: "Varlıklar", en: "Assets" },
  email_templates: { tr: "E-posta şablonları", en: "Email templates" },
  landing_pages: { tr: "Açılış sayfaları", en: "Landing pages" },
  sending_profiles: { tr: "Gönderim profilleri", en: "Sending profiles" },
  live_preview: { tr: "Canlı önizleme", en: "Live preview" },
  open_full_tab: { tr: "Tam ekran, yeni sekmede aç", en: "Open full-screen in new tab" },
  preview_sample_note: { tr: "Örnek verilerle gösteriliyor (gerçek gönderimde asıl bilgiler kullanılır)", en: "Shown with sample data (real sends use actual values)" },
  subject: { tr: "Konu", en: "Subject" },
  merge_tags: { tr: "Birleştirme etiketleri", en: "Merge-tags" },
  qr_insert_hint: { tr: "QR kod (quishing) eklemek için: ", en: "To insert a QR code (quishing): " },
  attachment_insert_hint: { tr: "Sahte ek eklemek için: ", en: "To insert a simulated attachment: " },
  clone_from_url: { tr: "URL'den sayfa klonla", en: "Clone a page from URL" },
  import: { tr: "İçe aktar", en: "Import" },
  fetching: { tr: "Getiriliyor…", en: "Fetching…" },
  clone_help: { tr: "Sayfa HTML'ini editöre getirir (stiller/görseller için <base> etiketi eklenir). Bazı siteler otomatik erişimi engeller — o durumda HTML'i elle yapıştırın.", en: "Fetches the page HTML into the editor (a <base> tag is added). Some sites block automated fetches — if so, paste HTML manually." },
  redirect_after_submit: { tr: "Gönderim sonrası farkındalık yönlendirme URL'si (opsiyonel — yoksa otomatik eğitim)", en: "Awareness redirect URL after submit (optional — else auto-training)" },
  capture_settings: { tr: "Yakalama ayarları", en: "Capture settings" },
  capture_field_names: { tr: "Doldurulan alan adlarını yakala", en: "Capture which field names were filled" },
  capture_values: { tr: "Gönderilen form değerlerini yakala", en: "Capture submitted form values" },
  capture_passwords: { tr: "Parola alanlarını da yakala", en: "Also capture password fields" },
  capture_pw_warn: { tr: "⚠ Parola yakalamak hassas veri saklar. Yalnızca açık müşteri onayıyla etkinleştirin; angajman kurallarına göre saklayıp imha edin.", en: "⚠ Capturing passwords stores sensitive data. Enable only with explicit client authorization." },
  captures_data: { tr: "veri yakalar", en: "captures data" },
  captures_pw: { tr: "parola yakalar", en: "captures pw" },
  from_address: { tr: "Gönderen adresi", en: "From address" },
  from_name: { tr: "Gönderen adı", en: "From name" },
  smtp_host: { tr: "SMTP sunucusu", en: "SMTP host" },
  port: { tr: "Port", en: "Port" },
  username: { tr: "Kullanıcı adı", en: "Username" },
  leave_blank_keep: { tr: "(boş bırakırsanız mevcut korunur)", en: "(leave blank to keep current)" },
  x_mailer: { tr: "Posta istemcisi başlığı (X-Mailer)", en: "Mail client header (X-Mailer)" },
  x_mailer_help: { tr: "Gerçek bir e-posta istemcisi gibi görünmesi için (opsiyonel), örn. \"Microsoft Outlook 16.0\"", en: "Optional, makes the message look like it came from a real client, e.g. \"Microsoft Outlook 16.0\"" },
  dkim_signing: { tr: "DKIM imzalama (teslimat)", en: "DKIM signing (deliverability)" },
  dkim_help: { tr: "DKIM, yetkili göndericiler için teslimatı artıran standart bir kimlik doğrulamadır. Anahtar üretin ve DNS TXT kaydını yayınlayın.", en: "DKIM is a standard authentication that improves deliverability for authorized senders. Generate a key and publish the DNS TXT record." },
  dkim_domain: { tr: "DKIM alan adı", en: "DKIM domain" },
  dkim_selector: { tr: "DKIM seçici", en: "DKIM selector" },
  generate_dkim: { tr: "DKIM anahtarı üret", en: "Generate DKIM key" },
  dkim_dns_record: { tr: "Bu TXT kaydını DNS'e ekleyin:", en: "Add this TXT record to DNS:" },
  sign_with_dkim: { tr: "Gönderimleri DKIM ile imzala", en: "Sign messages with DKIM" },
  provider: { tr: "Gönderim yöntemi", en: "Sending method" },
  provider_smtp: { tr: "SMTP (herhangi bir sunucu, Mailgun SMTP dahil)", en: "SMTP (any server, including Mailgun's SMTP relay)" },
  provider_mailgun_api: { tr: "Mailgun HTTP API (önerilen)", en: "Mailgun HTTP API (recommended)" },
  mailgun_api_key: { tr: "Mailgun API anahtarı", en: "Mailgun API key" },
  mailgun_domain: { tr: "Mailgun alan adı", en: "Mailgun domain" },
  mailgun_help: {
    tr: "Mailgun'un resmi olarak önerdiği entegrasyon yöntemi: ham SMTP'ye göre daha güvenilir, gerçek teslimat geri bildirimi (delivered/bounced/complained) alır. Kendi tıklama/açılma takibimizle çakışmaması için Mailgun'un kendi tracking'i otomatik kapatılır.",
    en: "Mailgun's officially recommended integration: more reliable than raw SMTP, gets real delivery feedback (delivered/bounced/complained). Mailgun's own tracking is disabled automatically so it doesn't conflict with ours.",
  },
  spoofed_from: { tr: "Görünen gönderen (pretext)", en: "Display sender (pretext)" },
  spoofed_from_help: {
    tr: "Gerçek bir e-posta adresini birebir görünen \"Kimden\" alanında gösterebilirsiniz. Teknik gönderim (SPF/DKIM) her zaman gönderim profilinin kendi alan adında kalır. Görünen adres farklı bir alan adındaysa DMARC hizalaması hedefte BAŞARISIZ olur — güvenilir teslimat için Teslimat sayfasındaki \"Hedef Mail Ağ Geçidi Tespiti\" ile bu gönderim altyapısını beyaz listeye aldırmanız gerekir.",
    en: "Show an exact real address in the visible \"From\" field. Technical sending (SPF/DKIM) always stays the sending profile's own domain. If the display address is on a different domain, DMARC alignment WILL fail at the target — for reliable delivery, allowlist this sending infrastructure via \"Target Mail Gateway Detection\" on the Deliverability page.",
  },
  spoofed_from_name: { tr: "Görünen gönderen adı", en: "Display sender name" },
  spoofed_from_address: { tr: "Görünen gönderen adresi", en: "Display sender address" },
  reply_to: { tr: "Yanıt adresi (Reply-To)", en: "Reply-To address" },

  // training
  training_title: { tr: "Güvenlik farkındalık eğitimi", en: "Security awareness training" },
  training_help: { tr: "Tıklayan veya gönderen hedefler otomatik olarak ilk eğitim modülüne atanır ve yönlendirilir. Modülü görüntülemek tamamlandı olarak işaretler.", en: "Targets who click or submit are auto-assigned the first module and redirected. Viewing marks it completed." },
  new_module: { tr: "Yeni eğitim modülü", en: "New training module" },
  modules: { tr: "Modüller", en: "Modules" },
  preview: { tr: "Önizleme", en: "Preview" },
  assignments: { tr: "Atamalar ve tamamlanma", en: "Assignments & completion" },
  target: { tr: "Hedef", en: "Target" },
  module: { tr: "Modül", en: "Module" },
  assigned: { tr: "Atandı", en: "Assigned" },
  completed: { tr: "Tamamlandı", en: "Completed" },

  // deliverability
  deliverability: { tr: "Teslimat", en: "Deliverability" },
  deliverability_help: { tr: "Meşru gönderim öncesi e-posta sağlık kontrolleri — SPF/DKIM/DMARC, kara listeler ve işaretlemeyi doğrulayın ki yetkili test postası gelen kutusuna ulaşsın. Bu bir spam-filtresi atlatma aracı DEĞİLDİR; müşteri mail gateway'inde allowlist ile koordine olun.", en: "Legitimate pre-send email health checks. Not a spam-filter evasion tool; coordinate an allowlist with the client's mail gateway." },
  run_check: { tr: "Kontrolü çalıştır", en: "Run check" },
  checking: { tr: "Kontrol ediliyor…", en: "Checking…" },
  authentication: { tr: "Kimlik doğrulama", en: "Authentication" },
  advice: { tr: "Öneriler", en: "Advice" },
  blocklists: { tr: "Kara listeler", en: "Blocklists" },
  delivery_score: { tr: "Teslimat Güven Skoru", en: "Delivery Confidence Score" },
  ptr_check: { tr: "PTR / Ters DNS (FCrDNS)", en: "PTR / Reverse DNS (FCrDNS)" },
  mta_sts: { tr: "MTA-STS", en: "MTA-STS" },
  tls_rpt: { tr: "TLS-RPT", en: "TLS-RPT" },
  dmarc_policy: { tr: "DMARC politikası", en: "DMARC policy" },
  content_analysis: { tr: "İçerik / spam-tetikleyici analizi", en: "Content / spam-trigger analysis" },
  subject_line: { tr: "Konu satırı", en: "Subject line" },
  trigger_words: { tr: "Tetikleyici kelimeler", en: "Trigger words" },
  shorteners: { tr: "Link kısaltıcılar", en: "Link shorteners" },
  all_caps: { tr: "TÜMÜ BÜYÜK HARF kelime", en: "ALL-CAPS words" },
  image_only_warning: { tr: "⚠ E-posta neredeyse tamamen görsellerden oluşuyor, çok az metin var", en: "⚠ Email is almost entirely images, very little text" },
  gateway_detect: { tr: "🎯 Hedef Mail Ağ Geçidi Tespiti", en: "🎯 Target Mail Gateway Detection" },
  gateway_detect_help: {
    tr: "Hedef şirketin e-posta güvenlik ağ geçidini (Proofpoint, Mimecast, Microsoft 365, Google Workspace, Barracuda, Cisco...) MX kaydından tespit eder ve o ürüne özel, adım adım beyaz listeye alma talimatlarıyla birlikte müşterinin IT ekibine gönderebileceğiniz hazır bir talep e-postası üretir. Yetkili bir testte inbox garantisi için en güçlü tek adım budur.",
    en: "Detects the target company's email security gateway from its MX records and generates step-by-step allowlisting instructions plus a ready-to-send request for the client's IT team. The single strongest lever for guaranteed inbox delivery in an authorized test.",
  },
  target_domain: { tr: "Hedef şirket alan adı", en: "Target company domain" },
  detect_gateway: { tr: "Ağ geçidini tespit et", en: "Detect gateway" },
  detecting: { tr: "Tespit ediliyor…", en: "Detecting…" },
  detected_gateway: { tr: "Tespit edilen ağ geçidi", en: "Detected gateway" },
  gateway_unknown: { tr: "Bilinen bir imzayla eşleşmedi (özel/tanınmayan ağ geçidi)", en: "No known signature matched (custom/unrecognized gateway)" },
  mx_records: { tr: "MX kayıtları", en: "MX records" },
  allowlist_steps: { tr: "Beyaz listeye alma adımları", en: "Allowlisting steps" },
  cover_email: { tr: "Müşteriye gönderilecek hazır talep e-postası", en: "Ready-to-send request for the client" },
  copy: { tr: "Kopyala", en: "Copy" },
  copied: { tr: "Kopyalandı ✓", en: "Copied ✓" },
  seed_test: { tr: "Seed-liste gelen kutusu testi", en: "Seed-list inbox placement test" },
  seed_test_help: {
    tr: "Gerçek bir e-posta hesabına (seed) gönderdiğiniz test postasının gelen kutusuna mı yoksa spam'e mi düştüğünü IMAP ile otomatik kontrol eder. Önce konu satırında benzersiz bir işaretle (marker) bir kampanya gönderin, sonra o kutunun IMAP bilgileriyle burada kontrol edin.",
    en: "Automatically checks via IMAP whether a test send landed in the inbox or spam of a real seed mailbox. Send a campaign with a unique subject marker first, then check that mailbox's IMAP here.",
  },
  imap_host: { tr: "IMAP sunucusu", en: "IMAP host" },
  subject_marker: { tr: "Konu işareti (benzersiz metin)", en: "Subject marker (unique text)" },
  run_seed_check: { tr: "Kontrol et", en: "Check" },
  seed_found_inbox: { tr: "Gelen kutusunda bulundu", en: "Found in inbox" },
  seed_found_spam: { tr: "Spam/Junk klasöründe bulundu", en: "Found in spam/junk" },
  seed_not_found: { tr: "Henüz bulunamadı (gecikmiş olabilir ya da başka bir klasörde)", en: "Not found yet (may be delayed or in another folder)" },

  // settings
  settings: { tr: "Ayarlar", en: "Settings" },
  notifications: { tr: "Bildirimler (webhook & Slack/Teams)", en: "Notifications (webhooks & Slack/Teams)" },
  notif_help: { tr: "Hedef eylemlerinde gerçek zamanlı uyarılar. Slack/Teams webhook URL'leri otomatik biçimlendirilir; diğerleri imzalı JSON alır.", en: "Real-time alerts on target actions. Slack/Teams URLs auto-formatted; others get signed JSON." },
  webhook_url: { tr: "Webhook URL", en: "Webhook URL" },
  signing_secret: { tr: "İmzalama sırrı (opsiyonel)", en: "Signing secret (optional)" },
  add_webhook: { tr: "Webhook ekle", en: "Add webhook" },
  none_all_events: { tr: "(hiçbiri seçili değilse tüm olaylar)", en: "(none selected = all events)" },
  events: { tr: "Olaylar", en: "Events" },
  api_keys: { tr: "API anahtarları (otomasyon)", en: "API keys (automation)" },
  api_help: { tr: "X-API-Key başlığında kullanın. Tam anahtar yalnızca oluşturmada bir kez gösterilir.", en: "Use in the X-API-Key header. The full key is shown once at creation." },
  api_created_once: { tr: "Yeni anahtar (şimdi kopyalayın, bir kez gösterilir):", en: "New key (copy now, shown once):" },
  role: { tr: "Rol", en: "Role" },
  create_key: { tr: "Anahtar oluştur", en: "Create key" },
  revoke: { tr: "İptal et", en: "Revoke" },
  revoked: { tr: "iptal edildi", en: "revoked" },
  prefix: { tr: "Önek", en: "Prefix" },
  last_used: { tr: "Son kullanım", en: "Last used" },
  never: { tr: "hiç", en: "never" },

  // audit
  audit_log: { tr: "Denetim günlüğü", en: "Audit log" },
  audit_help: { tr: "Organizasyonunuzdaki ayrıcalıklı eylemlerin değiştirilemez kaydı.", en: "Append-only record of privileged actions within your organization." },
  when: { tr: "Zaman", en: "When" },
  action: { tr: "Eylem", en: "Action" },
  entity: { tr: "Varlık", en: "Entity" },
  detail: { tr: "Ayrıntı", en: "Detail" },

  // report
  funnel: { tr: "Huni", en: "Funnel" },
  ab_variants: { tr: "A/B varyantları", en: "A/B variants" },
  variant: { tr: "Varyant", en: "Variant" },
  weight: { tr: "Ağırlık", en: "Weight" },
  add_variant: { tr: "Varyant ekle", en: "Add variant" },
  timeline: { tr: "Zaman çizelgesi", en: "Timeline" },
  event: { tr: "Olay", en: "Event" },
  captured_data: { tr: "Yakalanan veri", en: "Captured data" },
  event_sent: { tr: "gönderildi", en: "sent" },
  event_open: { tr: "açıldı", en: "opened" },
  event_click: { tr: "tıklandı", en: "clicked" },
  event_submit: { tr: "gönderim", en: "submitted" },
  event_report: { tr: "bildirildi", en: "reported" },
  event_scan: { tr: "QR tarandı", en: "QR scanned" },
  event_attachment_open: { tr: "ek açıldı", en: "attachment opened" },
  event_delivered: { tr: "teslim edildi", en: "delivered" },
  event_bounced: { tr: "geri döndü (bounce)", en: "bounced" },
  event_complained: { tr: "spam olarak işaretlendi", en: "complained" },
  delivery_health: { tr: "Teslimat sağlığı (ESP geri bildirimi)", en: "Delivery health (ESP feedback)" },
  delivery_health_help: {
    tr: "Mailgun webhook'ları üzerinden gelen gerçek teslimat sonuçları. Yüksek bounce/complaint oranında kampanya otomatik olarak durdurulur (domain itibarını korumak için).",
    en: "Real delivery outcomes reported via Mailgun webhooks. Campaigns are auto-paused on a high bounce/complaint rate to protect domain reputation.",
  },
};

interface I18nCtx {
  lang: Lang;
  setLang: (l: Lang) => void;
  t: (key: string) => string;
}

const Ctx = createContext<I18nCtx>({ lang: "tr", setLang: () => {}, t: (k) => k });

export function I18nProvider({ children }: { children: React.ReactNode }) {
  const [lang, setLangState] = useState<Lang>((localStorage.getItem("pf_lang") as Lang) || "tr");
  const value = useMemo<I18nCtx>(
    () => ({
      lang,
      setLang: (l) => { localStorage.setItem("pf_lang", l); setLangState(l); },
      t: (key) => dict[key]?.[lang] ?? key,
    }),
    [lang]
  );
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useI18n() {
  return useContext(Ctx);
}
