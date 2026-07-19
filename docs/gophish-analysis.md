# GoPhish Analizi ve PhishForge Kararları

> Kapsam notu: PhishForge, **yalnızca yazılı müşteri onayı** bulunan phishing simülasyonu ve
> güvenlik farkındalık (security awareness) angajmanları için tasarlanır. Belgedeki her karar,
> "gerçek saldırganı taklit ederek kullanıcı davranışını ölçme ve eğitme" hedefine hizmet eder;
> kimlik bilgisi hasadı değil, **davranış olayı ölçümü** esastır.

## 1. GoPhish nedir, mimarisi

GoPhish, açık kaynaklı bir phishing simülasyon çerçevesidir. Mimarisi kısaca:

- **Tekil Go monolit**: admin API + phishing (landing) sunucusu tek binary.
- **Varsayılan SQLite** (isteğe bağlı MySQL). Gömülü, tek dosya.
- **Gönderim**: kampanya başlatınca senkron/az-paralel bir mailer goroutine'i SMTP üzerinden yollar.
- **Şablonlar**: HTML e-posta + landing page, `{{.FirstName}}` gibi Go `text/template` merge-tag'leri.
- **İzleme**: 1x1 tracking pixel (açılma), benzersiz link (tıklama), form POST yakalama, opsiyonel
  parola yakalama, `RId` (result id) parametresiyle hedef eşleştirme.
- **Raporlama**: kampanya bazlı sent/opened/clicked/submitted sayaçları, timeline, CSV export.
- **UI**: jQuery + Bootstrap tabanlı klasik admin paneli.

GoPhish olgun ve güvenilir bir temeldir; aşağıdaki eksikler "kötü" olduğu için değil, **kurumsal
angajman ölçeği, teslimat mühendisliği ve çok-müşterili operasyon** için yetersiz kaldığı için ele alınır.

## 2. Zayıf noktalar → PhishForge çözümleri

| # | GoPhish sınırı | Etki | PhishForge kararı |
|---|----------------|------|-------------------|
| 1 | Tekil monolit + varsayılan SQLite | Eşzamanlı yazımda kilitlenme, tek-örnek ölçek, yatay ölçeklenememe | **PostgreSQL** ana veritabanı; API ile **gönderim worker'ı ayrı süreç**; durum ve kuyruk **Redis**'te. App ve worker bağımsız ölçeklenir. |
| 2 | Basit e-posta/landing editörü (pratikte HTML elde yazılıyor) | Yavaş şablon üretimi, tutarsız kalite, mobil render sürprizleri | **Drag-and-drop + kod (HTML) çift-mod editör**; canlı önizleme (masaüstü/mobil), merge-tag ekleyici, şablon kütüphanesi/versiyonlama. Landing için "yakala/klonla" + görsel düzenleme. |
| 3 | Teslimat (deliverability) araç yok | E-postalar spam'e düşünce test kullanıcı davranışını değil filtreyi ölçer; kör uçuş | **Deliverability modülü**: gönderim öncesi SPF/DKIM/DMARC doğrulama, SpamAssassin spam skoru, RBL/blocklist kontrolü, HTML lint, seed-list inbox-placement testi, warm-up planlayıcı, DKIM anahtar + DNS kayıt sihirbazı. (Meşru e-posta mühendisliği — filtre aldatma numarası değil.) |
| 4 | Zayıf analitik / funnel / timeline | Yöneticiye anlamlı içgörü zor; karşılaştırma yok | **Funnel** (gönderildi→açıldı→tıkladı→gönderdi→bildirdi), kullanıcı-bazlı timeline, kampanya A/B ve kampanyalar-arası karşılaştırma, PDF/CSV yönetici raporu. |
| 5 | Warm-up / hız kontrolü / TZ-bilinçli gönderim yok | Ani hacim → itibar kaybı & spam; yanlış saatte gönderim | Worker'da **rate-limit + throttle + warm-up eğrisi**, alıcı **zaman dilimine göre zamanlama**, pencere/jitter ayarları. |
| 6 | Çok-kiracılık ve RBAC yok | Birden çok müşteri verisi izole edilemez; en az yetki uygulanamaz | **Multi-tenant** (org/engagement izolasyonu) + **RBAC** (admin / operatör / salt-okunur-müşteri). Her sorgu tenant-scoped. |
| 7 | Farkındalık eğitimi entegrasyonu yok | "Yakala" var ama "eğit" yok; döngü kapanmıyor | Tıklayan/gönderen kullanıcı **otomatik awareness eğitim sayfasına/kaydına** yönlendirilir; teslim ve tamamlanma takibi. |
| 8 | Bildirim/entegrasyon zayıf | Ekip olayları geç görür; SIEM/sohbet entegrasyonu yok | **Webhook + Slack/Teams + e-posta** bildirimleri (kampanya olayları, "phishing bildirildi"), imzalı webhook payload'ları. |

## 3. GoPhish'ten devralınan iyi fikirler (koruyacaklarımız)

- `RId` benzeri **imzalı, benzersiz hedef token'ı** (tahmin edilemez; HMAC ile imzalı).
- Tracking pixel + benzersiz tıklama linki modeli.
- Go `text/template` uyumlu merge-tag sözdizimi (mevcut şablonlarla tanıdıklık).
- Tek-binary kolay dağıtım kolaylığı → biz **tek `docker compose up`** ile karşılıyoruz.

## 4. Güvenlik / yetkilendirme farkı (PhishForge'un ayrıştığı yer)

GoPhish "araç"tır; kötüye kullanım engeli operatöre bırakılır. PhishForge, **angajman guardrail'lerini
üründe birinci sınıf** yapar:

- **Engagement nesnesi**: müşteri, yazılı yetki referansı, geçerlilik tarih aralığı, kapsam.
- **Hedef allowlist'i**: kampanya yalnızca aktif+geçerli angajmanın izinli domain/e-posta desenlerine gönderir; kapsam dışı adres reddedilir.
- **Değiştirilemez audit log**: kim, ne zaman, hangi kampanyayı, kime, hangi sonuçla.
- **Veri minimizasyonu**: landing form gönderiminde **ham parola saklanmaz**; yalnızca "submitted" olayı + meta-veri.
- İlk kurulumda **yalnızca-yetkili-kullanım onayı**.

## 5. Kapsam dışı (bilinçli olarak yapmayacaklarımız)

- Gerçek kimlik bilgisi / oturum token'ı (MFA) hasadı ve reverse-proxy relay. Ürün **davranış ölçümü + eğitim** odaklıdır, gerçek hesap ele geçirme değil.
- Spam filtresini **aldatmaya** yönelik obfuscation/gizleme teknikleri. Teslimat, meşru e-posta
  altyapısı ve müşteriyle koordineli allowlist ile sağlanır.
