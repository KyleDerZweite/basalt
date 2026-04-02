# Future Module Ideas

## Implemented

The following were previously listed here and are now implemented:
Spotify, Medium, Telegram, Lichess candidates moved to Tier 1 below.

## Tier 1 — Zero Auth JSON APIs (high signal, easy to implement)

### Lichess (username)
- URL: `GET https://lichess.org/api/user/{username}`
- Detection: 200 (JSON) / 404
- Data: rating (blitz/classical/bullet), games played, bio, followers, real name if public
- Notes: Complements Chess.com; large chess community

### Codeforces (username)
- URL: `GET https://codeforces.com/api/user.info?handles={username}`
- Detection: 200 with `"status":"OK"` / `"status":"FAILED"`
- Data: rating, rank, contribution, max rating, last online, avatar, organization
- Rate limit: 5 req/sec per IP

### Speedrun.com (username)
- URL: `GET https://www.speedrun.com/api/v1/users?search={username}`
- Detection: 200 with populated `data` array / empty array
- Data: country, signup date, **linked social accounts (Twitch, YouTube, Twitter, SpeedRunsLive)**
- Notes: Cross-platform links are pivotable — high value

### Bluesky (username)
- URL: `GET https://public.api.bsky.app/xrpc/app.bsky.actor.getProfile?actor={username}`
- Detection: 200 (JSON) / error
- Data: display name, description, avatar, follower/following count, post count
- Notes: Growing fast, tech-forward user base

### Minecraft / Mojang (username)
- URL: `GET https://api.mojang.com/users/profiles/minecraft/{username}`
- Detection: 200 (JSON with id + name) / 404
- Data: UUID, current username, name history (via separate endpoint)
- Rate limit: 200 req/2 min per IP
- Notes: Huge gaming audience; UUID is permanent even if name changes

### Codeberg (username)
- URL: `GET https://codeberg.org/api/v1/users/{username}`
- Detection: 200 (JSON) / 404
- Data: avatar, location, bio, repository count, website
- Notes: Forgejo/Gitea API; privacy-focused Git hosting alternative

### ArtStation (username)
- URL: `GET https://www.artstation.com/users/{username}/profile.json`
- Detection: 200 (JSON) / 404
- Data: display name, bio, avatar, follower count, portfolio stats
- Notes: Professional digital artists and game developers

### Letterboxd (username)
- URL: `https://letterboxd.com/{username}/`
- Data: display name, avatar, bio, film stats, linked URLs
- Method: OG metadata scraping, clean 404 for missing

### Mastodon / Fediverse (username)
- URL: `GET https://{instance}/.well-known/webfinger?resource=acct:{user}@{instance}`
- Detection: 200 (JSON) / 404
- Data: profile URL, ActivityPub actor, linked accounts
- Notes: Check popular instances (mastodon.social, hachyderm.io, fosstodon.org); WebFinger is the standard discovery protocol

## Tier 2 — Zero Auth but Scraping or XML

### BoardGameGeek (username)
- URL: `GET https://boardgamegeek.com/xmlapi2/user?name={username}`
- Detection: XML with user data / empty response
- Data: avatar, collection size, plays, buddies, guilds
- Notes: XML format (not JSON); niche board game community

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

## Tier 3 — Infrastructure & Email Enrichment (unique data sources)

### GitHub Commit Email Harvesting (username)
- URL: `GET https://api.github.com/users/{username}/events/public`
- Data: Real email addresses from commit author metadata (even if hidden in profile)
- Notes: Upgrade existing GitHub module; extremely valuable pivot source

### PGP Keyservers (email)
- URLs:
  - `GET https://keys.openpgp.org/vks/v1/by-email/{email}`
  - `GET https://keyserver.ubuntu.com/pks/lookup?op=index&search={email}`
- Data: PGP fingerprints, key UIDs (email + name), creation dates
- Notes: Rich identity info; rarely checked by OSINT tools

### npm Registry (username/email)
- URL: `GET https://registry.npmjs.org/-/v1/search?text=author:{username}`
- Data: Published packages, maintainer emails, repository URLs
- Notes: Developer email discovery from package metadata

### PyPI (username)
- URL: `https://pypi.org/user/{username}/`
- Data: published packages, linked GitHub/homepage
- Method: page scraping, 404 for missing

### Wayback CDX Search (domain)
- URL: `GET https://web.archive.org/cdx/search/cdx?url={domain}&output=json&fl=timestamp,original,statuscode`
- Data: Full historical URL captures (not just availability check)
- Notes: Upgrade existing Wayback module; reveals old pages, contact info, email addresses

### DNS TXT Record Parsing (domain)
- Method: Direct DNS lookup for TXT records
- Data: google-site-verification, facebook-domain-verification tokens; SPF reveals email providers
- Notes: Upgrade existing DNS module; verification tokens prove domain ownership

## Tier 4 — Requires API Key (via ~/.basalt/config)

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
- Data: reverse IP, HTTP headers, subdomains
- Config key: `HACKERTARGET_API_KEY`
- Free: 100 req/day

### WakaTime (username)
- URL: `GET https://wakatime.com/share/@{username}/stats.json`
- Data: total coding time, languages, projects, editors, operating systems
- Notes: Only works for public profiles
