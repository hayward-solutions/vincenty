# Comparisons

How Vincenty compares to other situational awareness, location tracking, and team coordination platforms.

Platforms are grouped by category. Vincenty is included in every table as the baseline.

---

## Open-source / TAK ecosystem

| | Vincenty | TAK / ATAK | FreeTAK Server | OpenTAK Server | Traccar |
|---|---|---|---|---|---|
| **Primary use** | SA, location tracking, team comms | Military tactical SA + battle management | Community TAK-compatible server | Lightweight TAK-compatible server | GPS fleet / asset tracking |
| **Target market** | Any team needing lightweight SA | Military, government, first responders | Community orgs, researchers | Edge deployments, hobbyists | Businesses, fleet operators |
| **License** | MIT | Gov / proprietary | Apache 2.0 | Open source | Apache 2.0 |
| **Pricing** | Free | Free (govt) / licensed (commercial) | Free | Free | Free |
| **Deployment** | Docker, Kubernetes, AWS ECS, air-gapped | Mobile + cloud / on-prem TAK Server | Cloud, Pi, Docker, pip | Raspberry Pi, Ubuntu, Docker | Self-hosted |
| **Air-gap support** | ✅ Zero internet dependency by design | ✅ Mesh + HF radio | ⚠️ Partial | ⚠️ Partial (ZeroTier) | ⚠️ Self-hosted infra only |
| **Real-time tracking** | ✅ WebSocket | ✅ CoT protocol | ✅ CoT protocol | ✅ CoT protocol | ✅ Real-time GPS |
| **Map visualisation** | ✅ MapLibre GL JS, custom tile sources | ✅ Offline maps + NATO symbology | ✅ Via TAK clients | ✅ Web dashboard live map | ✅ Basic maps + trip history |
| **Group / team chat** | ✅ Group + DM + file attachments | ✅ Chat + PTT voice | ✅ Chat | ✅ Chat | ❌ |
| **Web UI** | ✅ Full Next.js app | ⚠️ Limited | ✅ Dashboard | ✅ Dashboard + live map | ✅ Web interface |
| **Mobile clients** | ✅ Native iOS (SwiftUI) | ✅ Android (ATAK), iOS (iTAK), Windows (WinTAK) | ✅ Via TAK clients | ✅ Via TAK clients | ✅ Android + iOS |
| **REST / WebSocket API** | ✅ 40+ endpoints + WebSocket | ✅ Plugin-based | ✅ REST | ✅ FastAPI REST | ✅ REST |
| **MFA / Auth** | ✅ TOTP, WebAuthn / FIDO2, passkeys | ✅ PKI / X.509 certificates | ⚠️ Basic | ✅ LDAP / AD + certificates | ⚠️ Basic |
| **CoT / ATAK ingest** | ✅ Ingest CoT XML events | ✅ Native (full protocol) | ✅ Full CoT server | ✅ Full CoT server | ❌ |
| **Location history export** | ✅ GPX | ⚠️ Via plugins | ⚠️ Limited | ⚠️ Limited | ✅ Reports + Excel |
| **Audit logging** | ✅ Comprehensive | ✅ Enterprise | ⚠️ Limited | ⚠️ Limited | ⚠️ Basic |
| **Drone / sensor integration** | ❌ | ✅ Extensive via plugins | ⚠️ Via plugins | ❌ | ❌ |
| **Hardened containers** | ✅ Distroless, non-root | ❌ | ❌ | ❌ | ❌ |
| **Kubernetes / Helm** | ✅ Manifests + Helm chart | ⚠️ Not native | ❌ | ❌ | ❌ |
| **Ease of deployment** | ✅ High — `make dev`, env-vars, auto migrations | ⚠️ Low — complex PKI / certificate management | ✅ Medium | ✅ High — automated installer | ✅ Medium |
| **Maturity** | Early OSS | Very high (15+ years) | Medium | Medium-low | High (10+ years) |

---

## Enterprise military / government C2

| | Vincenty | SitAware (Systematic) | Palantir Gotham | Blue Force Tracker (BFT) |
|---|---|---|---|---|
| **Primary use** | Lightweight SA + team comms | Full military C4ISR + battle management | Intelligence fusion + AI-driven operations | US Army vehicle position tracking |
| **Target market** | Any team | NATO militaries — 45+ nations | US / allied intelligence and defence agencies | US Army combat vehicles |
| **License** | MIT | Proprietary (enterprise contract) | Proprietary (government contract) | Government-owned — not commercially available |
| **Pricing** | Free | Undisclosed — defence procurement | $millions — enterprise government contract | N/A (government infrastructure) |
| **Deployment** | Docker, Kubernetes, AWS ECS, air-gapped | Cloud + on-prem (BattleCloud 2025) | On-prem / Gov cloud (air-gapped by design) | Dedicated hardware terminals (FBCB2 / JCR) |
| **Air-gap support** | ✅ Zero internet dependency | ⚠️ On-prem option | ✅ Designed for classified networks | ✅ SINCGARS / JTRS radio mesh |
| **Real-time tracking** | ✅ WebSocket | ✅ Real-time common operating picture | ✅ Real-time geospatial | ✅ Vehicle GPS via radio |
| **Map visualisation** | ✅ MapLibre GL JS, custom tiles | ✅ Advanced geospatial + overlays | ✅ Multi-source geospatial fusion | ✅ Blue force / red force overlays |
| **Group / team chat** | ✅ Group + DM + attachments | ✅ Messaging integration | ⚠️ Limited (data-focused) | ❌ Radio comms only |
| **Web UI** | ✅ Full Next.js app | ✅ HQ UI | ✅ Analyst workbench | ❌ Ruggedised hardware terminals only |
| **Mobile clients** | ✅ Native iOS | ✅ Mounted + dismounted clients | ⚠️ Limited (analyst-focused) | ❌ Vehicle-mounted terminals only |
| **REST / WebSocket API** | ✅ 40+ endpoints + WebSocket | ✅ API integrations | ✅ REST (OAuth 2.0) | ❌ |
| **MFA / Auth** | ✅ TOTP, WebAuthn / FIDO2, passkeys | ✅ Enterprise SSO + classified ACLs | ✅ Government-grade identity management | ✅ CAC / PKI |
| **AI / intelligence analysis** | ❌ | ✅ SitaWare Insight (AI-powered) | ✅ Core capability — pattern recognition, targeting | ❌ |
| **Drone / sensor integration** | ❌ | ✅ Multi-domain sensors | ✅ Satellite, SIGINT, drone feeds | ⚠️ Limited |
| **Kubernetes / Helm** | ✅ | ✅ BattleCloud (2025) | ✅ Kubernetes-based | ❌ |
| **Ease of deployment** | ✅ High | ⚠️ Low — enterprise procurement | ⚠️ Very low — requires forward-deployed engineers | ❌ Requires Army programme integration |
| **Maturity** | Early OSS | Very high (25+ years, combat-proven) | Very high (20+ years) | Very high (30+ years, legacy) |

---

## Commercial SA / geospatial platforms

| | Vincenty | Esri ArcGIS Velocity | Mutualink IRAPP |
|---|---|---|---|
| **Primary use** | Lightweight SA + team comms | Real-time geospatial streaming analytics | Multi-agency radio, video, and data bridging |
| **Target market** | Any team | Emergency services, fleet ops, enterprise | Multi-agency emergency response, govt, healthcare |
| **License** | MIT | Proprietary SaaS | Proprietary |
| **Pricing** | Free | Annual subscription — Standard / Advanced tiers | Custom enterprise |
| **Deployment** | Docker, Kubernetes, AWS ECS, air-gapped | Cloud-only (no on-prem) | SaaS cloud + on-prem hybrid |
| **Air-gap support** | ✅ Zero internet dependency | ❌ Requires live internet | ⚠️ On-prem option |
| **Real-time tracking** | ✅ WebSocket | ✅ IoT data streaming, 24/7 asset visibility | ✅ Location reporting + tracking |
| **Map visualisation** | ✅ MapLibre GL JS, custom tiles | ✅ Enterprise GIS — full ArcGIS ecosystem | ✅ Smart floor plans + site maps |
| **Group / team chat** | ✅ Group + DM + attachments | ❌ | ✅ Voice, video, data bridging across agencies |
| **Drone / sensor integration** | ❌ | ✅ IoT feeds, weather, sensors | ✅ IoT + video device connectivity |
| **Ease of deployment** | ✅ High | ✅ SaaS — no deployment overhead | ⚠️ Medium — enterprise setup |
| **Maturity** | Early OSS | Very high (Esri dominant 40+ years) | High |
| **Unique differentiator** | Air-gap, open source, lightweight | Deep ArcGIS ecosystem + TAK integration | True multi-agency radio / video / data interop |

---

## First responder / public safety platforms

| | Vincenty | Motorola WAVE PTX | Carbyne | RapidSOS | IamResponding | FirstNet (AT&T) |
|---|---|---|---|---|---|---|
| **Primary use** | SA, location tracking, team comms | Mission-critical PTT + location | Emergency comms — 911 / PSAP | Emergency data aggregation layer | Incident alerting + responder tracking | Dedicated first-responder cellular network |
| **Target market** | Any team | First responders, public safety | 911 call centres, PSAPs | CAD / dispatch integrators | Fire depts, EMS, volunteer orgs | All first responders |
| **Pricing** | Free | ~$8 / user / month | Custom enterprise | Free (UNITE) / ~$2,850 / yr per module | $300–$800 / yr per department | $23–$48 / user / month |
| **Deployment** | Docker, Kubernetes, AWS ECS, air-gapped | Cloud (Azure) + on-prem options | Cloud-only (AWS / AWS GovCloud) | Cloud-only | Cloud SaaS | Network infrastructure |
| **Air-gap support** | ✅ Zero internet dependency | ❌ Requires LTE / Wi-Fi | ❌ Cloud-only | ❌ Cloud-only | ❌ | ❌ Requires cellular coverage |
| **Real-time tracking** | ✅ WebSocket | ✅ GPS location presence | ✅ Caller breadcrumb tracking | ✅ 350M+ connected devices | ✅ Responder availability | ✅ Via third-party apps |
| **Map visualisation** | ✅ MapLibre GL JS, custom tiles | ✅ In-app map | ⚠️ Limited | ⚠️ Data layer only | ⚠️ Incident-based | ⚠️ Via third-party apps |
| **Group / team chat** | ✅ Group + DM + attachments | ✅ PTT + multimedia | ✅ Silent messaging | ❌ | ⚠️ Incident comms only | ✅ Via third-party apps |
| **REST / WebSocket API** | ✅ 40+ endpoints + WebSocket | ⚠️ Limited public docs | ✅ | ✅ Integrates with CAD / dispatch | ❌ | ✅ Developer SDK |
| **AI / intelligence** | ❌ | ❌ | ✅ AI translation + transcription | ⚠️ Data enrichment | ❌ | ❌ |
| **Kubernetes / Helm** | ✅ | ❌ | ✅ Microservices (AWS) | ✅ Cloud-native | ❌ | ❌ |
| **Maturity** | Early OSS | High (Motorola brand) | Medium | Medium-high | High (15k+ agencies) | High (19.5k+ agencies) |
| **Unique differentiator** | Air-gap, open source, self-hosted | Motorola PTT ecosystem, ambient listening | Caller-initiated video / location to 911 | 350M+ device location data partnerships | Lowest cost for fire / EMS alerting | Government-priority Band 14 spectrum |

---

## Team communications with location

| | Vincenty | Orion Labs |
|---|---|---|
| **Primary use** | SA, location tracking, team comms | Voice-first PTT + location for enterprise teams |
| **Target market** | Any team | Security, defence, enterprise frontline workers |
| **Pricing** | Free | $6 / user / month ($5 annual) |
| **Deployment** | Docker, Kubernetes, AWS ECS, air-gapped | Cloud SaaS (mobile-first) |
| **Air-gap support** | ✅ Zero internet dependency | ❌ Requires internet |
| **Real-time tracking** | ✅ WebSocket | ✅ GPS + indoor 3D positioning (add-on) |
| **Map visualisation** | ✅ MapLibre GL JS, custom tiles | ✅ In-app map (basic) |
| **Group / team chat** | ✅ Group + DM + file attachments | ✅ PTT voice + text + multimedia |
| **Mobile clients** | ✅ Native iOS (SwiftUI) | ✅ iOS + Android + Onyx wearable |
| **REST / WebSocket API** | ✅ 40+ endpoints + WebSocket | ⚠️ Not publicly documented |
| **MFA / Auth** | ✅ TOTP, WebAuthn / FIDO2, passkeys | ✅ Enterprise |
| **Real-time translation** | ❌ | ✅ 60+ languages, speech-to-speech |
| **Kubernetes / Helm** | ✅ | ❌ |
| **Unique differentiator** | Air-gap, open source, map-centric SA | Real-time 60-language translation, PTT wearable |

---

## Hardware / satellite / industrial tracking

| | Vincenty | Garmin inReach | RealTrac RTLS |
|---|---|---|---|
| **Primary use** | SA, location tracking, team comms | Satellite two-way comms + personal tracking | Industrial workplace safety RTLS |
| **Target market** | Any team | Outdoor recreation, remote workers, expedition teams | Mining, manufacturing, heavy industry |
| **Pricing** | Free | $7.99–$30+ / month per device (+ $150–$400 hardware) | Contact |
| **Deployment** | Docker, Kubernetes, AWS ECS, air-gapped | Consumer hardware + cloud MapShare portal | Cloud / on-prem + proprietary UWB / RF tags |
| **Air-gap support** | ✅ Zero internet dependency | ✅ Iridium satellite — no terrestrial infrastructure needed | ⚠️ On-prem possible, but requires hardware tags |
| **Real-time tracking** | ✅ WebSocket | ✅ Satellite (with latency) | ✅ UWB / RF indoor + outdoor positioning |
| **Group / team chat** | ✅ Group + DM + file attachments | ❌ Single-user text only (character limits) | ❌ Alert broadcasting only |
| **REST / WebSocket API** | ✅ 40+ endpoints + WebSocket | ✅ Garmin IPC API (JSON) | ✅ REST + WebSocket |
| **MFA / Auth** | ✅ TOTP, WebAuthn / FIDO2, passkeys | ⚠️ Basic | ✅ HTTPS (one / two-way) |
| **Safety features** | ❌ | ✅ SOS via satellite | ✅ Man-down, collision detection, gas monitoring |
| **Proprietary hardware required** | ❌ | ✅ Garmin device required | ✅ UWB / RF tags required |
| **Kubernetes / Helm** | ✅ | ❌ | ❌ |
| **Unique differentiator** | Air-gap, open source, no hardware lock-in | Global satellite coverage with zero terrestrial infrastructure | Man-down + collision detection, indoor UWB precision |

---

## Summary

Vincenty targets teams that need genuine situational awareness — real-time location, maps, and secure messaging — without the operational weight of TAK Server, a cloud SaaS dependency, or proprietary hardware.

| Strength | Vincenty | Closest alternative |
|---|---|---|
| Air-gap by default | ✅ Zero config | TAK / ATAK (mesh + HF radio) |
| Simplest deployment | ✅ `make dev` | OpenTAK Server |
| Passkey / WebAuthn auth | ✅ Built in | No direct equivalent across OSS alternatives |
| Kubernetes-native | ✅ Manifests + Helm | ArcGIS Velocity (managed) |
| Open source + MIT | ✅ | FreeTAK, OpenTAK, Traccar (Apache 2.0) |
| No proprietary hardware | ✅ | FreeTAK, OpenTAK, Traccar |
| CoT / ATAK event ingest | ✅ Ingest only | TAK / ATAK, FreeTAK, OpenTAK (full CoT servers) |
| Full C4ISR / AI analysis | ❌ | SitAware, Palantir Gotham |
| Global satellite coverage | ❌ | Garmin inReach |
| Industrial safety (man-down) | ❌ | RealTrac RTLS |
| 200+ GPS hardware protocols | ❌ | Traccar |
