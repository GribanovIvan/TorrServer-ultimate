# Звідки що взято

Ця гілка (`ultimate`) поєднує корисне з кількох форків TorrServer, зберігаючи
авторство. Ліцензія та copyright-рядки — без змін (GPL-3.0, див. `LICENSE`).

| Звідки | Що взято | Як |
| --- | --- | --- |
| [lampac-nextgen/TorrServer](https://github.com/lampac-nextgen/TorrServer) `master` (94bbb4c7) | Основа: вся історія й усі виправлення кешу, гонок, витоків, форк торент-бібліотеки v1.2.47, вебінтерфейс на React | Клон з оригінальними хешами |
| [YouROK/TorrServer](https://github.com/YouROK/TorrServer) `master` (0c48c9ac, MatriX.150) | Нічого в дереві: коміт лише підвищує версію й перезбирає старий вебінтерфейс | `merge -s ours`, запис у history |
| [AndrewSlastenko/TorrServer](https://github.com/AndrewSlastenko/TorrServer) `home-disk-cache` | Файл шматка лишається відкритим, тести | Справжній merge (fdeea3c4) |
| [Tommixoft/TorrServer](https://github.com/Tommixoft/TorrServer) `master` | Мережеві налаштування, вікно прелоаду від розміру цілі, виправлення завислого reader у Telegram-боті, закриття `r.Body` у MSX | Справжній merge (ddc9b9ae, 90805625), частково |
| [etidart/TorrServer-slim](https://github.com/etidart/TorrServer-slim) | Ідея `unix:/шлях` у параметрі адреси (c913cd83) | Переписано під netbind, `Co-authored-by` |
| [pxlvoid/TorrServer](https://github.com/pxlvoid/TorrServer) `homelab` | Позначка шматків, що пройшли перевірку хешем, щоб дисковий кеш переживав перезапуск (9750ede4) | Переписано компактно, `Co-authored-by` |
| [dhvcc/TorrServer](https://github.com/dhvcc/TorrServer) `lampa-stack` | Віддача вже завантаженого файла з диска, `--downloaddir` (84c27c02) | Перенесено, `--author` dhvcc |
| [VictorFedorov/TorrServer](https://github.com/VictorFedorov/TorrServer) `master` | Заборона вимкнути і TCP, і uTP (9f049a79); реальний прогрес прелоаду (bc0f3f7f) | `cherry-pick -x`, автор збережений |
| [reyohoho/TorrServer](https://github.com/reyohoho/TorrServer) `master` | Профіль full: перечитування `accs.db` на льоту, `--disable-shutdown` (1496fa0a) | Перенесено, `Co-authored-by` |
| [freetushkan/TorrServer](https://github.com/freetushkan/TorrServer) `master` | Профіль full: окремі торенти й позначки переглянутого на кожен обліковий запис, `--peruserdata` (2c27db8f) | `cherry-pick -x`, автор Ruslan |
| [sirin-media/TorrServer-NG](https://github.com/sirin-media/TorrServer-NG) `master` | Профіль full: субтитри MKV (`/subs?embedded`), шрифти ASS/SSA (`?fonts`), зовнішні аудіодоріжки (`/audio`), ліміти субтитрів | Справжній merge (5cfb323d), частково: файли NG під тегом `full` |

Власні коміти цієї гілки (інтеграція): межа дескрипторів дискового кешу та
запис вирішень конфліктів у повідомленнях злиттів.

## Свідомо не взято

- **Tommixoft**: мережеві значення, `Seed` з лімітом 128 КБ/с, `TotalHalfOpenConns` 100, `ConnectionsLimit` 40 і вікно прелоаду тепер лише в профілі full; не взято й там: `dropOthers` при старті потоку (закриває інші торенти); пам'ять пріоритетів шматків (у lampac кеш не викликає `SetPriority`); freelist буферів (є `piecebuf.go`); зміни `cache.go`/`reader.go`, що конфліктували з новішою логікою lampac.
- **pxlvoid**: решта режиму `homelab` (~5900 рядків: прибиральник із бюджетом, черга докачки, наступна серія наперед, ntfy, MKV-аудіо, «хто дивиться», свій UI) — написано під старі внутрішності кешу (`setLoadPriority`) і вебінтерфейс CRA. Наступна серія без глобальної черги/бюджету не має сенсу: лишилось не перенесеним.
- **etidart/TorrServer-slim**: сам форк (видалено вебінтерфейс, авторизацію, дисковий кеш, DLNA; позаду на 552).
- **VictorFedorov**: решта (те, що вже є в lampac, як-от звільнення буферів при `Close`, deadlock `CloseReader`, pprof, захист `OpenTorrent`; `Connection: close` лишено як у lampac).
- **sirin-media/TorrServer-NG**: офлайн-режим із квотами і ротацією (~3000 рядків, вбудований у життєвий цикл торента, сховище й API, які в lampac переписані); зміни GStreamer (зовнішнє аудіо в HLS, кодек ASS у probe); плеєр і JASSUB — старий вебінтерфейс.
- **Fktrcfylh/TorrServer-Stash**: попереднє завантаження на NAS через Transmission/qBittorrent — змінює семантику `RemTorrent` (помилка, якщо віддалений клієнт не відповів) і коди відповідей `/torrents` у кількох місцях ядра; кандидат на наступний порт.
- **mx57/TorrServer**: перетворення SRT/ASS у WebVTT зроблено в клієнті старого вебінтерфейсу; у lampac це вже робить сервер (`subs.ToVTT`).
- **LuckyRu/torrserver-manager**: застосунок для Windows і плагін для ТВ (клієнти поза сервером); його форк сервера — виправлення GStreamer на основі MatriX.145.
- **pxlvoid** (наступна серія наперед) — див. вище, без черги й бюджету homelab не переноситься.
- Форки radixun/Remix, 9000000 — не брались.
- Шум форків (CI, Docker, README, релізні скрипти, бренди) — не брали.

## Де перевага за lampac

Де дві гілки змінювали одне місце по-різному, лишалась версія lampac; виняток —
очевидно додаткові зміни. Конфлікти описані у повідомленнях відповідних злиттів.

## Профілі збірки

Один код, три профілі за тегами Go (`//go:build`); прибране в профілі не
компілюється разом із залежностями. Виправлення гонок, витоків і дедлоків —
в усіх профілях.

| Профіль | Збірка | Версія | Що це |
| --- | --- | --- | --- |
| типовий | `make ultimate` | `Ultimate.150` | Функції й типові значення офіційного MatriX.150 на виправленнях lampac |
| slim | `make ultimate-slim` (`-tags slim`) | `Ultimate.150-slim` | Безголовий стрімер: API торентів, потоку, плейлистів, переглянутого й налаштувань, Basic-авторизація з `accs.db`, `--ip` і Unix-сокет, кеш у RAM |
| full | `make ultimate-full` (`-tags full`) | `Ultimate.150-full` | Типовий плюс додатки форків (таблиця вище) |

`-tags "slim full"` не збирається. Теги `gst`, `nosqlite` поєднуються з будь-яким профілем.
