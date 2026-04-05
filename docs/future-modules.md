# Future Module Ideas

## Implemented

The following were previously listed here and are now implemented:
Spotify, Medium, Telegram, Chess.com, Trello, Wattpad, MyAnimeList, Roblox, OP.GG, Lichess, Codeforces, Codeberg, Security.txt

## 2026-04 Selection Notes

These were the 2026-04 selections that fit existing patterns best:

### 1. Security.txt
- Reason: domain coverage is still thin compared to username coverage, and `security.txt` is a zero-auth, standards-based source with deterministic fetch paths
- Reliability: `GET /.well-known/security.txt` with `GET /security.txt` fallback is straightforward and easy to test with fixed fixtures
- Graph fit: account properties, direct `email` pivots from `Contact: mailto:`, `website` pivots from HTTP contacts, and derived `domain` pivots from linked hosts
- Verify seed: `securitytxt.org`
- Manual smoke test:
  - `curl -fsSL https://securitytxt.org/.well-known/security.txt`
  - `cd cli && go test ./internal/modules/securitytxt/ -v`

### 2. Lichess
- Reason: zero-auth JSON API, deterministic username lookup, no scraping, same `username -> account/profile links` shape already used by existing username modules
- Reliability: `GET /api/user/{username}` is a clean `200` or `404`
- Graph fit: account properties, optional `full_name`, and linked `website` nodes from `profile.links`
- Verify seed: `lichess`
- Manual smoke test:
  - `curl -fsSL https://lichess.org/api/user/lichess`
  - `cd cli && go test ./internal/modules/lichess/ -v`

### 3. Codeforces
- Reason: zero-auth JSON API, high-signal profile metadata, stable response schema, and no custom auth or scraping
- Reliability: `GET /api/user.info?handles={username}` returns structured JSON for both found and not-found cases
- Graph fit: account properties plus optional `full_name`, `avatar_url`, and `organization`
- Verify seed: `tourist`
- Manual smoke test:
  - `curl -fsSL 'https://codeforces.com/api/user.info?handles=tourist'`
  - `cd cli && go test ./internal/modules/codeforces/ -v`

### 4. Codeberg
- Reason: Gitea-style API that closely matches the existing GitLab and GitHub module patterns
- Reliability: `GET /api/v1/users/{username}` is a clean `200` or `404`
- Graph fit: account, optional `full_name`, `avatar_url`, `website`, and derived `domain`
- Verify seed: `forgejo`
- Manual smoke test:
  - `curl -fsSL https://codeberg.org/api/v1/users/forgejo`
  - `cd cli && go test ./internal/modules/codeberg/ -v`

Deferred despite being attractive:
- Bluesky: public API works, and it is still the best next username module, but domain coverage had the larger gap for this sprint
- Speedrun.com: deferred on April 4, 2026 after a live probe returned a real user for a bogus `name=` lookup, which makes the current search path too nondeterministic for default scans
- ArtStation: deferred on April 4, 2026 because the documented `profile.json` endpoint returned `403`
- WakaTime and HackerRank: privacy and availability constraints make them less reliable for default scanning
- BuyMeACoffee and most Tier 2 ideas: scraping-heavy and inconsistent with the current codebase's preferred API-first modules

---

## Tier 1 - Zero Auth JSON APIs (high signal, easy to implement)

### Speedrun.com (username)
- URL: `GET https://www.speedrun.com/api/v1/users?name={username}` or `/users/{id}`
- Detection: 200 with populated `data` array / empty array
- Data: country/region, signup date, pronouns, **linked social accounts (Twitch, YouTube, Hitbox, Twitter)**
- Notes: Cross-platform links are pivotable - high value. API (v1) not actively maintained, pagination breaks at 10,000 items. A live probe on April 4, 2026 returned a real user for a bogus `name=` lookup, so this needs a deterministic fetch strategy before implementation

### Bluesky (username)
- URL: `GET https://public.api.bsky.app/xrpc/app.bsky.actor.getProfile?actor={username}`
- Detection: 200 (JSON) / error
- Data: persistent DID (immutable identifier), current handle, display name, description, avatar, follower/following count, post count
- Notes: AT Protocol maps domain names to cryptographic DIDs. Track users even after handle changes. Growing fast, tech-forward user base

### Minecraft / Mojang (username)
- URL: `GET https://api.mojang.com/users/profiles/minecraft/{username}`
- Detection: 200 (JSON with id + name) / 204 or 404
- Data: UUID (immutable), current username, historical usernames (via separate endpoint), skin image files
- Rate limit: 200 req/2 min per IP
- Notes: UUID is permanent even if name changes. Historical username arrays defeat pseudonym rotation, linking past and present identities. See also NameMC for skin/server history

### ArtStation (username)
- URL: `GET https://www.artstation.com/users/{username}/profile.json`
- Detection: 200 (JSON) / 404
- Data: display name, bio, avatar, follower count, portfolio stats
- Notes: Professional digital artists and game developers. Deferred on April 4, 2026 because the documented JSON endpoint returned `403`

### HackerRank (username)
- URL: `GET https://www.hackerrank.com/rest/contests/master/hackers/{username}/profile`
- Detection: 200 (JSON) / error
- Data: verified skills, earned badges, country, real name, linked social profiles (GitHub, LinkedIn)
- Notes: Technical interview platform; provides employer-verified skills and direct links to professional networking sites

### WakaTime (username)
- URL: `GET https://wakatime.com/api/v1/users/{username}/stats`
- Detection: 200 (JSON) / 404 or permission error for private profiles
- Data: total time coded, operating system, code editors, programming languages, recent activity
- Notes: IDE plugin metrics; reveals exact technology stack, daily working hours (implies timezone), and employment status

### BuyMeACoffee (username)
- URL: `GET https://buymeacoffee.com/{username}` (scrape) or widget APIs
- Detection: 200 / 404
- Data: display name, bio, connected social media, top financial supporters
- Notes: KYC-verified backend for payouts. Supporter usernames map the target's economic support graph

---

## Tier 2 - Zero Auth but Scraping or XML

### BoardGameGeek (username)
- URL: `GET https://boardgamegeek.com/xmlapi2/user?name={username}`
- Detection: XML with user data / `<user id="" name="notfound">`
- Data: name, geographic location, last login, avatar, collection size, plays
- Extended: `&buddies=1&guilds=1` exposes real-world social graphs (paginated)
- Notes: XML format (not JSON). Strictly rate-limited (min 5s between requests, 500/503 on overload). Buddies parameter reveals real-world social networks

### Letterboxd (username)
- URL: `https://letterboxd.com/{username}/`
- Data: display name, avatar, bio, film stats, linked URLs, Twitter handle, viewing diary
- Method: OG metadata scraping, clean 404 for missing
- Notes: Diary feature provides exact dates/times of media consumption. API exists but requires complex OAuth

### Pinterest (username)
- URL: `https://www.pinterest.com/{username}/`
- Method: OG metadata scraping
- Data: display name, avatar, bio, follower count

### Patreon (username)
- URL: `https://www.patreon.com/{username}`
- Method: OG metadata scraping
- Data: creator name, bio, social links

### Redbubble (username)
- URL: `https://www.redbubble.com/people/{username}`
- Detection: 200 / 404
- Data: artist name, bio, design count
- Method: HTML scraping

### Archive of Our Own / AO3 (username)
- URL: `https://archiveofourown.org/users/{username}/profile`
- Detection: 200 / 404
- Data: "Pseuds" (alternate names linked to same account), biography, join date, bookmarks
- Method: HTML scraping (no formal API)
- Notes: Famously pseudonymous, but users cross-link to Tumblr, Twitter, Dreamwidth. Multiple pseuds immediately map alt-accounts under one identity

### About.me (username)
- URL: `https://about.me/{username}`
- Detection: 200 / 404
- Data: biography, location, occupation, education, outbound links to major social networks
- Method: HTML scraping
- Notes: Users create these specifically to centralize identity - a pre-compiled, curated master list of active platforms

### Goodreads (username)
- URL: `https://www.goodreads.com/user/show/{id}-{username}`
- Detection: 200 / 404
- Data: real name, location, personal website, join date, written reviews
- Method: HTML scraping (API deprecated)
- Notes: High probability of real-name correlation with pseudonyms due to social integration design

### Mastodon / Fediverse (username)
- URL: `GET https://{instance}/.well-known/webfinger?resource=acct:{user}@{instance}`
- Detection: 200 (JRD+JSON) / 404
- Data: profile URL, ActivityPub actor, linked accounts, aliases
- Notes: Check popular instances (mastodon.social, hachyderm.io, fosstodon.org). WebFinger defeats domain-level obfuscation by tracing canonical server location

### IndieWeb h-card Scraping (domain)
- URL: Any personal domain
- Method: Parse HTML DOM for `h-card` CSS class (microformats)
- Data: preferred name (`p-name`), profile photo (`u-photo`), canonical URL (`u-url`), email (`u-email`)
- Notes: Transforms unstructured personal blogs into machine-readable identity profiles

---

## Tier 3 - Infrastructure & Email Enrichment (unique data sources)

### GitHub Commit Email Harvesting (username)
- URL: `GET https://api.github.com/users/{username}/events/public`
- Data: Real email addresses from commit author metadata (even if hidden in profile)
- Alt: Append `.patch` to any commit URL to get raw `From: Name <email>` header
- Notes: Upgrade existing GitHub module. Bypasses GitHub's web-interface privacy masking entirely

### PGP Keyservers (email)
- HKP: `GET https://keys.openpgp.org/pks/lookup?op=get&search={email}`
- VKS: `GET https://keys.openpgp.org/vks/v1/by-email/{email}`
- Ubuntu: `GET https://keyserver.ubuntu.com/pks/lookup?op=index&search={email}`
- Data: PGP fingerprints, key UIDs (real name + email), creation dates, "Web of Trust" signatures (maps social network)
- Notes: Rich identity info rarely checked. Comment field often contains corporate affiliation or secondary aliases. Rate limit ~1 req/min

### Keyoxide (email/fingerprint)
- URL: `https://keyoxide.org/wkd/{user@domain}` or `https://keyoxide.org/{fingerprint}`
- Data: Cryptographic proofs linking accounts across platforms (bidirectional verification)
- Notes: Maps a single email to verified secondary profiles, eliminating username-matching ambiguity. Also checks DNS TXT for `openpgp4fpr`

### npm Registry (username/email)
- URL: `GET https://registry.npmjs.org/-/v1/search?text=author:{username}`
- Metadata: `GET https://registry.npmjs.org/{package}/{version}`
- Data: Published packages, `_npmUser` object with exact email used during `npm publish`, maintainers array
- Notes: Email in `_npmUser` bypasses GitHub email privacy settings completely

### PyPI (username)
- URL: `https://pypi.org/user/{username}/` (page scraping) + `https://pypi.org/pypi/{project}/json`
- Data: published packages, `author_email`, `maintainer_email` from `info` dict, linked GitHub/homepage
- Notes: Developers routinely insert primary personal/corporate email in `setup.py`, permanently exposed via JSON API

### Crates.io / Rust (username)
- URL: `GET https://crates.io/api/v1/crates/{crate}/owners`
- Data: GitHub login handles of crate owners (e.g., `github:username`)
- Notes: Definitively links Rust developer pseudonyms back to GitHub identities

### RubyGems (username)
- URL: `GET https://rubygems.org/api/v1/owners/{username}/gems.json` + `GET https://rubygems.org/api/v1/gems/{gem}/owners.json`
- Data: All gems owned by user, explicit email addresses of all maintainers
- Notes: Direct email extraction from package maintainer metadata

### Docker Hub Manifest Inspection (username)
- Auth: `GET https://auth.docker.io/token?service=registry...` (anonymous bearer token)
- Manifest: `GET /v2/{namespace}/{repo}/manifests/{tag}`
- Data: `MAINTAINER` instruction (name + email), `ENV USER=`, `LABEL maintainer=`, internal paths from Dockerfile history
- Notes: Extracts identity from container build metadata without downloading image layers. Ignored by all major OSINT frameworks

### Wayback CDX Search (domain)
- URL: `GET https://web.archive.org/cdx/search/cdx?url={domain}&output=json&fl=timestamp,original,statuscode`
- Wildcard: `url=*.example.com/*` discovers forgotten subdomains and deep URL paths
- Filter: `&filter=mimetype:application/json` isolates leaked API responses
- Notes: Upgrade existing Wayback module. Reveals old pages, contact info, leaked data dumps

### Common Crawl CDX (domain)
- URL: `GET https://index.commoncrawl.org/CC-MAIN-{YYYY-WW}-index?url=*.example.com/`
- Data: WARC offset and length for raw HTML extraction from AWS S3
- Notes: Independent from Internet Archive. Monthly indices pinpoint URLs briefly live during specific time windows. Discovers mentions on short-lived forums or paste sites

### DNS TXT Record Parsing (domain)
- Method: Direct DNS lookup for TXT records
- Data: Verification tokens map the target's exact SaaS stack:
  - `google-site-verification=` → Google Workspace
  - `facebook-domain-verification=` → Facebook Business/advertising
  - `atlassian-domain-verification=` → Jira/Confluence/Bitbucket
  - `1password-site-verification=` → 1Password enterprise
  - `docusign=` → Digital contracts
- SPF records (`v=spf1 include:amazonses.com`) reveal authorized mail relays (Mailchimp, Sendgrid, Amazon SES)
- Notes: Upgrade existing DNS module. Interpreting SaaS verification tokens to map internal workflow is rarely automated

### Certificate Transparency via crt.sh (domain)
- Web: `https://crt.sh/?q=%.example.com&output=json`
- Direct SQL: `psql -h crt.sh -p 5432 -U guest -d certwatch`
- Data: Every historical SSL certificate, SANs (subdomains), issuer DN contact emails
- Notes: Direct PostgreSQL access bypasses web UI pagination limits. Older certificates often contain admin email addresses

### Security.txt (domain)
- URLs: `https://{domain}/.well-known/security.txt`, `https://{domain}/security.txt`
- Data: security team email contacts, policy URLs, acknowledgments pages, hiring pages, encryption key URLs
- Notes: Implemented in April 2026 as the next domain-focused module because it is standards-based and deterministic

### humans.txt (domain)
- URL: `https://{domain}/humans.txt`
- Data: real names, Twitter handles, personal websites of engineering team
- Notes: Still attractive, but remains separate from `security.txt` because the format is heuristic-heavy and less standardized

### Paste Repository Search (username/email)
- Paste.ee API: `GET https://api.paste.ee/v1/pastes` (requires X-Auth-Token)
- GitHub Gists: `GET https://api.github.com/users/{username}/gists`
- Data: Paste metadata, author accounts, raw code with hardcoded emails/API keys
- Notes: Gists frequently contain unredacted scripts with corporate email addresses

### Homebrew Tap Analysis (username)
- Method: Analyze contributors to custom Homebrew taps via GitHub API
- Data: Tap maintainer network, installation analytics
- Notes: Maps developers maintaining niche macOS software infrastructure

### Libravatar (email)
- DNS Discovery: `SRV _avatars._tcp.{domain}` or `_avatars-sec._tcp.{domain}`
- Default CDN: `GET https://seccdn.libravatar.org/avatar/{md5_or_sha256_hash}`
- Data: Avatar image (enables reverse image search)
- Notes: Federated avatar alternative to Gravatar, popular in Linux/open-source communities. DNS SRV discovery reveals self-hosted avatar infrastructure

---

## Tier 4 - Requires API Key (via ~/.basalt/config)

### Last.fm (username)
- URL: `GET http://ws.audioscrobbler.com/2.0/?method=user.getinfo&user={username}&api_key={key}&format=json`
- Data: play count, age, gender, country, real name, subscriber status
- Config key: `LASTFM_API_KEY` (free registration)
- Notes: Music taste reveals personality; high engagement data

### Have I Been Pwned (email)
- URL: `GET https://haveibeenpwned.com/api/v3/breachedaccount/{email}`
- Header: `hibp-api-key: {key}`
- Data: breach names, dates, data types leaked
- Config key: `HIBP_API_KEY`
- Notes: $3.50/month; extremely high OSINT value for email

### VirusTotal (domain)
- URL: `GET https://www.virustotal.com/api/v3/domains/{domain}`
- Header: `x-apikey: {key}`
- Data: passive DNS, reputation, malware associations, subdomains, SSL certs
- Config key: `VIRUSTOTAL_API_KEY`
- Free: 500 req/day, 4 req/min

### SecurityTrails (domain)
- URL: `GET https://api.securitytrails.com/v1/domain/{domain}`
- Header: `apikey: {key}`
- Data: DNS history, associated domains, subdomain enumeration, WHOIS history
- Config key: `SECURITYTRAILS_API_KEY`
- Free: 50 req/month

### EmailRep.io (email)
- URL: `GET https://emailrep.io/{email}`
- Header: `Key: {key}` (optional)
- Data: reputation score, breach count, profiles detected, domain age
- Config key: `EMAILREP_API_KEY`
- Free without key: 100 req/day

### AlienVault OTX (domain)
- URL: `GET https://otx.alienvault.com/api/v1/indicators/domain/{domain}/general`
- Header: `X-OTX-API-KEY: {key}`
- Data: passive DNS, threat intelligence pulses, URL history
- Config key: `OTX_API_KEY`
- Free: 10k req/hour

### URLScan.io (domain)
- URL: `GET https://urlscan.io/api/v1/search/?q=domain:{domain}`
- Data: technologies detected, third-party services, trackers
- Config key: `URLSCAN_API_KEY`
- Free: search unlimited without auth

### HackerTarget (domain)
- URL: `GET https://api.hackertarget.com/reverseiplookup/?q={domain}`
- Alt: `GET https://api.hackertarget.com/reversedns/?q={ip}`, `GET https://api.hackertarget.com/hostsearch/?q={domain}`
- Data: reverse IP, HTTP headers, subdomains, all domains hosted on an IP
- Config key: `HACKERTARGET_API_KEY`
- Free: 100 req/day (50 for reverse DNS)

### Censys (domain/email)
- URL: `GET https://search.censys.io/api/v2/certificates/search`
- Auth: HTTP Basic (API ID/Secret)
- Query: CenQL syntax, e.g. `parsed.subject.email_address="target@domain.com"`
- Data: Certificates linking email addresses directly to infrastructure (IP addresses, servers)
- Config key: `CENSYS_API_ID`, `CENSYS_API_SECRET`
- Notes: Free tier with rate limits. Bridges human identity to physical infrastructure

### osu! (username)
- URL: `GET https://osu.ppy.sh/api/v2/users/{username}`
- Auth: OAuth 2.0 Client Credentials flow (`client_id` + `client_secret`)
- Data: play stats, country, global rank, join date, last visit timestamp, online status
- Config key: `OSU_CLIENT_ID`, `OSU_CLIENT_SECRET`
- Notes: "Last visit" timestamp reveals activity patterns and timezone. Competitive rhythm game with massive global player base

### RetroAchievements (username)
- URL: `GET https://retroachievements.org/API/API_GetUserSummary.php?u={username}&y={api_key}`
- Detection: JSON with `{"User": "..."}` / error object
- Data: registration date, last activity, total points, recent games, status messages
- Config key: `RETROACHIEVEMENTS_API_KEY` (free registration)
- Notes: Niche retro gaming/emulation community. Precise gaming activity timestamps

### Behance / Adobe (username)
- URL: `GET https://api.behance.net/v2/users/{username}?client_id={api_key}`
- Detection: 200 (JSON) / 404
- Data: first name, last name, occupation, company, location, social links (Twitter, Facebook, LinkedIn, Dribbble)
- Config key: `BEHANCE_API_KEY` (Adobe Developer)
- Notes: Professional creative portfolios. Consistently yields verified full names tied to corporate employment

### ProductHunt (username)
- URL: `POST https://api.producthunt.com/v2/api/graphql`
- Auth: Bearer token
- Data: maker profile, products launched, Twitter handle, personal website URLs
- Config key: `PRODUCTHUNT_TOKEN`
- Notes: Ties pseudonymous developers directly to incorporated businesses and monetized products

### SourceHut (username)
- URL: GraphQL at `https://meta.sr.ht/query`
- Query: `{ user(username: "{username}") { id canonicalName } }`
- Auth: Personal Access Token
- Data: canonical name, bio, location, `externalId` mappings (e.g., `github:username`)
- Config key: `SOURCEHUT_TOKEN`
- Notes: Minimalist open-source dev suite. `externalId` convention deterministically links fragmented developer identities

---

## Tier 5 - Anti-Bot Protected (requires headless browser or advanced techniques)

### LeetCode (username)
- URL: `POST https://leetcode.com/graphql` with `operationName: getUserProfile`
- Detection: non-existent users return null nodes
- Data: real name, avatar, country, company, university, GitHub links, submission history
- Notes: Requires specific User-Agent headers. Cloudflare 403 blocks common. Bridges competitive coding pseudonyms to corporate employment

### Kaggle (username)
- URL: `GET https://www.kaggle.com/{username}/account.json` (internal endpoint)
- Data: user tier (Grandmaster etc.), dataset contributions, competition history, forum activity
- Notes: Hidden internal JSON endpoints may work without auth. Maps targets to academic/corporate research

### Rate Your Music / Sonemic (username)
- URL: `https://rateyourmusic.com/~{username}`
- Data: ratings, reviews, "user compatibility" metric identifying similar users
- Notes: Aggressive Cloudflare Human Verification. Requires TLS fingerprint spoofing or headless browsing. `find_similar_users` maps real-world peer groups

### Trustpilot (username)
- URL: Company/reviewer pages
- Method: Extract JSON from `__NEXT_DATA__` script tags (Next.js SSR)
- Data: reviewer username, location (country), verification status, dates, targeted businesses
- Notes: Geographic triangulation from reviewed business clustering. Anti-bot measures require headless browsing

### Yelp (username)
- URL: "Find Friends" search pathway
- Method: Search by `{FirstName} {LastInitial}` within city constraints
- Data: profile picture, reviewed establishments, elite status, geographic origin
- Notes: Session cookies required. Reveals localized movement patterns - dining, medical, nightlife

---

## Tier 6 - Web3 and Cryptographic Identity

### Ethereum Name Service / ENS (username.eth)
- URL: `GET https://metadata.ens.domains/mainnet/{contractAddress}/{tokenId}`
- Data: owner wallet address, avatar, text records (GitHub, Twitter, email)
- Notes: Gas fees mean ENS records are deliberate and curated. Permanently links email/social to financial ledger address

### OpenSea (address/username)
- URL: `GET https://api.opensea.io/api/v2/accounts/{address_or_username}`
- Auth: `X-API-KEY` header
- Data: bio, profile image, connected social media (Twitter/Instagram), NFT holdings
- Notes: Forces users to expose Twitter/Instagram to market assets, eliminating blockchain anonymity

---

## Research Notes

### Password Reset Telemetry
Platforms like Amazon, Apple, Twitter, Microsoft leak partial contact info during password reset (masked phone numbers like `**73`, masked emails like `j****@h******.com`). Cross-reference with breach data to validate. Requires headless browsing and CAPTCHA bypass - complex but highest fidelity linkage.

### Breach Enrichment
Commercial APIs (Clearbit, Hunter.io) map emails to corporate identities and LinkedIn profiles. Free tiers available for targeted, high-value resolution when enumeration fails.
