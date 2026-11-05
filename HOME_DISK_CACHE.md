# Home Windows disk-cache fix

This branch is based on upstream **MatriX.147**, commit `a86d8fa7fd4e07605f240fe69bf299392f7cb023`. The installed local version is **MatriX.147-home-diskio.1**.

## Change and reason

`DiskPiece` previously opened and closed its file for every small read/write. Retain a lazy handle per live piece, allow a read-only handle to upgrade for writing, and serialize access with the piece mutex. Eviction closes/deletes the piece; cache shutdown first rejects new I/O, then closes retained handles, optionally removing disk files. Authentication, IP checks, origin filtering and torrent APIs are unchanged.

On the affected Windows NVMe host, a 16 MiB probe using 16 KiB writes took 4.617 seconds with the original implementation and 0.00663 seconds with the patch. The latter uses OS write cache and is **not sustained physical-drive throughput**. More relevantly, an original full4K stream with a 512 MiB disk cache changed from user-observed stalls to smooth playback over the tested several-minute interval. Whole-film stability was not established by that short test.

## Build and verify

Use Go 1.25.7. From `server`:

```powershell
go test ./torr/storage/...
$env:CGO_ENABLED = '0'
go build -buildvcs=false -tags nosqlite -trimpath -ldflags '-s -w -checklinkname=0 -X server/version.Version=MatriX.147-home-diskio.1' -o TorrServer-windows-amd64.exe ./cmd
```

The added tests cover close/late-I/O behavior, byte integrity under concurrent writes, eviction/reuse, read-only upgrade, concurrent shutdown and cache-file removal. The host-speed budget is opt-in: set `TORRSERVE_IO_TEST_ROOT` to a scratch directory on the target disk; the test creates/removes its own subdirectory. It is not a portable benchmark guarantee.

The currently installed Windows binary SHA256 is `9f4313d98edd6cdc1b64ae321977ab13bc7fd48b40fb5e9b3cf41f33255a435f`. Rebuilding from this fork with the command above produced that exact binary hash. Source changes match the tested/deployed source. No service configuration, local paths, credentials, media, cache or viewing history is included.

## Scope and limits

Retained descriptors scale with touched, non-evicted pieces. The tested 512 MiB cache with large pieces uses a modest count, but there is no independent descriptor cap. Add a bounded handle pool and validate Linux/Android before proposing this as a general upstream change or using huge caches/small pieces on systems with low file limits. The Android box still runs the official server; this patch was installed only on Windows.

Stop playback and the server before replacing the executable or switching between RAM/disk storage. Keep the original executable and current configuration for rollback. No upstream pull request has been opened; this fork preserves the working local patch for maintenance.
