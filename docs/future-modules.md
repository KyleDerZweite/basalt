# Future Module Ideas

## Tier 2 — OG Scraping Pattern (no auth)

### Spotify (username)
- URL: `https://open.spotify.com/user/{username}`
- Data: display name, avatar, public playlists, follower count
- Method: OG metadata scraping, 404 for missing users

### Medium (username)
- URL: `https://medium.com/@{username}`
- Data: display name, bio, avatar, follower count
- Method: OG metadata scraping

### Letterboxd (username)
- URL: `https://letterboxd.com/{username}/`
- Data: display name, avatar, bio, film stats, linked URLs
- Method: OG metadata + page scraping, clean 404 for missing

### Telegram (username)
- URL: `https://t.me/{username}`
- Data: display name, avatar, bio
- Method: OG scraping, distinguishable 200 vs redirect for existence

### Mastodon / Fediverse (username)
- URL: `https://{instance}/.well-known/webfinger?resource=acct:{user}@{instance}`
- Data: bio, linked accounts, avatar, post count
- Method: WebFinger standard + profile scrape
- Notes: need to check multiple popular instances (mastodon.social, hachyderm.io, fosstodon.org, etc.)

### Pinterest (username)
- URL: `https://www.pinterest.com/{username}/`
- Data: display name, avatar, bio, follower count
- Method: OG metadata scraping

### Patreon (username)
- URL: `https://www.patreon.com/{username}`
- Data: creator name, bio, social links
- Method: OG metadata scraping

### npm (username)
- URL: `https://www.npmjs.com/~{username}`
- Data: published packages, linked GitHub
- Method: page scraping, 404 for missing

### PyPI (username)
- URL: `https://pypi.org/user/{username}/`
- Data: published packages, linked GitHub/homepage
- Method: page scraping, 404 for missing

### Lichess (username)
- URL: `GET https://lichess.org/api/user/{username}`
- Data: rating, bio, linked accounts, play stats
- Method: JSON API, no auth, 404 for missing

## Tier 3 — Requires API Key (via ~/.basalt/config)

### Have I Been Pwned (email)
- URL: `GET https://haveibeenpwned.com/api/v3/breachedaccount/{email}`
- Header: `hibp-api-key: {key}`
- Data: breach names, dates, data types leaked (passwords, emails, IPs, etc.)
- Config key: `HIBP_API_KEY`
- Notes: $3.50/month, extremely high OSINT value for email lookups
- Rate limit: 10 req/min

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
- Header: `Key: {key}` (optional, higher limits)
- Data: reputation score, breach count, profiles detected, domain age, deliverability
- Config key: `EMAILREP_API_KEY`
- Free without key: 100 req/day

### AlienVault OTX (domain)
- URL: `GET https://otx.alienvault.com/api/v1/indicators/domain/{domain}/general`
- Header: `X-OTX-API-KEY: {key}`
- Data: passive DNS, threat intelligence pulses, URL history, malware associations
- Config key: `OTX_API_KEY`
- Free: 10k req/hour

### URLScan.io (domain)
- URL: `GET https://urlscan.io/api/v1/search/?q=domain:{domain}`
- Data: technologies detected, third-party services, trackers, HTTP transactions
- Config key: `URLSCAN_API_KEY` (only needed for submitting new scans)
- Free: search unlimited without auth, ~100 scans/day with key

### HackerTarget (domain)
- URL: `GET https://api.hackertarget.com/reverseiplookup/?q={domain}`
- Data: reverse IP (shared hosting), HTTP headers, subdomains
- No auth for free tier, API key for higher limits
- Config key: `HACKERTARGET_API_KEY`
- Free: 100 req/day
