# Privacy Policy

This page documents what data leaves your machine when you run `mxlrcgo-svc`,
and what does not.

## Musixmatch (default provider)

**What is sent.** Each lyrics lookup sends an HTTPS request to the Musixmatch
API. The request includes the track name, artist name, and (where available)
album name as query parameters. Your API token (`usertoken`) is also transmitted
as a query parameter in the URL.

**Credentials scope.** The Musixmatch token is read from the CLI flag, the
`MUSIXMATCH_TOKEN` environment variable, or the TOML config file, in that order
of precedence. It is transmitted only to `apic.musixmatch.com`; it is never
sent anywhere else.

## PetitLyrics (optional provider)

**What is sent.** When PetitLyrics is enabled, each lookup sends track name and
artist name to the PetitLyrics API. No credentials are required or transmitted
for this provider.

## Local cache

**What stays local.** Lookup results are stored in a local SQLite database
(path resolved via XDG conventions). Nothing in the cache is transmitted
anywhere. Subsequent lookups for the same track are served from this local cache
without contacting any external API.

## Telemetry, analytics, and crash reporting

**None.** `mxlrcgo-svc` does not collect telemetry, analytics, or crash
reports. No data is ever sent to the project maintainers or any third party
other than the lyrics providers listed above.

## Summary

| Data                         | Destination          |
|------------------------------|----------------------|
| Track name, artist name      | Active lyrics provider API |
| Album name (when available)  | Musixmatch API only  |
| Musixmatch token             | Musixmatch API only  |
| Lookup results               | Local SQLite cache   |
| Telemetry / analytics        | Not collected        |

## Cross-references

- `internal/musixmatch/client.go` - Musixmatch HTTP client (query parameters and token handling)
- `internal/petitlyrics/client.go` - PetitLyrics HTTP client (no credentials transmitted)
- `internal/cache/` - local SQLite cache repository
