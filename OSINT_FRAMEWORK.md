# OSINT Framework Research for Basalt

Research date: 2026-04-04.

Sources:
- https://osintframework.com/
- https://github.com/lockfale/OSINT-Framework
- https://raw.githubusercontent.com/lockfale/OSINT-Framework/master/public/arf.json

## Goal

This file captures OSINT Framework tools that look relevant to Basalt as a consent-first, graph-based Go OSINT CLI. It is not a mirror of the whole framework. The cut line is: tools that can plausibly map to Basalt modules, enrich Basalt nodes, or inform adjacent future expansion.

## Selection Rules

- Included: username, email, domain, cloud, IP, social/community, archive, and people-search categories that can enrich identity, account, domain, or infrastructure graphs.
- Excluded: training material, malware triage, blockchain, generic OPSEC resources, active exploit tooling, and other framework areas outside Basalt's current product direction.
- Current Basalt overlap is noted separately so the backlog focuses on gaps, not modules that already exist.
- `Fit` means: `Direct` = good candidate for a first-class Basalt module now, `Context` = useful enrichment around existing seeds, `Reference` = better used as a data source, wrapper, or design reference than as a native module, `Future` = valuable but needs new seed types or a wider evidence model.

## Existing Overlap

Basalt already overlaps with these framework areas or tools at a meaningful level: Keybase, GitHub, GitLab, Codeberg, Docker Hub, DEV.to, Hacker News, Reddit, Discord, Instagram, TikTok, Telegram, Steam, WHOIS/RDAP, DNS/CT, Shodan, Wayback Machine, IPinfo.

## Highest-Value Candidates

- WhatsMyName Web: Fast username coverage with a passive web front end.
- Namechk: Good cheap signal for username and domain availability pivots.
- Hunter: High-value email discovery with API support and company/domain pivots.
- Epieos Email Tool: Useful reverse-email enrichment and social-profile correlation.
- Have I been pwned?: Strong breach signal that fits Basalt's email workflows.
- Hudson Rock: Adds infostealer exposure context for email, username, domain, and IP.
- GHunt (T): Reference implementation for deeper Google-account style enrichment.
- Holehe (T): Reference implementation for service enumeration from an email.
- ViewDNS.info: Straightforward domain/IP enrichment that maps cleanly to graph nodes.
- Netlas.io: Multi-surface domain, DNS, and exposure data with API support.
- urlscan.io: Strong page, redirect, and infrastructure context once a domain is known.
- BuiltWith: Useful website stack and related-property enrichment from domains.
- Wappalyzer: Technology fingerprinting for websites and hosted properties.
- Censys: Certificate, host, and exposure pivots around domains and IPs.
- crt.sh - Certificate Search: Easy certificate-transparency pivot source for domain expansion.
- DNS Dumpster: Fast passive DNS and subdomain context for domain scans.
- SpyOnWeb: Good relationship pivots via analytics and ad IDs.
- Fediverse Observer: Missing social surface with structured instance/account context.
- Threads Dashboard: A clean path into Threads/Meta public-content discovery.
- TGStat: Telegram discovery surface with API support.
- PeekYou: Candidate anchor for future `person_name` and broader people-search support.
- Common Crawl: Good archive source for historical domain/web evidence extraction.

## Inventory Summary

- Account and username discovery: 17 tools
- Email discovery and enrichment: 29 tools
- Domain and website footprinting: 95 tools
- Cloud and infrastructure context: 58 tools
- Missing social/community platforms: 47 tools
- Future seed types: 21 tools
- Community and archive search: 31 tools
- Total candidates in this file: 298 tools

## Account and username discovery

| Tool | Framework Path | Input | Fit | Flags | Why It Matters |
| --- | --- | --- | --- | --- | --- |
| Amazon Usernames (M) | Username / Specific Sites | Username (inserted into Google search query) | Reference | dork, manual-url | Finding Amazon public profiles, wishlists, and review activity by username |
| Github User (M) | Username / Specific Sites | GitHub username (inserted into URL path) | Reference | api, manual-url | Enumerating a GitHub user's recent public activity and repository interactions |
| MIT PGP Key Server | Username / Specific Sites | Name, email address, or key ID | Direct | api | Looking up PGP public keys associated with a username or email address |
| ProtonMail Domains (M) | Username / Specific Sites | Full email address (any domain that may be hosted on ProtonMail) | Reference | api, manual-url | Checking if an email address on a custom domain is hosted on ProtonMail |
| ProtonMail users (M) | Username / Specific Sites | ProtonMail username (appended with @protonmail.com) | Reference | api, manual-url | Confirming whether a ProtonMail username exists and retrieving its PGP public key |
| Tinder Usernames (M) | Username / Specific Sites | Tinder username (appended to URL after @) | Reference | manual-url | Confirming existence of a Tinder profile and viewing public profile details |
| FootprintIQ | Username / Username Search Engines | Username, email address, or phone number | Direct | freemium | Username and email footprint scanning with breach and data broker exposure checks |
| GitFive (T) | Username / Username Search Engines | GitHub username or email address | Reference | local, reg | Deep investigation of GitHub user profiles and email-to-account mapping |
| Lullar | Username / Username Search Engines | Email address, full name, or username | Direct | - | Social media profile discovery by username, email, or name |
| NameCheckup | Username / Username Search Engines | Username or domain name | Direct | api | Username and domain availability checking with WHOIS info |
| Namechk | Username / Username Search Engines | Username or domain name | Direct | - | Quick username availability check across social media and domains |
| Names Directory | Username / Username Search Engines | First name or surname | Direct | - | Finding name combinations and frequency data for a given first or last name |
| Sherlock (T) | Username / Username Search Engines | Username(s) | Reference | local | Mass username enumeration across 400+ sites |
| Sylva Identity Discovery (T) | Username / Username Search Engines | Username | Reference | api, local | Username enumeration with identity branching |
| Thats Them | Username / Username Search Engines | Name, email address, phone number, or physical address | Direct | freemium | People search by name, email, phone, or address |
| WhatsMyName (T) | Username / Username Search Engines | Username | Reference | local | Username enumeration using community-maintained site detection data |
| WhatsMyName Web | Username / Username Search Engines | Username | Direct | - | Quick web-based username enumeration across social media, forums, gaming platforms, and professional networks |

## Email discovery and enrichment

| Tool | Framework Path | Input | Fit | Flags | Why It Matters |
| --- | --- | --- | --- | --- | --- |
| DeHashed (R) | Email Address / Breach Data | Email, username, password, domain, phone, or IP | Direct | api, reg, freemium | Breach searching, credential lookup, historical breach analysis |
| Have I been pwned? | Email Address / Breach Data | Email address, phone number, password hash | Direct | api, reg, freemium | Breach detection, credential exposure checks |
| Hudson Rock | Email Address / Breach Data | Email address, domain, username, or IP | Direct | api, freemium | Infostealer detection, breach assessment, device compromise verification |
| Vigilante.pw | Email Address / Breach Data | Email, username, domain | Direct | - | Breach research, public breach database navigation |
| Email Format | Email Address / Common Email Formats | Sample email addresses or company info | Reference | manual-url | Corporate email pattern analysis, email format discovery |
| Email Permutator | Email Address / Common Email Formats | Person name, nickname, domain(s) | Direct | - | Email pattern generation, targeted email guessing |
| breach.vip | Email Address / Email Search | Email, domain, Discord ID, or phone number | Direct | - | Breach database search, credential lookup |
| Email to Address (R) | Email Address / Email Search | Email addresses, contact data | Direct | api, reg, paid | Email validation, address enrichment |
| Epieos Email Tool | Email Address / Email Search | Email address or phone number | Direct | freemium | Email reverse lookup, social media profile discovery |
| GHunt (T) | Email Address / Email Search | Gmail address or GAIA ID | Reference | local | Google account investigation, YouTube/Google Photos OSINT |
| Holehe (T) | Email Address / Email Search | Email address | Reference | local | Email account enumeration, service detection |
| Hunter | Email Address / Email Search | Domain name, person name, or company info | Direct | api, freemium | Business email discovery, email verification |
| Infoga (T) | Email Address / Email Search | Email address | Reference | local, dork | Early-stage email reconnaissance, information gathering |
| OSINT Industries | Email Address / Email Search | Email address, phone number, username, or crypto wallet | Direct | freemium | Account enumeration, breach detection, digital footprint mapping |
| Skymem | Email Address / Email Search | Domain name or person name + domain | Direct | freemium | Email discovery by domain, bulk email list creation |
| Sylva Identity Discovery (T) | Email Address / Email Search | Email, username, or PGP fingerprint | Reference | local | Identity correlation via GitHub and PGP |
| ThatsThem | Email Address / Email Search | Email address | Direct | freemium | Reverse email lookup, person identification |
| theHarvester (T) | Email Address / Email Search | Domain name | Reference | local | Email harvesting, subdomain enumeration, passive recon |
| VoilaNorbert | Email Address / Email Search | Domain, name, or LinkedIn URL | Direct | api, freemium | Business email discovery, bulk email finding |
| Burner Email Providers (T) | Email Address / Email Verification | Email domain | Reference | local | Identifying burner email providers for integration into custom investigation tools |
| Disposable Email Domains (T) | Email Address / Email Verification | Domain name to check against the blocklist | Reference | local | Detecting disposable and temporary email addresses during verification |
| Disposable Emails Registry | Email Address / Email Verification | Domain name or bulk list download | Direct | - | Bulk blocking and threat intelligence integration for disposable email detection |
| Email Reputation | Email Address / Email Verification | Email address | Direct | api | Email reputation checking, risk assessment |
| MailboxValidator | Email Address / Email Verification | Email addresses or bulk lists | Direct | api, reg, paid | Email validation, list cleaning, bounce prevention |
| MailScrap | Email Address / Email Verification | Email addresses or email lists | Direct | freemium | Email validation, list cleaning, disposable email detection |
| Reacher Demo | Email Address / Email Verification | Email address | Direct | - | Email verification testing, demonstration |
| Reacher Github (T) | Email Address / Email Verification | Email address | Reference | api, local | Email verification, bounce detection, list cleaning |
| VerifyEmail (R$) | Email Address / Email Verification | - | Direct | reg | General OSINT lookup or enrichment source. |
| MxToolbox | Email Address / Mail Blacklists | Domain name or email address | Direct | - | Email server diagnostics, deliverability testing, DNS validation |

## Domain and website footprinting

| Tool | Framework Path | Input | Fit | Flags | Why It Matters |
| --- | --- | --- | --- | --- | --- |
| Alexa Site Statistics | Domain Name / Analytics | Domain | Reference | manual-url, down | Historical reference only |
| BuiltWith | Domain Name / Analytics | Website URL or domain | Direct | api, freemium | Web technology intelligence and competitive analysis |
| Cisco Umbrella Popularity List | Domain Name / Analytics | Domain or list lookup | Direct | api | Top-domain popularity and DNS trend context |
| ClearWebStats.com | Domain Name / Analytics | Domain | Direct | - | Lightweight web popularity lookups |
| Ewhois | Domain Name / Analytics | Domain | Direct | - | Quick WHOIS lookups |
| Keyword Density | Domain Name / Analytics | URL or text | Direct | - | On-page keyword frequency analysis |
| Moonsearch | Domain Name / Analytics | Domain or registrant details | Reference | manual-url, degraded | Historical domain ownership correlation |
| Open Site Explorer | Domain Name / Analytics | Domain or URL | Reference | api, reg, manual-url, freemium, degraded | Backlink and linking-domain analysis |
| PubDB | Domain Name / Analytics | Query terms | Reference | manual-url, down | Historical reference only |
| SEMrush | Domain Name / Analytics | Domain | Direct | api, reg, paid | Traffic and backlink competitive analysis |
| SimilarWeb | Domain Name / Analytics | Domain | Direct | api, reg, freemium | Competitor traffic and referral analysis |
| Sitedossier | Domain Name / Analytics | Domain or IP | Direct | - | Quick domain intelligence aggregation |
| Siteliner | Domain Name / Analytics | Domain | Direct | reg, freemium | Duplicate-content and link-health audits |
| SiteSleuth | Domain Name / Analytics | Domain, Google Analytics ID, AdSense ID, or Stripe key | Direct | - | Tracking code intelligence and related domain discovery |
| SpyOnWeb | Domain Name / Analytics | Domain or analytics/ad IDs | Direct | api, reg, freemium | Finding related infrastructure via shared IDs |
| StatsCrop | Domain Name / Analytics | Domain | Direct | - | Quick website popularity snapshots |
| Visual Site Mapper | Domain Name / Analytics | Domain or URL seed | Reference | local | Generating website structure maps |
| Wappalyzer (T) | Domain Name / Analytics | Domain or URL | Reference | api, local, reg, freemium | Technology stack fingerprinting and recon |
| WhatWeb | Domain Name / Analytics | Domain or URL | Reference | local | CLI-based web technology fingerprinting |
| Censys | Domain Name / Certificate Search | Domain, IP, certificate fingerprint, search query | Direct | api, reg, freemium | Certificate discovery, host enumeration, exposure monitoring |
| certgraph (T) | Domain Name / Certificate Search | Hostname or domain name | Reference | local | Certificate mapping, domain relationship discovery, hostname enumeration via SSL certificates |
| CertKit - Certificate Transparency Log Search | Domain Name / Certificate Search | Domain name | Direct | api | CT certificate search, subdomain enumeration, certificate misuse detection |
| crt.sh - Certificate Search | Domain Name / Certificate Search | Domain name (with or without wildcard) | Reference | api, manual-url | Certificate search, subdomain discovery via CT logs, detecting unauthorized certificates |
| Google's Certificate Transparency | Domain Name / Certificate Search | Domain name or certificate fingerprint | Direct | api | Certificate discovery, unauthorized cert detection, domain monitoring |
| Netlas.io | Domain Name / Certificate Search | Domain name, IP address, ASN, DNS records | Direct | api, freemium | Internet reconnaissance, DNS and WHOIS lookups, attack surface discovery, vulnerability research |
| Spyse | Domain Name / Certificate Search | Domain, IP, certificate, email, or organization name | Direct | api, reg, freemium | Domain intelligence, certificate discovery, subdomain enumeration, vulnerability identification |
| Change Detection | Domain Name / Change Detection | URL and monitoring rules | Reference | api, local | Self-hosted page change monitoring |
| Follow That Page | Domain Name / Change Detection | Target page URL and optional keyword filters | Direct | reg, freemium | Tracking updates on specific web pages by keyword |
| UPcheck | Domain Name / Change Detection | URL/domain | Direct | down | Quick site availability checks |
| Urlwatch | Domain Name / Change Detection | URLs, feeds, and local watch configuration | Reference | local, manual-url | Self-hosted web page change monitoring automation |
| VisualPing | Domain Name / Change Detection | URL and watch settings | Direct | api, reg, freemium | Automated webpage change monitoring |
| WatchThatPage | Domain Name / Change Detection | Web page URL and watch configuration | Direct | reg, freemium | Monitoring static web pages for updates over time |
| DNSSEC Analyzer | Domain Name / DNSSEC | Domain names | Direct | - | DNSSEC chain-of-trust validation |
| DNSViz | Domain Name / DNSSEC | Domain name | Direct | - | Visual DNSSEC validation and DNS misconfiguration analysis |
| AnalyzeID | Domain Name / Discovery | Tracking ID (analytics, ads, affiliate, or publisher ID) | Reference | manual-url | Pivoting from shared tracking IDs to related domains |
| BuiltWith | Domain Name / Discovery | Domain or URL | Direct | api, freemium | Website technology stack fingerprinting and ecosystem mapping |
| Criminal IP Search | Domain Name / Discovery | IP, domain, ASN, CVE, or filter-based threat query | Direct | api, reg, freemium | Threat-focused lookup of internet-facing assets and exposures |
| Daily DNS Changes | Domain Name / Discovery | Domain name | Direct | freemium | DNS change detection, subdomain discovery, infrastructure monitoring |
| Kraken (T) | Domain Name / Discovery | Domain, host, or target parameters supported by selected module | Reference | local | CLI-driven reconnaissance against domain and host assets |
| Netlas.io | Domain Name / Discovery | Domain name, IP address, ASN, DNS records | Direct | api, freemium | Internet reconnaissance, DNS and WHOIS lookups, attack surface discovery, vulnerability research |
| Redirect Detective | Domain Name / Discovery | URL | Reference | manual-url | Understanding redirect paths and affiliate or cloaking behavior |
| Sitediff (T) | Domain Name / Discovery | Two URLs or snapshots to compare | Reference | local, manual-url | Tracking site changes between snapshots for monitoring and QA |
| urlDNA | Domain Name / Discovery | URL or domain | Reference | reg, manual-url, freemium | Quick URL/domain triage and intelligence pivoting |
| urlscan.io | Domain Name / Discovery | URL or domain | Reference | api, reg, manual-url, freemium | Investigating suspicious URLs with scan snapshots and indicators |
| Wappalyzer | Domain Name / Discovery | Domain, URL, or browsed webpage | Direct | api, reg, freemium | Detecting web technologies and software dependencies at scale |
| ZoomEye.ai | Domain Name / Discovery | Domain, IP, port, service, or natural language query | Reference | api, reg, dork, freemium | Internet device discovery, service enumeration, vulnerability mapping, attack surface assessment |
| Deteque (R) | Domain Name / PassiveDNS | Domain, IP, URL, file hash, or AS number | Direct | api, reg, freemium | Domain/IP threat intelligence, malware tracking, botnet detection, abuse data |
| DNS Dumpster | Domain Name / PassiveDNS | Domain name | Direct | - | Subdomain enumeration, DNS reconnaissance, infrastructure mapping |
| Mnemonic | Domain Name / PassiveDNS | Domain or IP address | Direct | api | Passive DNS lookups, historical domain resolutions, DNS reconnaissance |
| AltDNS (T) | Domain Name / Subdomains | Known subdomains, wordlist, and target domain | Reference | local | Discovering likely subdomain variants through permutations |
| Aquatone (T) | Domain Name / Subdomains | Domain name | Reference | local | Visual subdomain reconnaissance, HTTP service discovery, attack surface mapping |
| Bluto (T) | Domain Name / Subdomains | Target domain and optional scan switches | Reference | local | Initial domain footprinting and asset discovery |
| DNS Recon (T) | Domain Name / Subdomains | Domain name, IP range/CIDR, subdomain wordlist, DNS server address | Reference | local | DNS enumeration, zone transfer testing, subdomain brute forcing, DNS security assessment |
| dnspop (T) | Domain Name / Subdomains | Domain and optional scan parameters | Reference | local | Command-line DNS recon and record analysis |
| Fierce Domain Scanner (T) | Domain Name / Subdomains | Domain, DNS server options, and optional wordlist/range parameters | Reference | local | DNS recon and subdomain-to-IP mapping |
| FindSubDomains | Domain Name / Subdomains | Domain name or keyword | Direct | - | Automated subdomain enumeration, organization name filtering, subdomain statistics |
| gdns (T) | Domain Name / Subdomains | Domain and query options | Reference | local | Quick DNS enumeration via Google DNS services |
| Gobuster (T) | Domain Name / Subdomains | Domain, wordlist, and optional resolver/thread settings | Reference | local | Fast DNS and vhost brute-force enumeration |
| Google Subdomains (D) | Domain Name / Subdomains | Domain name (as Google Dork syntax: site:domain.com) | Reference | dork, manual-url | Indexed subdomain discovery, publicly visible subdomain enumeration |
| Netlas.io | Domain Name / Subdomains | Domain name, IP address, ASN, DNS records | Direct | api, freemium | Internet reconnaissance, DNS and WHOIS lookups, attack surface discovery, vulnerability research |
| OWASP Maryam (T) | Domain Name / Subdomains | Domain, IP, email, username, or module-specific query terms | Reference | local | Scriptable multi-module OSINT reconnaissance workflows |
| Pentest-tools.com Subdomains | Domain Name / Subdomains | Domain name | Direct | reg, freemium | Quick browser-based subdomain discovery without local setup |
| Recon-ng (T) | Domain Name / Subdomains | Domain, company name, email, IP | Reference | api, local | Modular web recon, API-driven data collection |
| SecLists DNS Subdomains (T) | Domain Name / Subdomains | Domain and chosen wordlist file used in external tooling | Reference | local | Supplying high-quality DNS wordlists for enumeration tools |
| Sublist3r | Domain Name / Subdomains | Domain and optional brute-force/thread settings | Reference | local | Combining passive and active subdomain discovery in one tool |
| SynapsInt | Domain Name / Subdomains | Domain, IP, email, phone, username, CVE ID | Direct | - | Unified OSINT research, subdomain discovery, multi-vector intelligence gathering |
| theHarvester (T) | Domain Name / Subdomains | Domain and selected data source(s) | Reference | local | Passive email and subdomain collection from indexed sources |
| XRay | Domain Name / Subdomains | Domain name, subdomain wordlist, Shodan API key (optional), ViewDNS API key (optional) | Reference | local | Automated subdomain discovery with banner grabbing, open port enumeration, Shodan integration |
| Catphish (T) | Domain Name / Typosquatting | Target domain | Reference | local | Red team phishing domain generation |
| DNS Twist (T) | Domain Name / Typosquatting | Domain name | Reference | local | Typosquatting and phishing domain detection |
| dnstwister | Domain Name / Typosquatting | Domain name | Direct | freemium | Typosquatting monitoring |
| URLCrazy (T) | Domain Name / Typosquatting | Domain name | Reference | local | Typosquatting domain discovery |
| CheckShortURL | Domain Name / URL Expanders | Shortened URL | Direct | - | Safe short-link destination checks |
| KnowURL | Domain Name / URL Expanders | URL | Reference | manual-url, degraded | Historical reference only |
| Link Expander | Domain Name / URL Expanders | Shortened URL | Direct | - | Expanding shortened links safely |
| URL Expander | Domain Name / URL Expanders | Shortened URL | Direct | - | Resolving opaque short links |
| Where Does This Link Go? | Domain Name / URL Expanders | URL | Direct | - | Tracing redirect chains for suspicious links |
| Daily DNS Changes | Domain Name / Whois Records | Domain name | Direct | freemium | DNS change detection, subdomain discovery, infrastructure monitoring |
| DNSstuff | Domain Name / Whois Records | Domain name, IP address | Direct | - | Quick DNS and WHOIS lookups, network diagnostics |
| Domain Dossier | Domain Name / Whois Records | Domain name or IP address | Direct | - | Quick domain and IP reconnaissance with DNS and WHOIS data |
| Domaincrawler.com | Domain Name / Whois Records | Domain name, DNS data, technology stack filters | Direct | api, reg, paid | Large-scale domain research, brand protection monitoring, zone file analysis, market intelligence |
| domainIQ | Domain Name / Whois Records | Domain name | Direct | reg, freemium | Domain ownership history, reverse analytics lookup, competitor domain research |
| DomainTools Whois | Domain Name / Whois Records | Domain name or IP address | Direct | api, reg, paid | Historical WHOIS research, threat actor tracking, enterprise domain intelligence |
| easyWhois | Domain Name / Whois Records | Domain name | Direct | - | Quick domain WHOIS lookups and DNS checks |
| IP2WHOIS | Domain Name / Whois Records | Domain name or IP address | Direct | api | Domain and IP WHOIS lookups, registrant research |
| MarkMonitor Whois Search | Domain Name / Whois Records | Domain name | Direct | reg, paid | Corporate domain portfolio management, brand protection, trademark monitoring |
| Netlas.io | Domain Name / Whois Records | Domain name, IP address, ASN, DNS records | Direct | api, freemium | Internet reconnaissance, DNS and WHOIS lookups, attack surface discovery, vulnerability research |
| Robtex (R) | Domain Name / Whois Records | Domain name, IP address, hostname, autonomous system | Direct | reg | DNS reconnaissance, IP and domain relationship mapping, historical internet data lookup |
| SWITCH Internet Domains Whois (.ch) | Domain Name / Whois Records | .ch or .li domain name | Direct | - | .ch and .li domain ownership research, Swiss Internet infrastructure lookup |
| ViewDNS.info | Domain Name / Whois Records | Domain name, IP address, registrant name/email, nameserver | Direct | api | DNS reconnaissance, reverse IP and reverse WHOIS lookups, historical DNS tracking |
| Website Informer | Domain Name / Whois Records | Domain name or URL | Direct | - | Website profiling, ownership verification, traffic estimation, technical stack discovery |
| Who.is | Domain Name / Whois Records | Domain name or IP address | Direct | - | Domain registration research, WHOIS lookups, RDAP queries, IP tracking |
| Whois AMPed | Domain Name / Whois Records | Domain name | Direct | - | Mobile-friendly WHOIS lookups, quick domain information retrieval |
| Whois ARIN | Domain Name / Whois Records | IP address, ASN, organization name, contact information | Direct | - | IP address and ASN registration data, North American internet resource tracking |
| Whoisology | Domain Name / Whois Records | Domain name, email, registrant name | Direct | freemium | Historical domain ownership, reverse WHOIS lookups, domain connection tracking |

## Cloud and infrastructure context

| Tool | Framework Path | Input | Fit | Flags | Why It Matters |
| --- | --- | --- | --- | --- | --- |
| AWSBucketDump (T) | Cloud Infrastructure / AWS Enumeration | AWS account naming patterns, keywords, and optional wordlists | Reference | local | Targeted S3 bucket discovery and object collection |
| cloud_enum (T) | Cloud Infrastructure / AWS Enumeration | Company names, domains, and custom keywords/wordlists | Reference | local | Rapid discovery of cloud storage exposure across major providers |
| Subfinder (T) | Cloud Infrastructure / AWS Enumeration | Domain name and optional API credentials for data sources | Reference | api, local, reg | Passive subdomain enumeration for cloud asset inventorying |
| AADInternals (T) | Cloud Infrastructure / Azure/GCP Discovery | Tenant identifiers, domain names, and account context | Reference | local | Deep Azure AD reconnaissance and security assessment |
| GCPBucketBrute (T) | Cloud Infrastructure / Azure/GCP Discovery | Target company names, domains, and custom wordlists | Reference | local | Enumerating likely GCS bucket names at scale |
| MicroBurst (T) | Cloud Infrastructure / Azure/GCP Discovery | Azure tenant/subscription context and optional credentials | Reference | local | Azure subscription and service-level exposure testing |
| ROADtools (T) | Cloud Infrastructure / Azure/GCP Discovery | Azure AD tenant context and authentication tokens/credentials | Reference | api, local | Enumerating Azure AD objects and privilege relationships |
| Stormspotter (T) | Cloud Infrastructure / Azure/GCP Discovery | Azure subscription/tenant metadata collected by collectors | Reference | api, local | Visual analysis of Azure attack paths and privilege chains |
| BucketLoot (T) | Cloud Infrastructure / S3/Blob Storage | Bucket name patterns and target-related keywords | Reference | local, degraded | Supplemental bucket discovery when validating legacy workflows |
| goblob (T) | Cloud Infrastructure / S3/Blob Storage | Target naming patterns and optional custom wordlists | Reference | local | Enumerating Azure blob container exposure quickly |
| lazys3 (T) | Cloud Infrastructure / S3/Blob Storage | Base target keywords and optional custom wordlists | Reference | local | Quick permutation-based S3 bucket name discovery |
| Public Buckets | Cloud Infrastructure / S3/Blob Storage | Keywords, domains, filenames, and object metadata filters | Context | reg, freemium | Investigating exposed bucket contents without running local scanners |
| S3Scanner (T) | Cloud Infrastructure / S3/Blob Storage | Bucket names, generated candidates, or wordlist-driven targets | Reference | local | Validating bucket exposure and permissions across S3-compatible targets |
| Amass (T) | Cloud Infrastructure / SaaS Footprinting | Domain names, ASN data, CIDRs, and optional API credentials | Reference | api, local, reg | Comprehensive external attack-surface and subdomain mapping |
| dnsrecon (T) | Cloud Infrastructure / SaaS Footprinting | Domain names, name servers, and optional DNS wordlists | Reference | local | Detailed DNS reconnaissance and validation |
| SpiderFoot (T) | Cloud Infrastructure / SaaS Footprinting | Domain, IP, email, name, phone, subnet | Reference | api, local | Automated recon, attack surface mapping, threat intelligence |
| Sublist3r (T) | Cloud Infrastructure / SaaS Footprinting | Domain name | Reference | local | Quick passive subdomain discovery for reconnaissance |
| theHarvester (T) | Cloud Infrastructure / SaaS Footprinting | Domain names, company names, and selected data-source modules | Reference | api, local | Email and host discovery tied to a target organization |
| BGP Malicious Content Ranking | IP & MAC Address / BGP | ASN or prefix | Context | - | Identify malicious ASNs and networks |
| BGP Tools | IP & MAC Address / BGP | ASN, IP, or prefix | Reference | manual-url | BGP routing and AS analysis |
| Hurricane Electric BGP Toolkit | IP & MAC Address / BGP | ASN, IP range, or prefix | Context | - | BGP analysis and routing intelligence |
| PeeringDB | IP & MAC Address / BGP | ASN, organization, or IX | Context | api | Internet peering and AS relationship mapping |
| DB-IP | IP & MAC Address / Geolocation | IP address | Context | api, freemium | Accurate IP geolocation with developer API |
| Info Sniper | IP & MAC Address / Geolocation | IP, email, or phone | Context | reg, freemium | Multi-field reverse lookup (IP/email/phone) |
| IP Fingerprints | IP & MAC Address / Geolocation | IP address | Context | - | Find domains on shared hosting |
| IP Location Finder | IP & MAC Address / Geolocation | IP address | Context | - | Quick IP location with maps |
| IP2Location.com | IP & MAC Address / Geolocation | IP address | Context | api, reg, freemium | Accurate geolocation with proxy detection |
| IPv4/IPv6 lists by country code | IP & MAC Address / Geolocation | Country code | Reference | manual-url | Country-level IP enumeration |
| MaxMind Demo | IP & MAC Address / Geolocation | IP address | Context | - | Quick IP geolocation |
| utrace | IP & MAC Address / Geolocation | IP or hostname | Context | - | IP location and traceroute |
| BinaryEdge (R) | IP & MAC Address / Host / Port Discovery | IP, domain, query | Context | api, reg, paid | Commercial internet threat intelligence |
| Criminal IP Search | IP & MAC Address / Host / Port Discovery | IP address | Context | api, reg, freemium | IP reputation and malicious activity analysis |
| Internet Census Search | IP & MAC Address / Host / Port Discovery | Service type, IP range, port | Context | - | Search open services and devices |
| Masscan (T) | IP & MAC Address / Host / Port Discovery | IP range | Reference | local | Large-scale network port scanning |
| Netlas.io | IP & MAC Address / Host / Port Discovery | IP, domain, ASN | Context | api, reg, freemium | Internet asset reconnaissance with web, DNS, WHOIS |
| Nmap (T) | IP & MAC Address / Host / Port Discovery | IP range or hostname | Reference | local | Network reconnaissance and port scanning |
| Online Port scanner | IP & MAC Address / Host / Port Discovery | IP address and port range | Context | - | Quick port scanning without tools |
| Portmap | IP & MAC Address / Host / Port Discovery | IP address or hostname | Context | - | Port scanning and service discovery |
| Scanless (T) | IP & MAC Address / Host / Port Discovery | IP and port | Reference | local | Stealthy port scanning via proxies |
| Scans.io | IP & MAC Address / Host / Port Discovery | IP or domain | Context | - | Historical internet scan data access |
| Spyse | IP & MAC Address / Host / Port Discovery | IP, domain, email, organization | Context | api, reg, freemium | Internet asset discovery and reconnaissance |
| urlscan.io | IP & MAC Address / Host / Port Discovery | URL or domain | Context | api, freemium | URL/domain scanning for malware and phishing |
| ASlookup.com | IP & MAC Address / IPv4 | ASN or IP address | Reference | manual-url | BGP and ASN lookup |
| Hacker Target - Reverse DNS | IP & MAC Address / IPv4 | IP address or range | Context | api, freemium | Reverse DNS lookup of IP addresses |
| IP to ASN DB | IP & MAC Address / IPv4 | IP address | Context | api | IP to ASN lookup with historical data |
| IPv4 CIDR Report | IP & MAC Address / IPv4 | CIDR block | Context | - | CIDR block analysis and subnet enumeration |
| Onyphe | IP & MAC Address / IPv4 | IP, domain, CVE | Context | api, reg, freemium | Internet asset discovery and threat intel |
| Port scanner Online | IP & MAC Address / IPv4 | IP and port | Context | - | Quick port availability checks |
| Reverse.report | IP & MAC Address / IPv4 | IP address or domain | Context | reg, freemium | Reverse IP and domain lookups |
| Team Cymru IP to ASN | IP & MAC Address / IPv4 | IP address | Context | api | IP to ASN mapping |
| Bing IP Search (D) | IP & MAC Address / Neighbor Domains | IP address | Reference | dork | Find domains on IP using Bing index |
| IP Fingerprints - Reverse IP Lookup | IP & MAC Address / Neighbor Domains | IP address | Context | - | Find domains on shared hosting |
| MyIPNeighbors | IP & MAC Address / Neighbor Domains | IP address | Context | - | Find all domains on same shared IP |
| TCP/IP Utils - Domain Neighbors | IP & MAC Address / Neighbor Domains | Domain or IP | Context | - | Identify related domains on same IP |
| CloudFail (T) | IP & MAC Address / Protected by Cloud Services | Domain protected by Cloudflare | Reference | local | Bypass Cloudflare to find origin IP |
| CloudFlare Watch | IP & MAC Address / Protected by Cloud Services | Domain or IP | Context | - | Identify Cloudflare-protected sites |
| OpenCellid: Database of Cell Towers | IP & MAC Address / Wireless Network Info | Cell tower ID or location | Future | api | Find cellular tower locations and coverage |
| WiGLE: Wireless Network Mapping | IP & MAC Address / Wireless Network Info | Location, SSID, or BSSID | Future | api, reg, freemium | Map wireless networks and find signal coverage |

## Missing social/community platforms

| Tool | Framework Path | Input | Fit | Flags | Why It Matters |
| --- | --- | --- | --- | --- | --- |
| Disboard | Instant Messaging / Discord | Keyword, category, or tag searches | Direct | - | Discovering public Discord communities and server metadata |
| DiscordOSINT (T) | Instant Messaging / Discord | Manual review of documentation and linked resources | Reference | local | Learning Discord investigation methods and toolchains |
| OSINT Industries | Instant Messaging / Signal / Phone Lookup | Email addresses, usernames, phone numbers, and account identifiers | Future | reg, paid | Enterprise-grade identity enrichment and account correlation |
| slack-intelbot (T) | Instant Messaging / Slack | Indicators posted in Slack and configured API credentials | Reference | api, local, reg | In-channel IOC enrichment for threat intelligence triage |
| slack-web-scraper (T) | Instant Messaging / Slack | Slack-authenticated browser/session context | Reference | local, reg | Archiving Slack channel content for offline analysis |
| SlackPirate (T) | Instant Messaging / Slack | Authenticated Slack session/token and workspace target | Reference | api, local, reg | Slack workspace enumeration and sensitive data exposure assessment |
| Google CSE for Telegram links | Instant Messaging / Telegram | Keyword search terms | Reference | manual-url | Finding public Telegram channels and groups by keyword |
| Telegago (T) | Instant Messaging / Telegram | Keyword search terms | Reference | local | Keyword discovery across publicly indexed Telegram content |
| Telegram-OSINT (T) | Instant Messaging / Telegram | Manual review of listed tools and references | Reference | local | Sourcing Telegram-specific tooling and investigative playbooks |
| TGStat | Instant Messaging / Telegram | Channel names, keywords, and Telegram entity identifiers | Direct | api, reg, freemium | Telegram channel trend analysis and engagement benchmarking |
| Tosint (T) | Instant Messaging / Telegram | Telegram bot usernames, links, or identifiers | Reference | api, local | Telegram bot reconnaissance and metadata extraction |
| line-message-analyzer (T) | Instant Messaging / WeChat / LINE | LINE exported chat history files | Reference | local | LINE conversation frequency and behavior analysis |
| linelog2py (T) | Instant Messaging / WeChat / LINE | Exported LINE chat history files | Reference | local | Converting LINE chat exports for downstream analysis workflows |
| Sogou WeChat Search | Instant Messaging / WeChat / LINE | Chinese keywords, account names, or article titles | Direct | - | Discovering public WeChat posts and organization presence |
| wechat-dump (T) | Instant Messaging / WeChat / LINE | Rooted Android device data and WeChat app storage | Reference | local | Authorized mobile WeChat chat history extraction and preservation |
| wechat-text-backup (T) | Instant Messaging / WeChat / LINE | Local WeChat database files and decryption context | Reference | local | Decrypting and backing up local WeChat message archives |
| WechatSogou (T) | Instant Messaging / WeChat / LINE | Search keywords and query parameters | Reference | local | Automating batch WeChat article and account discovery |
| Email2WhatsApp (T) | Instant Messaging / WhatsApp | Email address targets | Reference | local | Email-to-WhatsApp account correlation during profiling |
| WhatsApp-OSINT (T) | Instant Messaging / WhatsApp | Phone numbers or WhatsApp account identifiers | Reference | api, local, reg, freemium | Rapid WhatsApp account reconnaissance and metadata checks |
| Treeverse (T) | Social Networks / Bluesky | Post or thread URLs | Reference | local | Conversation structure mapping |
| Fedifinder | Social Networks / Fediverse/Mastodon | Twitter/X account (via OAuth login) | Direct | reg, down | Finding Twitter contacts who moved to Mastodon/Fediverse |
| Fediverse Observer | Social Networks / Fediverse/Mastodon | Search filters (software type, country, language, instance name) | Direct | api | Discovering and mapping Fediverse instances by software, country, or size |
| Fediverse_OSINT (T) | Social Networks / Fediverse/Mastodon | Username or search terms | Reference | local, degraded | Cross-instance Fediverse user and content search |
| Masto (T) | Social Networks / Fediverse/Mastodon | Mastodon username and instance (e.g., user@mastodon.social) | Reference | api, local | Mastodon user profile investigation and account analysis |
| InSpy (T) | Social Networks / LinkedIn | Company name and domain context | Reference | api, local, reg | Employee and email pattern discovery |
| LinkedInt - LinkedIn Recon Tool (T) | Social Networks / LinkedIn | Company names, LinkedIn URLs, and search targets | Reference | local, reg, degraded | LinkedIn employee enumeration |
| raven (T) | Social Networks / LinkedIn | Company, role, and location filters | Reference | local, reg | Automated LinkedIn org mapping |
| ScrapedIn (T) | Social Networks / LinkedIn | LinkedIn search queries and profile targets | Reference | local, reg | LinkedIn profile data extraction |
| Asian Avenue | Social Networks / Other Social Networks | Username or profile search | Direct | defunct | Searching archived Asian Avenue profile data |
| Ask FM | Social Networks / Other Social Networks | Ask FM username | Reference | manual-url | Finding Ask FM profiles by username |
| BlackPlanet.com - Member Find | Social Networks / Other Social Networks | Username, email, or profile search criteria | Direct | - | Finding BlackPlanet user profiles by username or criteria |
| Delicious | Social Networks / Other Social Networks | Username or URL/tag search | Direct | api | Finding user bookmarking profiles and discovering curated link collections |
| MiGente (Latino) | Social Networks / Other Social Networks | Username or email search | Direct | reg, degraded | Searching archived MiGente profiles |
| Myspace | Social Networks / Other Social Networks | Username or artist name | Direct | - | Searching for archived Myspace profiles and historical social media data |
| Odnoklassniki | Social Networks / Other Social Networks | Username or search criteria | Direct | - | Finding Russian-speaking users and investigating Odnoklassniki profiles |
| Orkut (Brazil) | Social Networks / Other Social Networks | Username or profile ID (via Wayback Machine) | Direct | down | Accessing archived Orkut profiles via Wayback Machine |
| Share Secret Feedback (M) | Social Networks / Other Social Networks | Secreto username or user ID | Reference | manual-url | Looking up Secreto profiles by username to find anonymous feedback pages |
| TheHoodUp (NSFW) | Social Networks / Other Social Networks | Username, keywords, or topic search | Direct | - | Searching forum posts, user profiles, and discussions on TheHoodUp community board |
| Tumblr | Social Networks / Other Social Networks | Search terms, tags, or keywords | Direct | - | Finding content and user profiles by tag or keyword across Tumblr |
| VK | Social Networks / Other Social Networks | Username, profile URL, or search criteria | Direct | api | Finding Russian and Eastern European users and profile investigation |
| PinGroupie | Social Networks / Search | Pinterest username or board name | Direct | - | Pinterest user and board analysis |
| Social Searcher | Social Networks / Search | Keywords, hashtags, or usernames | Direct | api, freemium | Cross-platform social media content search |
| Talkwalker Social Media Search (R) | Social Networks / Search | Keywords, brands, or topics | Direct | api, reg, freemium | Enterprise-grade social media monitoring and trend analysis |
| Bellingcat Meta Content Library | Social Networks / Threads | Approved research queries and archive search filters | Direct | api, reg | Meta platform archive research for eligible organizations |
| Threads Dashboard | Social Networks / Threads | Threads username or account URL | Direct | api, reg, freemium | Threads account analytics and engagement investigation |
| Threads-Scraper (T) | Social Networks / Threads | Threads profile URL or post URL | Reference | local, degraded | Bulk extraction of Threads posts for offline analysis |
| ThreadsRecon (T) | Social Networks / Threads | Threads username | Reference | local | Threads profile investigation with sentiment and network analysis |

## Future seed types

| Tool | Framework Path | Input | Fit | Flags | Why It Matters |
| --- | --- | --- | --- | --- | --- |
| Addresses.com | People Search Engines / General People Search | Name, phone number, or address | Future | freemium | Quick US people and address lookups |
| Ancestry.com | People Search Engines / General People Search | Name, date of birth, location, or family details | Future | api, freemium | Comprehensive genealogy and family history research |
| AnyWho | People Search Engines / General People Search | Name, phone number, or address | Future | - | US white pages people and phone lookups |
| FaceCheckID | People Search Engines / General People Search | Uploaded face photograph | Future | freemium | Reverse face image search and identity verification |
| FamilySearch.org | People Search Engines / General People Search | Name, date, place, or family relationships | Future | api | Free genealogy research and historical record access |
| findmypast.com | People Search Engines / General People Search | Name, date of birth, location, or family details | Future | reg, paid | UK and Irish genealogy and historical records research |
| IDCrawl | People Search Engines / General People Search | Name, username, phone, or email | Future | - | Aggregated people search across social media and public records |
| InfoFlow Public People Search In Chilean | People Search Engines / General People Search | Name, RUN (Chilean ID number), or vehicle plate | Future | reg, freemium | Chilean public records and identity verification |
| Lullar | People Search Engines / General People Search | Email address or username | Future | - | Username and email-based social media profile enumeration |
| Melissa Data - People Finder (R) | People Search Engines / General People Search | Name, address, phone, or email | Future | api, reg, paid | Enterprise-grade identity verification and data enrichment |
| PeekYou | People Search Engines / General People Search | Name and optional location, or username | Future | api | Finding social media profiles and web presence by name |
| Snitch.name | People Search Engines / General People Search | First and last name | Future | degraded | Cross-platform social media profile discovery |
| ThatsThem | People Search Engines / General People Search | Name, address, phone number, email, or IP address | Future | freemium | Quick free people lookups by name, address, phone, email, or IP |
| usa-people-search.com | People Search Engines / General People Search | Name, address, or phone number | Future | reg, paid | US background checks and people search via public records |
| Webmii | People Search Engines / General People Search | First and last name | Future | - | Assessing online visibility and web footprint |
| Yasni | People Search Engines / General People Search | Name, location, profession, company, or skills | Future | - | Name-based people search with professional context |
| Amazon Registry Search | People Search Engines / Registries | Registrant name | Future | manual-url, degraded | Finding Amazon gift registries by name |
| My Registry | People Search Engines / Registries | Registrant name | Future | api | Universal cross-store gift registry creation and search |
| Registry Finder | People Search Engines / Registries | Registrant first and last name | Future | - | Cross-retailer gift registry search |
| The Bump | People Search Engines / Registries | Parent first and last name | Future | - | Finding baby registries by parent name |
| The Knot | People Search Engines / Registries | Couple first and last names | Future | - | Finding wedding registries by couple name |

## Community and archive search

| Tool | Framework Path | Input | Fit | Flags | Why It Matters |
| --- | --- | --- | --- | --- | --- |
| Anna's Archive | Archives / Web | Book title, author, ISBN, DOI, or keyword | Context | reg | Locating mirrored copies of books and papers from multiple sources |
| Archive.is | Archives / Web | URL | Reference | manual-url | Capturing and retrieving snapshots of volatile web pages |
| Browsershots | Archives / Web | URL and browser configuration | Context | down | Historical reference for legacy browser rendering captures |
| Cached Pages | Archives / Web | URL | Reference | manual-url | Finding recent cached copies of pages that changed or disappeared |
| Cached View | Archives / Web | URL | Reference | manual-url | Quick verification of whether a removed page still exists in cache |
| Common Crawl | Archives / Web | CC index query, URL, domain, or WARC request | Reference | api, manual-url | Large-scale historical web content mining and corpus analysis |
| PDF My URL | Archives / Web | URL | Reference | manual-url, freemium | Generating quick PDF evidence captures of web pages |
| Screenshots.com | Archives / Web | Domain or URL | Reference | reg, manual-url, freemium, degraded | Visual timeline checks of website appearance changes |
| Textfiles.com | Archives / Web | Keyword, directory path, or file browsing | Context | - | Researching legacy digital culture and historical text archives |
| UK Web Archive | Archives / Web | URL, title, topic, or keyword | Reference | manual-url, degraded | Accessing preserved UK web content and historical domain captures |
| Wayback Machine Chrome Extension | Archives / Web | Current tab URL or missing page request | Reference | local | Fast archive lookups while browsing dead or changed pages |
| Waybackpack (T) | Archives / Web | Domain/URL and optional date filters | Reference | api, local | Batch export of historical snapshots for offline analysis |
| Web Archive-RU | Archives / Web | URL or keyword | Reference | manual-url, degraded | Supplemental archive checks when mainstream archives lack coverage |
| WebCite | Archives / Web | Archived URL, DOI, or query string | Reference | manual-url, degraded | Retrieving historical citation captures that still remain accessible |
| Blog Search Engine | Online Communities / Blog Search Engines | Keywords and blog topics | Reference | manual-url | Blog discovery and topic-focused blog post searching |
| Live Journal Seek | Online Communities / Blog Search Engines | Keywords and search terms | Reference | manual-url | Finding public LiveJournal entries and historical community discussions |
| Discord Bot List | Online Communities / Discord Servers | Bot names, keywords, and categories | Context | api | Discord bot discovery and ecosystem mapping |
| ReconXplorer (T) | Online Communities / Discord Servers | IP addresses, emails, Discord tokens, and host data | Reference | api, local | Multi-input OSINT checks from a local scriptable toolkit |
| Top.gg | Online Communities / Discord Servers | Bot names, tags, and search filters | Context | api | Discord bot ranking analysis and app discovery |
| BoardReader | Online Communities / Forum Search Engines | Keywords, forum names, and topical queries | Reference | dork, manual-url | Finding forum threads and topic-centric discussion history |
| Craigslist Forums | Online Communities / Forum Search Engines | Forum categories, keywords, and regional navigation | Reference | dork, manual-url | Reviewing Craigslist community discussions and regional forum activity |
| Delphi Forum Search | Online Communities / Forum Search Engines | Forum names, categories, and keywords | Reference | manual-url, freemium | Niche forum discovery and historical community thread review |
| Google Groups Search | Online Communities / Forum Search Engines | Keywords, group names, authors, and date ranges | Reference | api, dork, manual-url | Researching archived mailing-list and discussion-group content |
| Omgili | Online Communities / Forum Search Engines | Keywords and Boolean-style forum queries | Reference | api, dork, manual-url, freemium | Forum discussion discovery with optional API-driven workflows |
| IRCP (T) | Online Communities / IRC Search | Target ranges, IRC ports, and server parameters | Reference | local | IRC server enumeration and protocol-level reconnaissance |
| ircsnapshot (T) | Online Communities / IRC Search | IRC server details, bot config, and channel targets | Reference | local | IRC topology mapping and user/channel relationship analysis |
| Mibbit | Online Communities / IRC Search | Channel or keyword queries (historical behavior) | Context | down | Legacy reference for historical IRC channel search workflows |
| netsplit.de | Online Communities / IRC Search | Channel names, keywords, and network filters | Reference | dork, manual-url | Passive IRC channel discovery and network trend checks |
| Arctic Shift | Online Communities / Reddit Communities | Search terms, dataset queries, or API-style requests | Context | api, freemium | Historical Reddit dataset analysis and subreddit research |
| Cama's Reddit Search | Online Communities / Reddit Communities | Usernames, subreddits, keywords, and date constraints | Context | - | Reddit user and subreddit content discovery |
| Reveddit | Online Communities / Reddit Communities | Reddit URLs, usernames, or subreddit paths | Reference | manual-url | Investigating deleted or removed Reddit discussions |

## Notes for Implementation

- Start with `Direct` tools in the username, email, and domain groups. They map cleanly onto Basalt's existing `username`, `email`, `domain`, `website`, and `account` node shapes.
- Treat most `Reference` tools as design input or optional wrappers. Many are multi-source orchestrators, local CLIs, or manual-url/dork helpers rather than stable single-service modules.
- Treat `Context` tools as secondary enrichment. They are useful once Basalt already has a confirmed domain or IP pivot, but they are less central than identity/account discovery.
- Treat `Future` tools as a backlog for expanding node types such as `phone`, `address`, `person_name`, `asn`, `ip`, `social_post`, or `message_channel`.
- Several `Reference` rows are active recon utilities. They are useful as ideas or optional wrappers, but they should not become Basalt defaults if the product stays primarily passive and consent-first.
