# Sistem Auto Leave Userbot

## Deskripsi

Sistem auto leave userbot telah dikonfigurasi untuk meninggalkan chat secara otomatis ketika memenuhi kondisi berikut:

### Kondisi Auto Leave:

1. **Tidak ada musik yang sedang diputar** (`IsActive = false`)
2. **Tidak ada queue lagu** (panjang queue = 0)
3. **Durasi inaktif sudah mencapai 10 menit**

### Detail Implementasi:

#### Interval Pengecekan

- Service memeriksa setiap **3 menit** sekali
- Delay awal: 30 detik setelah bot dimulai

#### Threshold Waktu

- **10 menit** (600 detik) tanpa aktivitas musik
- Waktu dihitung dari `LastActive` di cache

#### Proses:

1. Bot memeriksa semua dialog userbot
2. Untuk setiap chat:
   - Cek apakah ada musik yang sedang diputar
   - Cek apakah ada lagu di queue
   - Cek kapan terakhir kali chat aktif
3. Jika ketiga kondisi terpenuhi (tidak aktif, tidak ada queue, dan sudah 10 menit), userbot akan:
   - Leave dari channel/grup tersebut
   - Membersihkan cache chat
   - Log aktivitas leave

#### Error Handling:

- Mengabaikan error `USER_NOT_PARTICIPANT` (sudah tidak di grup)
- Mengabaikan error `CHANNEL_PRIVATE` (channel private)
- Log error lainnya untuk monitoring

### Konfigurasi:

Service dapat dinonaktifkan dengan mengatur `AutoLeaveTime` di config menjadi 0 atau nilai negatif.

### File yang Dimodifikasi:

- `src/vc/auto_leave.go` - Implementasi sistem auto leave

### Dependencies:

- `src/core/cache/chat_cache.go` - Untuk tracking aktivitas chat dan queue
- `github.com/amarnathcjd/gogram/telegram` - Untuk interaksi Telegram API
