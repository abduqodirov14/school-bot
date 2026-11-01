# school-bot

O'zbekiston maktablari uchun zamonaviy **Maktab Davomati (School Attendance)** tizimi — Telegram Bot + Telegram Mini App (Web App) + REST API.

---

## 📌 Texnologiyalar (Tech Stack)

- **Backend:** Go 1.22+, Chi router, `github.com/go-telegram/bot`
- **Database:** PostgreSQL (Supabase / AWS RDS), `pgx/v5` connection pool, SQL queries via `sqlc`, migratsiyalar `goose`
- **Frontend (Mini App):** React 19 + TypeScript + Vite + Tailwind CSS (`@telegram-apps/sdk`)
- **Hisobotlar (Excel):** `github.com/xuri/excelize/v2` (.xlsx ko'rinishida ko'p varaqli, rangli katakchalar, muzlatilgan panellar)
- **Avtomatizatsiya (Scheduler):** `github.com/robfig/cron/v3` (`Asia/Tashkent` vaqt zonasida)
- **Konteynerlash:** Docker multi-stage build & Docker Compose

---

## 👥 Foydalanuvchi rollari va huquqlari

1. **Direktor (director):**
   - Maktabdagi barcha sinflar va o'quvchilar umumiy statistikasini ko'radi (Mini App boshqaruv paneli).
   - Yangi o'qituvchilarni tasdiqlaydi (bot orqali tasdiqlash/rad etish inline tugmalari).
   - Sinflar va o'quvchilarni yaratish, tahrirlash, o'chirish.
   - Excel orqali o'quvchilar ro'yxatini yuklash (`POST /api/students/import`).
   - Oylik Excel davomat hisobotini yuklab olish (`GET /api/reports/monthly` yoki `/hisobot`).
   - Bot buyruqlari: `/bugun`, `/kelmaganlar`, `/sinflar`, `/hisobot`.
   - Har bir o'quvchi bo'yicha shaxsiy oylik kalendar tarixi.
   - Sozlamalar: oylik ruxsatsiz dars qoldirish chegarasi (standart: 3) va davomat topshirish vaqti (standart: 09:00).

2. **O'qituvchi (teacher):**
   - Faqat o'ziga biriktirilgan sinf(lar) davomatini oladi.
   - Mini App ochilganda bugungi sana va barcha o'quvchilar "Belgilanmagan" (unmarked) holatda chiqadi.
   - **"Hammasi keldi"** tugmasi bir bosishda barcha o'quvchilarni "Keldi" qiladi, so'ng faqat kelmaganlarni o'zgartirish kifoya.
   - Statuslar: `Keldi (+)` -> `Kelmadi (-)` -> `Kechikdi (K)` -> `Sababli (S)`.
   - Kun davomida davomatni qayta tahrirlash imkoniyati (har bir o'zgarish `attendance_log` audit jadvaliga yoziladi).
   - Bot orqali inline tugmali variant: `/davomat` buyrug'i.

---

## ⏰ Avtomatizatsiya (Cron Schedules)

Barcha vaqtlar **Asia/Tashkent** bo'yicha ishlaydi:
- **Belgilangan vaqtda (masalan, 09:00):** Davomat olmagan sinf o'qituvchilariga bot orqali eslatma yuboriladi.
- **Belgilangan vaqt + 30 daqiqa (masalan, 09:30):** Hali ham davomat olmagan sinflar ro'yxati direktorga yuboriladi.
- **Ruxsatsiz dars qoldirish chegarasi (Threshold alert):** O'quvchi bir oyda 3 marta (yoki sozlangan sondan ko'p) sababsiz dars qoldirsa, direktorga darhol ogohlantirish yuboriladi (har bir o'quvchi uchun oyiga faqat 1 marta).

---

## 🚀 O'rnatish va Ishga tushirish (Local Setup)

### 1. Talablar:
- Go 1.22+
- Node.js 20+ va npm
- PostgreSQL (yoki Supabase)

### 2. Konfiguratsiya:
`.env` faylini yarating:
```bash
cp .env.example .env
```
Parametrlarni kiriting:
```ini
PORT=8080
BOT_TOKEN=1234567890:ABCdefGhIJKlmNoPQRsTUVwxyZ
DATABASE_URL=postgresql://postgres.vlavjveawxlrevfeljop:school-bot1%40112121212@aws-0-eu-north-1.pooler.supabase.com:5432/postgres?sslmode=require
WEB_APP_URL=https://your-domain-or-tunnel.com
DIRECTOR_TELEGRAM_ID=YOUR_TELEGRAM_ID
TIMEZONE=Asia/Tashkent
```

> **Eslatma:** Supabase ulanishida parolida `@` belgisi bo'lsa, URL-encoded ko'rinishda `%40` deb yoziladi va session ulanish uchun port `5432` ishlatiladi.

### 3. Migratsiyalarni ishga tushirish:
Tizim dastur ishga tushganda avtomatik ravishda migratsiyalarni bajaradi. Qo'lda tekshirish uchun:
```bash
goose -dir migrations postgres "$DATABASE_URL" up
```

### 4. Frontendni qurish (Build Frontend):
```bash
cd web
npm install
npm run build
cd ..
```

### 5. Backendni ishga tushirish:
```bash
go run ./cmd/server
```
Server `http://localhost:8080` manzilida ishga tushadi va bir vaqtning o'zida Telegram Bot va Web App'ni xizmat ko'rsatadi.

---

## 🤖 Telegram Bot & Mini App ni BotFather da ulash

1. [@BotFather](https://t.me/BotFather) ga kiring va `/newbot` buyrug'i bilan yangi bot yarating. Olingan `token`ni `BOT_TOKEN`ga qo'ying.
2. Bot menyusiga Mini App tugmasini qo'shish:
   - BotFather da `/mybots` -> Botingizni tanlang -> `Bot Settings` -> `Menu Button` -> `Configure menu button`.
   - Mini App URL manzilini kiriting: `https://your-domain.com` (HTTPS shart, ngrok yoki domen).
   - Tugma nomiga masalan: `📱 Davomat ilovasi` deb yozing.
3. Yangi Web App havolasi (ixtiyoriy):
   - BotFather da `/newapp` -> Botingizni tanlang -> Ilova nomi va rasmini bering -> URL sifatida `https://your-domain.com` kiriting -> Short name oling (masalan, `app`).
   - To'g'ridan-to'g'ri havola: `https://t.me/your_bot?startapp=launch`.

---

## 🐳 Docker orqali Deploy qilish

Yagona buyruq bilan barcha komponentlarni ko'tarish:
```bash
docker compose up -d --build
```
`Dockerfile` multi-stage:
1. Node.js orqali Vite frontendni to'liq kompilyatsiya qiladi.
2. Go kompilyatori orqali statik binar fayl yaratadi.
3. Alpine asosidagi yengil konteynerda `Asia/Tashkent` vaqt zonasi bilan xavfsiz ishga tushiradi.

---

## 📂 Loyiha tuzilmasi (Clean Architecture)

```
├── cmd/
│   └── server/          # Main entrypoint (Chi HTTP router, Bot, Cron, SPA static server)
├── internal/
│   ├── api/             # REST API handlerlar va initData HMAC-SHA256 middleware
│   ├── auth/            # Telegram Web App initData validatsiyasi
│   ├── bot/             # Telegram Bot mantiqi, inline tasdiqlash, buyruqlar, davomat tugmalari
│   ├── config/          # Muhit o'zgaruvchilari (env) yuklovchi
│   ├── db/              # sqlc generatsiya qilgan kod va pgxpool ulanish
│   ├── export/          # Excelize orqali zamonaviy .xlsx hisoboti
│   ├── scheduler/       # Robfig Cron eslatmalar va dars qoldirish ogohlantirishlari
│   └── service/         # Davomat biznes logikasi va status o'zgarishlari
├── migrations/          # Goose SQL migratsiyalari
├── supabase/            # Supabase CLI uchun migratsiyalar
├── web/                 # React 19 + TypeScript + Vite + Tailwind Mini App
├── Dockerfile           # Multi-stage production container
├── docker-compose.yml   # Docker compose konfiguratsiyasi
└── README.md
```

---

## 🧪 Sinovlar (Unit Tests)

Barcha asosiy mantiq to'liq testlangan:
```bash
go test -v ./...
```
- Telegram `initData` HMAC-SHA256 tekshiruvi va soxtalashtirishdan himoya
- Oylik sababsiz dars qoldirish chegarasi (threshold) mantiqi va oyiga 1 marta ogohlantirish
- Davomat statuslarining almashinuvi (`unmarked` -> `present` -> `absent` -> `late` -> `excused`)
- Excel import va ko'p varaqli hisobot generatsiyasi

