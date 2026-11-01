# school-bot: O'zbekiston Maktablari Uchun Davomat Tizimi (School Attendance MVP)

Telegram Bot + Telegram Mini App (Web App) + REST API orqali maktab davomatini boshqarish va nazorat qilishning to'liq avtomatlashtirilgan zamonaviy tizimi.

UI tili: **O'zbek tili (Lotin)** | Vaqt mintaqasi: **Asia/Tashkent**

---

## 🚀 Texnologiyalar Steki

- **Backend:** Go 1.22+, router: `github.com/go-chi/chi/v5`, Telegram Bot: `github.com/go-telegram/bot`
- **Ma'lumotlar bazasi:** PostgreSQL (yoki Supabase), drayver: `pgx/v5` connection pool, SQL generator: `sqlc`, migratsiyalar: `goose`
- **Excel Eksport & Import:** `github.com/xuri/excelize/v2` (ko'p sahifali formatlangan oylik hisobotlar va o'quvchilarni import qilish)
- **Avtomatlashtirish & Eslatmalar:** `github.com/robfig/cron/v3` (Asia/Tashkent vaqt mintaqasida har daqiqali tekshiruvlar)
- **Frontend (Mini App):** React 19, TypeScript, Vite, Tailwind CSS v4, Lucide Icons, Telegram Web Apps SDK
- **Konteynerlashtirish:** Ko'p bosqichli `Dockerfile` va `docker-compose.yml`

---

## 🎯 Asosiy Rollar va Imkoniyatlar

### 1. Direktor (Director)
- **Mini App Boshqaruv Paneli:**
  - Bugungi maktab bo'yicha umumiy statistika (Jami, Kelgan, Kelmagan, Kechikkan, Sababli).
  - Davomat olinmagan sinflar ro'yxati va ularning o'qituvchilari.
  - Sinflar bo'yicha guruhlangan kelmagan va kechikkan o'quvchilar ro'yxati.
  - O'quvchining oylik davomat kalendar tarixi.
  - Excel formatida oylik to'liq hisobotni yuklab olish.
  - O'quvchilar ro'yxatini Excel fayl orqali ommaviy yuklash (Sinfi, F.I.Sh).
- **Telegram Bot Buyruqlari:**
  - `/bugun` — Bugungi umumiy davomat ko'rsatkichlari.
  - `/kelmaganlar` — Bugun darsga kelmagan barcha o'quvchilar ro'yxati (sinflari bilan).
  - `/sinflar` — Davomat olingan va olinmagan sinflar holati.
  - `/hisobot` — Oylik to'liq hisobotni Excel (.xlsx) fayl ko'rinishida to'g'ridan-to'g'ri Telegramga yuborish.
- **O'qituvchilarni Tasdiqlash:**
  - Yangi foydalanuvchi botga `/start` bosganda holati `pending` bo'ladi.
  - Direktorga tasdiqlash xabari va inline tugmalar keladi: `[ ✅ O'qituvchi qilib tasdiqlash ]` / `[ ❌ Rad etish ]`.
- **Sozlamalar:**
  - Davomat topshirish muddati (Default: `09:00`).
  - Sababsiz qoldirish limiti (Default: `3` ta dars).

### 2. O'qituvchi (Teacher)
- **Telegram Mini App Davomat Ekrani:**
  - O'ziga biriktirilgan sinf ro'yxati va sana.
  - Barcha o'quvchilar dastlab "Belgilanmagan" holatida bo'ladi.
  - **"Hammasi keldi"** tugmasi bir bosishda hammani "keldi" holatiga o'tkazadi.
  - Har bir o'quvchi ustiga bosib holatini tezkor o'zgartirish: *Keldi* (`keldi`), *Kelmadi* (`kelmadi`), *Kechikdi* (`kechikdi`), *Sababli* (`sababli`).
  - **"Saqlash"** tugmasi bosilmaguncha sinf "davomat olinmagan" hisoblanadi.
  - Kun davomida tahrirlash imkoniyati (har bir tahrir `attendance_log` audit jadvaliga yoziladi).
- **Telegram Bot Inline Keyboard Davomati (Fallback):**
  - Mini App ochilmagan holatda botda `/davomat` buyrug'i orqali to'g'ridan-to'g'ri Telegram inline tugmalari bilan davomat qilish mumkin.

### 3. Avtomatlashtirish (Cron Scheduler)
- **09:00 (Muddati):** Davomat olmagan barcha sinf rahbarlariga bot orqali avtomatik eslatma yuboriladi.
- **09:30 (Muddat + 30 daqiqa):** Agar hanuzgacha davomat olinmagan bo'lsa, direktorga davomat olmagan sinflar va o'qituvchilar ro'yxati haqida ogohlantirish yuboriladi.
- **Limitdan oshish (Threshold Alert):** O'quvchining bir oydagi sababsiz dars qoldirishlari soni limitga (masalan, 3 taga) yetganda, direktorga zudlik bilan bildirishnoma yuboriladi (oyiga bir marta).

### 4. Excel Oylik Hisoboti (.xlsx)
- **1-sahifa "Umumiy":** Har bir sinf va butun maktab bo'yicha umumiy qatnashish foizlari.
- **Sinflar sahifalari:** Qatorlarda o'quvchilar, ustunlarda oy kunlari (1..31), katakchalarda rangli belgilar (`+`, `-`, `K`, `S`), o'quvchi bo'yicha jami hisob-kitoblar, qotirilgan panellar (frozen panes).
- Limitdan oshgan o'quvchilar qizil rang bilan ajratib ko'rsatiladi.

---

## 🛠️ O'rnatish va Ishga Tushirish

### 1. Talablar
- Go 1.22 yoki undan yuqori
- Node.js 20+ va npm (agar lokal web build qilinsa)
- PostgreSQL yoki Supabase ma'lumotlar bazasi
- Telegram Bot Token ([@BotFather](https://t.me/BotFather) dan olinadi)

### 2. Muhit O'zgaruvchilari (.env)

Loyihaning ildiz papkasida `.env` fayl yarating:

```env
PORT=8080
BOT_TOKEN=7777777777:AAxxxxxxxxxxxxxxxxxxxxxxxxxxxx
DATABASE_URL=postgresql://postgres.vlavjveawxlrevfeljop:school-bot1%40112121212@aws-0-eu-north-1.pooler.supabase.com:5432/postgres?sslmode=require
WEB_APP_URL=https://your-domain.com
DIRECTOR_TELEGRAM_ID=123456789
TIMEZONE=Asia/Tashkent
```

> **Eslatma (Supabase):** Agar parolingizda `@` belgisi bo'lsa, uni URL-kodlang (masalan `@` o'rniga `%40`). Goose va transaction pooler uchun port `5432` va `?sslmode=require` parametrlarini ishlating.

### 3. Migratsiyalar (Baza Jadvallari)

Dastur ishga tushganda o'rnatilgan goose migratsiyalari avtomatik tarzda bazaga qo'llaniladi. Agar alohida qo'lda ishga tushirmoqchi bo'lsangiz:

```bash
# Goose orqali:
goose -dir migrations postgres "$DATABASE_URL" up

# Supabase CLI orqali (ixtiyoriy):
supabase link --project-ref vlavjveawxlrevfeljop
supabase db push
```

### 4. Lokal Rivojlantirish

```bash
# 1. Frontendni yig'ish (build)
cd web
npm install
npm run build
cd ..

# 2. Serverni ishga tushirish (API + Bot + Cron + Mini App)
go run ./cmd/server
```

Server ishga tushgach:
- REST API: `http://localhost:8080/api`
- Health check: `http://localhost:8080/health`
- Mini App: `http://localhost:8080/`

---

## 🤖 Telegram BotFather orqali Mini App sozlash

1. [@BotFather](https://t.me/BotFather) ga kiring va `/mybots` buyrug'ini bering.
2. O'zingizning botingizni tanlang.
3. **Bot Settings** -> **Menu Button** -> **Configure menu button** ni bosing.
4. Mini App havolasini kiriting (masalan: `https://your-domain.com/` yoki ngrok havolasi).
5. Tugma nomiga masalan **"Davomat tizimi"** deb yozing.
6. Shuningdek, `/newapp` buyrug'i orqali alohida Web App havolasi (masalan `t.me/botusername/app`) yaratishingiz mumkin.

---

## 🐳 Docker va Serverga Joylash (Deployment)

Tizim ko'p bosqichli Dockerfile orqali bitta ixcham konteynerda frontend, backend va barcha resurslarni birlashtiradi.

### Docker orqali ishga tushirish:

```bash
docker compose up -d --build
```

Konteyner holatini tekshirish:
```bash
docker compose logs -f school-attendance
```

---

## 🧪 Testlarni Ishga Tushirish

Tizim barcha asosiy mantiqiy bloklar (initData xavfsizligi, ruxsatlar, limit hisob-kitoblari, davomat sikllari, Excel eksport) bo'yicha to'liq testlar bilan ta'minlangan:

```bash
go test -v ./...
```

---

## 📂 Loyiha Tuzilishi

```text
├── cmd/
│   └── server/          # Asosiy ishga tushiruvchi kod (main.go)
├── db/
│   └── queries/         # sqlc SQL so'rovlari
├── internal/
│   ├── api/             # Chi router HTTP handlerlari va middleware
│   ├── auth/            # Telegram initData HMAC-SHA256 tekshiruvi
│   ├── bot/             # Telegram bot handlerlari va inline interfeyslar
│   ├── config/          # Muhit o'zgaruvchilari konfiguratsiyasi
│   ├── db/              # sqlc generatsiya qilgan kod va pgxpool ulanishi
│   ├── export/          # Excelize orqali formatlangan oylik hisobot eksporti
│   ├── scheduler/       # Cron scheduler (eslatmalar va limit ogohlantirishlari)
│   └── service/         # Davomat va biznes mantiq servislari
├── migrations/          # Goose SQL migratsiyalari
├── supabase/            # Supabase CLI migratsiyalari
├── web/                 # React + TypeScript + Tailwind CSS Telegram Mini App
├── Dockerfile           # Multi-stage Docker konteyner
├── docker-compose.yml   # Docker compose konfiguratsiyasi
└── README.md            # Loyiha qo'llanmasi
```
