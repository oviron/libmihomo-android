# Security policy

## Reporting a vulnerability

Send a private report to **awdonkin@gmail.com** with subject prefix
`[libmihomo-android security]`. PGP-encrypt to fingerprint
`1139 C91B 6525 883E 6783 DCF0 4A94 DA48 8A4C 5033` if the issue is sensitive
(public key in `oviron-signing.pub.asc`). For coordinated disclosure GitHub
security advisories are also accepted: https://github.com/oviron/libmihomo-android/security/advisories/new.

## Response SLA

- Acknowledgement of receipt: **within 72 hours**.
- Triage and severity classification: within 7 days.
- Fix or detailed mitigation plan: within 30 days for high/critical, 90 days for medium/low.

## Scope

This repository ships a JNI bridge to upstream
[metacubex/mihomo](https://github.com/MetaCubeX/mihomo). Vulnerabilities in
the bridge code (`src/main/jni/core/`) and the build pipeline are in scope.
Issues inside upstream mihomo should be reported to that project directly.

## Supported versions

Only the latest tagged release is supported. Prerelease tags
(`v0.X.Y-rc.N`) are not supported.
