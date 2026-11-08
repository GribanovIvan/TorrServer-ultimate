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

Власні коміти цієї гілки (інтеграція): межа дескрипторів дискового кешу та
запис вирішень конфліктів у повідомленнях злиттів.

## Свідомо не взято

- **Tommixoft**: `Seed` увімкнений завжди з лімітом 128 КБ/с за замовчуванням (змінює поведінку); `TotalHalfOpenConns` 100 і `ConnectionsLimit` 40 (у lampac 500 і 25, на це спираються тести, веб і бот); `dropOthers` при старті потоку (не в переліку, закриває інші торенти); пам'ять пріоритетів шматків (у lampac кеш не викликає `SetPriority`); freelist буферів (є `piecebuf.go`); зміни `cache.go`/`reader.go`, що конфліктували з новішою логікою lampac.
- **pxlvoid**: решта режиму `homelab` (~5900 рядків: прибиральник із бюджетом, черга докачки, наступна серія наперед, ntfy, MKV-аудіо, «хто дивиться», свій UI) — написано під старі внутрішності кешу (`setLoadPriority`) і вебінтерфейс CRA. Наступна серія без глобальної черги/бюджету не має сенсу: лишилось не перенесеним.
- **etidart/TorrServer-slim**: сам форк (видалено вебінтерфейс, авторизацію, дисковий кеш, DLNA; позаду на 552).
- **VictorFedorov**: решта (те, що вже є в lampac, як-от звільнення буферів при `Close`, deadlock `CloseReader`, pprof, захист `OpenTorrent`; `Connection: close` лишено як у lampac).
- Форки radixun/Remix, sirin-media/NG, 9000000, reyohoho, LuckyRu — не брались.
- Шум форків (CI, Docker, README, релізні скрипти, бренди) — не брали.

## Де перевага за lampac

Де дві гілки змінювали одне місце по-різному, лишалась версія lampac; виняток —
очевидно додаткові зміни. Конфлікти описані у повідомленнях відповідних злиттів.
