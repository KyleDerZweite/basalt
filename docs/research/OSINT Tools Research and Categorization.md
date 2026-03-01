# **Comprehensive Architectural Analysis of Open-Source Intelligence (OSINT) Utilities: Functionality, Overlaps, and Licensing Models**

The modern digital ecosystem has catalyzed an unprecedented proliferation of publicly accessible data, fundamentally transforming the disciplines of threat intelligence, digital forensics, and proactive cybersecurity operations. Open-Source Intelligence (OSINT) represents the systematic collection, processing, and analysis of this publicly available information to produce actionable strategic and tactical intelligence.1 While initially formalized within government and military intelligence communities, the application of OSINT has democratized, becoming a foundational element across commercial cybersecurity operations, law enforcement, investigative journalism, and corporate due diligence workflows.3

Reflecting its critical importance in modern security architectures, the global open-source intelligence market, valued at approximately $5.02 billion in 2018, is projected to reach $29.19 billion by 2026, driven by an expanding digital universe and the necessity to manage external attack surfaces.3 The intelligence spectrum encompasses multiple domains, and OSINT frequently acts as the unifying matrix that feeds into and complements these specialized intelligence types, particularly when organizations operate across complex, multi-jurisdictional environments.4

| Intelligence Type | Description | Primary Operational Use Cases |
| :---- | :---- | :---- |
| **OSINT** (Open Source) | Intelligence derived from publicly accessible digital or offline sources. | Due diligence, continuous threat monitoring, behavioral profiling. |
| **CYBINT** (Cyber) | Intelligence extracted from network traffic, malware analysis, IPs, and server logs. | Cybersecurity operations, incident response, network defense. |
| **HUMINT** (Human) | Information gathered directly from human sources or informants. | Field investigations, law enforcement operations, espionage. |
| **SOCMINT** (Social Media) | A specialized subset of OSINT focused exclusively on social networking platforms. | Extremism monitoring, brand risk assessment, psychosocial behavioral analysis. |

To harness this vast and chaotic data landscape effectively, investigators and security engineers rely on a complex, highly specialized ecosystem of software tools. These utilities range from localized metadata extractors to advanced, semi-autonomous orchestration frameworks capable of visualizing sprawling global networks. For security architects and software developers tasked with building a unified collection, repository, or integrated platform of OSINT tools, understanding the technical mechanisms, the precise operational targets, the functional overlaps, and the software licensing restrictions of these utilities is paramount. Licensing dictates not only legal compliance but also the architectural feasibility of integrating multiple tools into a cohesive enterprise or community platform, particularly regarding the distribution of derivative works.5

The following report provides an exhaustive, deeply analytical evaluation of the premier OSINT tools available today. It dissects their core purposes and operational targets, delineates where their functionalities intersect, and provides a rigorous analysis of their licensing frameworks to guide the construction of an integrated, legally compliant OSINT repository.

## **The Taxonomic Structure of OSINT Operations and Directory Frameworks**

The application of OSINT tools follows a highly structured lifecycle: source identification, data collection, data enrichment, behavioral or network analysis, and the presentation of results.6 Because the data sources are exceptionally diverse—spanning domain name registries, cryptocurrency blockchains, social media platforms, public government records, and the dark web—no single utility can perform all functions optimally. Instead, tools are highly specialized, often operating in interconnected, automated chains where the output of one application (such as a discovered email address) becomes the input vector for another (such as a reverse-identity analyzer).

To navigate this complexity, overarching directories and taxonomies have been developed by the intelligence community. The most prominent of these is the OSINT Framework.7 Rather than being an executable software suite itself, the OSINT Framework is a comprehensive web-based directory that organizes hundreds of open-source intelligence resources by source, type, and operational context.3 It classifies tools across a granular taxonomy, including distinct categories for Email Addresses, IP and MAC Addresses, Digital Currency, the Dark Web, Metadata, and Malicious File Analysis.8

The OSINT Framework employs a specific taxonomic legend to help researchers understand the operational requirements of each tool prior to deployment or integration.8 Tools marked with a "(T)" indicate a localized software utility requiring installation and local execution, differentiating them from web-based services. Tools marked with a "(D)" denote the use of Google Dorks, which leverage advanced search operators to bypass standard indexing limits. The "(R)" designation indicates that the service requires user registration or an API key, while "(M)" indicates a URL that requires the investigator to manually modify search terms within the web address to execute the query.8

Beyond the primary OSINT Framework, the community relies heavily on curated repositories hosted on platforms like GitHub, such as the "Awesome-OSINT" lists maintained by various security researchers and organizations like TraceLabs.1 These repositories categorize tools into actionable investigative phases, such as Multi-Search, Username Check, Threat Actor Search, and Live Cyber Attack Maps.1 While these directories serve as the conceptual map for investigators, executing an investigation requires deploying specific software engines. These engines can be broadly categorized into centralized orchestration frameworks, automated reconnaissance engines, infrastructure mapping tools, identity tracking utilities, and specialized single-purpose analyzers.

## **Centralized OSINT Frameworks and Package Managers**

Centralized frameworks are designed to act as the primary operational terminal for an investigator. They provide a standardized, command-line environment where various independent modules can be executed, ensuring that disparate data streams are collected, normalized, and stored in a unified database format for subsequent querying.

### **Recon-ng**

Recon-ng is a full-featured web reconnaissance framework written in Python, explicitly designed to reduce the time an analyst spends harvesting open-source data by providing a powerful, modular environment.10 Its exact operational target is web-based open-source reconnaissance, distinctly separating itself from exploitation frameworks or social engineering toolkits.11 Recon-ng features a command-line interface designed to closely mirror the look, feel, and operational logic of the Metasploit Framework, significantly reducing the learning curve for penetration testers.11

The framework is architected around a highly extensible modular system. Developers can easily build and maintain modules by creating subclasses of the core module class, which acts as a customized interpreter with built-in interfaces for database interaction, credential management, and web request handling.11 Recon-ng enforces strict PEP8 compliance for module submissions and utilizes an indexing system to manage version control within its marketplace.11 Its internal API provides advanced mixins, such as threading capabilities, mechanized browser objects, and DNS resolvers, allowing modules to easily execute threaded API searches against platforms like Twitter, Shodan, and GitHub.11 Recon-ng is licensed under the GNU General Public License v3.0 (GPL-3.0), making it free software but imposing strong copyleft restrictions on any derivative works.11

### **sn0int**

sn0int operates as a semi-automatic OSINT framework and package manager tailored for IT security professionals, bug bounty hunters, and intelligence analysts.14 Its primary purpose is enumerating external attack surfaces by processing public information and mapping the results in a unified format for follow-up investigations.17 Its targets include harvesting subdomains from Certificate Transparency logs, interrogating passive DNS databases, extracting emails from PGP keyservers, identifying compromised logins in data breaches, and gathering intelligence on phone numbers.19

A critical architectural distinction of sn0int is its execution environment. Heavily inspired by Recon-ng and Maltego, sn0int operates as a flexible package manager where investigations are not hardcoded into the core source. Instead, investigations are provided by community-developed modules that are executed within a highly secure sandbox environment.17 This allows users to write their own modules, publish them to the sn0int registry, and seamlessly ship updates to end-users without requiring pull requests to the core codebase.17 sn0int is licensed under the GNU General Public License v3.0 or later (GPLv3+), ensuring its ecosystem remains entirely open-source.18

### **OWASP Maryam**

OWASP Maryam is a modular, open-source framework specifically engineered to provide a robust environment for rapidly harvesting data from search engines, social networks, and public databases.21 Written in Python, its primary targets include vulnerability research, incident response, and threat intelligence.24 The framework categorizes its operational modules primarily into "Footprint" and "Search" designations.12 The Footprint modules help discover technical network information, ranging from DNS brute-forcing to top-level-domain variant discovery and deep web page crawling.12 The Search modules provide command-line interfaces to a vast array of search engines, enabling automated keyword and link extraction.12 Like Recon-ng and sn0int, OWASP Maryam operates under the strong copyleft GNU General Public License v3.0 (GPL-3.0).21

### **Overlap and Differentiation in Centralized Frameworks**

OWASP Maryam, sn0int, and Recon-ng exhibit significant functional overlap, as all three are modular, command-line-driven platforms designed to unify the execution of multiple OSINT tasks. Recon-ng is the oldest and most established, boasting a massive library of user-contributed modules and a familiar Metasploit-like interface, making it highly suitable for traditional red-team operations.11 sn0int differentiates itself through its modern architecture, functioning explicitly as a package manager where modules are executed in isolated sandboxes, providing enhanced security and a streamlined update cycle for custom scripts.17 OWASP Maryam serves as a direct alternative, focusing heavily on both technical footprinting and external search engine integrations.12 An exhaustive OSINT collection could technically include all three, but for automated pipeline integration, sn0int's package management provides a more modernized infrastructure, whereas Recon-ng offers unmatched legacy module support.

## **Automated Reconnaissance and Visual Link Analysis Engines**

While command-line frameworks require the manual execution of specific modules in a sequential manner, automated engines attempt to ingest a single starting node (a seed) and autonomously traverse the internet to build a comprehensive intelligence map. Conversely, visual link analysis engines allow investigators to manually tease apart complex relationships within massive data sets.

### **SpiderFoot**

SpiderFoot is an advanced OSINT automation tool designed to streamline the process of gathering intelligence for threat intelligence and mapping attack surfaces.2 Its primary purpose is to pull data from over 100 public sources—including social media, websites, threat intelligence feeds, and DNS records—to map the relationships between different entities.2 Investigators input targets such as IP addresses, subnets (CIDR), ASNs, email addresses, phone numbers, or cryptocurrency wallets, and SpiderFoot exhaustively enumerates the target's digital footprint.25 It integrates with platforms like SHODAN, HaveIBeenPwned, GreyNoise, and SecurityTrails, while also featuring built-in TOR integration for dark web querying.25

SpiderFoot's architecture operates on a publisher/subscriber model across more than 200 modules, executing targeted scanning, enumeration, cloud infrastructure discovery (such as exposed Amazon S3 buckets), and metadata analysis.25 A standout feature is its YAML-configurable correlation engine, which automates analysis by querying scan results, filtering data, and grouping it to present opinionated results regarding potential vulnerabilities.25 These correlation rules utilize collection blocks with exact or regex matching, aggregation logic, and analysis logic that can drop results failing to meet specific thresholds.25 The open-source version of SpiderFoot is distributed under the highly permissive MIT License, allowing developers to integrate its scanning capabilities into proprietary platforms without copyleft restrictions.13

### **Maltego**

Maltego represents the industry standard for visual link analysis and data mining.10 Unlike SpiderFoot's automated, "set-and-forget" reporting approach, Maltego provides an interactive graphical interface where analysts can correlate domains, email addresses, people, and network assets through the automated collection of public information using scripts known as "Transforms".26 Its exact purpose is to reveal hidden connections between entities, presenting them in interactive graphs that help investigators immediately understand complex networks and unexpected paths for exploitation.26 Maltego is heavily utilized in complex cybersecurity operations, financial crime investigations, and law enforcement profiling.30

Maltego's licensing model differs fundamentally from the open-source tools discussed previously. It is a commercial, proprietary product, though it offers a "Community Edition" which is free to use but strictly limited to non-commercial, testing, and evaluation purposes.30 The Community Edition enforces artificial limitations on the number of results returned per transform and the overall size of the graph.33 Commercial integration requires purchasing Maltego Professional or Enterprise licenses, which operate on a per-seat, subscription basis and provide unlimited API lookups and access to premium commercial data connectors.31

### **Gephi**

While not explicitly a data-gathering OSINT tool, Gephi is a critical companion utility for advanced intelligence analysis. It is an interactive, open-source visualization and exploration platform designed to analyze complex networks, graphs, and spatial structures.34 Investigators use Gephi to ingest massive datasets—often exported as CSV or GEXF files from tools like SpiderFoot or custom scrapers—to apply rigorous mathematical algorithms for link analysis.35 Its purpose is to identify central nodes, analyze clustering metrics, and reveal hidden structural patterns within illicit networks or massive disinformation campaigns.13 Gephi is highly regarded in both academic research and journalism for its ability to render visually compelling network graphs that highlight complex relationships.35 Gephi is open-source software and is released under the GNU General Public License (GPL).34

| Analysis Tool | Primary Analytical Focus | Execution Style | Licensing Model |
| :---- | :---- | :---- | :---- |
| **SpiderFoot** | Automated external attack surface enumeration and data correlation. | Autonomous / Set-and-Forget | MIT License |
| **Maltego** | Interactive visual link analysis and recursive node expansion. | Human-in-the-Loop GUI | Proprietary / Freemium |
| **Gephi** | Mathematical network graphing, centrality analysis, and spatial mapping. | Post-Collection Analysis GUI | GPL-3.0 |

## **External Attack Surface Management: Domain and Infrastructure Reconnaissance**

In cyber intelligence, understanding the technical footprint of an organization is the necessary first step in assessing its vulnerability profile, identifying misconfigurations, or attributing malicious infrastructure. Tools in this category focus strictly on the underlying architecture of the internet: DNS records, IP allocations, and web server configurations.

### **OWASP Amass**

The OWASP Amass project specializes in highly rigorous, in-depth attack surface mapping and external asset discovery.39 Its exact target is the complete enumeration of an organization's DNS infrastructure and subdomains.39 Amass achieves this by deploying a combination of active reconnaissance and passive OSINT techniques, pulling from dozens of data sources, scraping SSL/TLS certificates, and utilizing advanced, recursive DNS resolution tactics.39 Because of its ability to process massive amounts of data and discover obscure subdomains that traditional bruteforcing misses, Amass is widely considered the gold standard for DNS mapping at scale.29 It operates under the Apache License 2.0, a permissive license that allows for commercial use, modification, and distribution, while explicitly granting patent rights to the user.39

### **theHarvester**

theHarvester is a focused, lightweight command-line tool designed to gather email accounts, subdomain names, virtual hosts, open ports, and employee names related to a specific target domain or company.3 It is heavily utilized during the initial, passive reconnaissance phases of red team assessments and penetration testing.26 Rather than executing aggressive DNS resolution like Amass, theHarvester sources its data passively by scraping major search engines (Google, Bing, Baidu), parsing PGP key servers, and integrating with external APIs like Shodan to query discovered hosts.43 It features capabilities for DNS brute-forcing, virtual host verification, and exporting results directly to XML and JSON formats for pipeline integration.44 theHarvester is licensed under the GNU General Public License (historically GPLv2, transitioning to GPL-3.0).45

### **FinalRecon**

FinalRecon is an all-in-one web reconnaissance tool written in Python. Its primary goal is to provide a rapid, comprehensive overview of a target web application without requiring the investigator to sequentially execute multiple discrete tools.47 It targets header information, WHOIS records, SSL certificate data, and performs deep web crawling—extracting HTML, CSS, JavaScript links, internal and external routing, and historical snapshots from the Wayback Machine.47 It also executes basic DNS enumeration (including DMARC records) and fast port scanning for the top 1000 standard services.47 FinalRecon is distributed under the highly permissive MIT License.48

### **Overlap and Differentiation in Infrastructure Tooling**

The overlap between Amass, theHarvester, and FinalRecon highlights the varying tactical depths of infrastructure OSINT. theHarvester is optimized for speed and passive collection; it is the ideal utility for quickly mapping an organization's employee email naming conventions and indexing external-facing subdomains without sending direct traffic to the target's servers, thereby minimizing detection.26 Amass is a much heavier, more rigorous tool, built to conduct exhaustive DNS mapping and recursive resolution, making it the superior choice for comprehensive external attack surface management programs.29 FinalRecon occupies the middle ground, focusing heavily on the web-application layer (headers, SSL analysis, deep web crawling) rather than strictly operating at the network routing layer.47 Integrating all three provides a tiered approach: theHarvester for passive sweeps, Amass for deep infrastructure enumeration, and FinalRecon for web-application profiling.

## **Identity Resolution and Social Media Intelligence (SOCMINT)**

Human behavior online is fundamentally habitual. Threat actors, targets of investigations, and standard users frequently reuse usernames, profile images, and email addresses across multiple, disparate platforms.43 Identity OSINT tools exploit this behavioral consistency, utilizing high-speed automation to map a target's digital footprint across the internet.

### **Sherlock**

Sherlock is a high-speed command-line utility designed to hunt down social media accounts associated with a specific username.43 Its operational target is the systematic querying of over 400 different social networks, forums, and web platforms to determine if a specific username exists on that service.43 Sherlock is unique in its built-in support for network routing privacy; investigators can pass arguments (--tor or \-t) to route all queries through the Tor network, maintaining operational anonymity during highly sensitive investigations, such as fraud, harassment, or human trafficking cases.43 Discovered accounts are automatically stored in individual text files, and results can be exported in CSV, JSON, or XLSX formats for further analysis.50 Sherlock is distributed under the MIT License.43

### **Maigret**

Maigret is an advanced evolution and fork of the Sherlock project, providing significantly deeper investigative capabilities.51 While Sherlock confirms the mere existence of a username, Maigret's exact purpose is to generate a comprehensive, detailed dossier on an individual.43 It vastly expands the search scope, interrogating more than 3,000 sites by default, including Tor hidden services and I2P networks.51

Crucially, Maigret goes beyond simple HTTP status checking; it parses the HTML of the discovered profile pages to extract secondary intelligence, such as full names, geographic locations, profile images, and links to alternate platforms.43 It also possesses recursive search capabilities, meaning if it discovers a new, associated username or ID on a profile page, it will automatically feed that new ID back into its search engine to expand the dossier autonomously.51 As a fork of Sherlock, Maigret inherits and maintains the MIT License.51

### **WhatsMyName**

WhatsMyName is a foundational nickname enumeration tool and project. Unlike standalone CLI executables that run custom logic, WhatsMyName operates primarily as a massive, community-maintained JSON dataset of web platform endpoints, outlining the specific HTTP response structures that indicate a user's presence or absence.52 Many broader OSINT frameworks, including Recon-ng and SpiderFoot, rely on the WhatsMyName dataset as their underlying engine for username enumeration.

### **Holehe**

Holehe performs a highly specialized form of reverse-identity OSINT. While tools like Sherlock start with a username, Holehe's operational target is an email address, and its purpose is to determine which online services are registered to that specific email.54 It achieves this not by searching for public, indexed mentions of the email address, but by programmatically interacting with the "forgot password," login, and registration APIs of over 120 websites (including Twitter, Instagram, Imgur, and various dating platforms).54 It carefully parses the server responses from these APIs to confirm account existence without actually resetting the password or triggering security alerts to the target user.54 Because of its unique mechanism, it is a highly potent tool for deanonymizing email addresses. Holehe is released under the GNU General Public License v3.0 (GPL-3.0).56

### **Social Mapper**

Social Mapper, developed by Trustwave, is an open-source intelligence tool that bridges traditional text-based profiling with advanced biometric analysis.58 Its target is a provided list of names and photographs; its purpose is to autonomously correlate those individuals across major social networks like LinkedIn, Facebook, Twitter, Instagram, and VKontakte.58

Social Mapper bypasses the restrictive API limitations of modern social networks by instrumenting a live Firefox browser. It automatically logs into the networks, searches for the provided names, downloads the top 10 to 20 profile pictures associated with those search results, and applies facial recognition algorithms to verify a biometric match against the original target photo.59 The final output is a highly visual HTML report and a CSV file, which penetration testers frequently utilize to craft highly targeted spear-phishing or social engineering campaigns based on the target's verified social media presence.58 Social Mapper is licensed as free software, with repositories frequently listing it under dual-compatible frameworks, typically MIT or GPL.60

| Identity Tool | Primary Input Vector | Analytical Methodology | Software License |
| :---- | :---- | :---- | :---- |
| **Sherlock** | Username | Rapid HTTP Status matching across 400+ endpoints. | MIT License |
| **Maigret** | Username | Deep HTML parsing, recursive dossier generation across 3000+ sites. | MIT License |
| **Holehe** | Email Address | Interrogation of password reset and registration APIs. | GPL-3.0 |
| **Social Mapper** | Name & Image | Automated browser scraping coupled with facial recognition. | MIT / GPL |

## **Telephonic, Spatial, and Metadata Analytics**

Beyond names, domains, and email addresses, critical infrastructure and human targets produce vast amounts of secondary, often hidden telemetry. Tools in this category specialize in dissecting specific data structures, such as telephonic routing information, embedded geospatial coordinates, and document metadata.

### **PhoneInfoga**

PhoneInfoga is an advanced information-gathering framework exclusively targeting international phone numbers.62 It processes numbers using international standard formats (such as E.164 and RFC3966) to extract fundamental details, including the country of origin, area code, service carrier, and the specific line type, which helps investigators distinguish between cellular devices, landlines, and Voice over IP (VoIP) numbers.62

Beyond static parsing, PhoneInfoga executes advanced OSINT footprinting by querying reputation APIs, phone fraud databases, and utilizing highly customized Google Dorks to find web documents containing the specific number.62 Recognizing that complex Google Dorks frequently trigger bot-detection captchas and result in IP blacklisting, PhoneInfoga integrates clever bypass mechanisms, mimicking normal browser headers and allowing users to manually pass Google abuse exemption tokens back into the command-line interface.62 It is highly effective at identifying if a number belongs to a disposable or temporary provider. PhoneInfoga is licensed under the GNU General Public License v3.0 (GPL-3.0).62

### **Creepy (Geocreepy)**

Creepy is a specialized OSINT tool explicitly targeted at gathering geolocation telemetry.64 Its purpose is to harvest location-based data from social networking platforms and image hosting services.65 By analyzing data points such as platform check-ins, geotagged tweets, and the embedded metadata within hosted images, Creepy aggregates a target's physical movements. It plots these movements onto an interactive map interface (such as Google Maps), allowing investigators to filter the data by specific date ranges and precise location coordinates to establish a geographic pattern of life.65 Creepy is distributed under the GNU General Public License v3.0 (GPL-3.0).65

### **ExifTool**

ExifTool is the ubiquitous, platform-independent Perl library and command-line application used globally for reading, writing, and manipulating metadata across a massive variety of file formats.66 In the context of OSINT, its target is digital media—specifically images, PDFs, and video files.13 Its purpose is to extract hidden telemetry, including EXIF data, GPS coordinates, camera make and model, software editing versions, and creation timestamps.13 This extraction is essential for verifying the authenticity of images, identifying the exact physical location where a photograph was taken, or determining if digital manipulation has occurred via Error Level Analysis (ELA).67 ExifTool is released under the GNU General Public License v1.0 or later (GPLv1+) and the Artistic License.68

### **FOCA (Fingerprinting Organizations with Collected Archives)**

FOCA operates as a specialized metadata analysis tool, but rather than targeting individual files like ExifTool, it targets entire corporate network infrastructures.2 Its purpose is to query major search engines to discover publicly hosted documents (such as.doc,.pdf,.xls, and.ppt files) associated with a specific target domain, autonomously download them, and extract their metadata in bulk.69

The metadata extracted by FOCA frequently leaks highly sensitive internal information, including usernames, internal corporate email addresses, specific software builds, network printer names, and internal IP routing architectures.69 Penetration testers and red teams use FOCA's output to map Active Directory environments from the outside and craft high-probability spear-phishing payloads tailored to the exact software versions running inside the network.69 FOCA requires a SQL backend to process the massive amounts of data it harvests and is licensed under the GNU General Public License v3.0 (GPL-3.0).69

## **Specialized Enclaves: Google Ecosystems, Browser Extensions, and the Dark Web**

Certain OSINT tools are built to exploit the highly specific architecture of a single service provider, to integrate directly into the analyst's workflow via the browser, or to operate within the encrypted confines of specific network topologies like the Tor network.

### **GHunt**

GHunt is described as an "offensive Google framework" specifically targeting the Google architecture.72 Its purpose is to extract extensive intelligence from Google-associated accounts. Using a known email address, a Google Drive link, or a Gaia ID, GHunt queries backend Google APIs to retrieve a wealth of associated profile details, connected Google services, YouTube channel data, Google Maps reviews (which frequently leak a target's physical location and habits), and specific device telemetry.72 It is highly effective because it leverages Google's own interconnected single-sign-on (SSO) ecosystem against the target.

GHunt utilizes the GNU Affero General Public License (AGPL).72 The AGPL is the strictest form of copyleft licensing. It was specifically designed to close the "SaaS loophole" of the standard GPL. If an organization incorporates GHunt into a web-based OSINT dashboard offered to clients over the internet, the organization must make the source code of that entire web service available to its users.73

### **Mitaka**

Mitaka takes a different architectural approach, functioning as a lightweight browser extension rather than a standalone command-line application.74 Its target is Indicators of Compromise (IoCs) encountered natively within a web browser during an investigation, such as suspicious IP addresses, URLs, file hashes, and cryptocurrency wallets.74 Its purpose is to provide immediate, context-menu-driven OSINT. When an analyst highlights a defanged URL (e.g., hxxp://example\[.\]com), Mitaka automatically refangs it (http://example.com) and securely queries over 65 external intelligence services (including VirusTotal, Shodan, and Censys) to assess its threat level without exposing the analyst to the malicious payload.75 Mitaka is licensed under the permissive MIT License.77

### **OnionScan**

Standard web crawlers and OSINT tools fail to index the deep and dark web. OnionScan's operational target is the Tor anonymity network, specifically Tor Hidden Services (domains ending in.onion).79 Its purpose is to systematically scan these hidden services to map the dark web ecosystem and identify operational security (OpSec) failures made by the operators.79 OnionScan actively looks for leaked cleartext IP addresses, open server status pages (such as Apache server-status), exposed metadata in images hosted on the hidden service, and overlapping SSH keys.79 By finding these overlaps, researchers and law enforcement agencies attempt to de-anonymize illicit dark web marketplaces and tie multiple hidden services back to a single physical server or operator.1 OnionScan is distributed under the MIT License.82

## **Licensing Frameworks for Enterprise OSINT Integration**

Constructing a unified, enterprise-grade collection of OSINT tools is not merely a technical challenge of pipeline integration; it is a complex legal and architectural undertaking governed by software licenses. Open-source licenses dictate exactly how the code can be used, modified, and redistributed, which is a critical consideration when these tools are integrated into commercial or proprietary investigative platforms.5

The tools analyzed in this report fall into four primary licensing categories, each with distinct implications for building a unified collection.

### **Permissive Licenses (MIT, Apache 2.0)**

Tools operating under permissive licenses represent the safest options for integration into proprietary or commercial collections. The **MIT License** is utilized by **SpiderFoot** 25, **Sherlock** 50, **Maigret** 51, **FinalRecon** 49, **Mitaka** 77, and **OnionScan**.82 The **Apache License 2.0** is utilized by **Amass**.39

These highly permissive frameworks allow organizations to freely incorporate the code into proprietary threat intelligence dashboards, modify the logic, and distribute the resulting software without being forced to release their own source code.61 The primary requirement is preserving the original copyright and license notices within the documentation or software distribution.77 The Apache 2.0 license provides an additional layer of safety for commercial entities by explicitly granting patent rights to the user, protecting against future patent litigation from the original developers.73

### **Strong Copyleft (GPL-3.0 and Derivatives)**

A vast majority of the deeper, highly specialized investigative tools use the **GNU General Public License v3.0 (GPL-3.0)**, including **OWASP Maryam** 21, **Recon-ng** 11, **sn0int** (GPLv3+) 19, **Holehe** 57, **PhoneInfoga** 62, **FOCA** 71, and **Creepy**.65 **theHarvester** utilizes a mix of GPLv2 and GPL-3.0 45, while **ExifTool** utilizes GPLv1+ and the Artistic License.68

The GPL family imposes strong copyleft restrictions. These licenses mandate that any modifications to the tool, or any larger software projects that directly incorporate or link to this licensed work, must also be released under the same GPL license.61 For a security architect building a unified OSINT toolkit, this requires careful pipeline design. Running these tools as independent executables via command-line sub-processes (where the tools remain separate programs) is generally permissible in a commercial setting. However, statically linking their source code directly into a proprietary graphical interface or central execution binary would legally force the entire proprietary interface to become open-source.61

### **Network Copyleft (AGPL)**

**GHunt** utilizes the **GNU Affero General Public License (AGPL-3.0)**.72 The AGPL contains a specific clause designed for software running over a network. If an organization incorporates GHunt into a web-based OSINT dashboard or SaaS platform offered to clients, the organization is legally obligated to make the source code of that entire web service available to its users.73 This makes AGPL tools highly restrictive and generally incompatible with integration into closed-source, commercial SaaS threat intelligence platforms.

### **Proprietary and Freemium Models**

**Maltego** represents the proprietary edge of the spectrum. While a free Community Edition exists, its software license agreement strictly prohibits commercial use, monetary compensation, or use for commercial benefit.32 Integrating Maltego's powerful graphing capabilities into a commercial OSINT collection requires negotiating enterprise licensing agreements and paying per-seat subscriptions.31

| OSINT Tool | Core Operational Category | Primary Target / Output | Software License |
| :---- | :---- | :---- | :---- |
| **SpiderFoot** | Automated Intelligence | IP, Domain, Network, Name enumeration | MIT License |
| **Sherlock** | Identity OSINT | Username existence across 400+ platforms | MIT License |
| **Maigret** | Identity OSINT | Username dossier generation (3000+ platforms) | MIT License |
| **FinalRecon** | Infrastructure OSINT | Rapid web target profiling and crawling | MIT License |
| **OnionScan** | Dark Web OSINT | Tor hidden service OpSec failures | MIT License |
| **Amass** | Infrastructure OSINT | Deep DNS and subdomain mapping | Apache License 2.0 |
| **OWASP Maryam** | Centralized Framework | Search engine and footprint automation | GPL-3.0 |
| **Recon-ng** | Centralized Framework | Modular web-based reconnaissance | GPL-3.0 |
| **sn0int** | Centralized Framework | Sandboxed attack surface package management | GPLv3+ |
| **PhoneInfoga** | Telemetry OSINT | International phone number footprinting | GPL-3.0 |
| **Holehe** | Identity OSINT | Email address platform registration | GPL-3.0 |
| **FOCA** | Telemetry OSINT | Network-scale document metadata extraction | GPL-3.0 |
| **GHunt** | Specialized OSINT | Google ecosystem profiling | AGPL-3.0 |
| **Maltego** | Visual Analysis | Interactive graphical link analysis | Proprietary (Free CE) |

## **Architectural Recommendations for an Integrated OSINT Collection**

The strategic value of a comprehensive OSINT repository lies not merely in aggregating a high volume of independent tools, but in the intelligent orchestration of their overlapping capabilities. An effective architecture must account for the natural flow of an investigation, utilizing the output of one tool as the input vector for the next, while strictly adhering to licensing constraints.

A robust architectural blueprint should begin with broad infrastructure reconnaissance. An investigation typically initiates with a target domain or corporate entity. Tools like **Amass** and **theHarvester** are deployed first to perform the heavy lifting, exhaustively mapping the external infrastructure, identifying hidden subdomains, and extracting corporate email naming conventions.39 **FinalRecon** can be run concurrently against discovered subdomains to profile the web-application layer, extracting SSL data and internal document structures.47

Once initial endpoints and identities are discovered, the collection pipeline must pivot to identity resolution. Discovered email addresses are fed iteratively into **Holehe** to reveal associated third-party platform accounts and social media profiles.54 The usernames associated with those profiles are then piped into **Maigret** for deep-dive dossier generation, capitalizing on its recursive link discovery to expand the identity map.43 If high-value targets are identified visually, **Social Mapper** can be deployed to utilize facial recognition algorithms to verify biometric matches across networks.58

Secondary telemetry must then be analyzed. Target phone numbers extracted from web documents or WHOIS records are processed via **PhoneInfoga** to determine carrier routing and regionalization.62 Corporate documents pulled from the target domain during the initial web crawl are passed through **FOCA** in bulk to reveal internal network maps, printer configurations, and employee identities hidden within the file metadata.69

Finally, the disparate data streams generated by these specialized command-line utilities must be aggregated and visualized. Permissive engines like **SpiderFoot** can automate much of this initial correlation using its YAML rule sets.25 However, the final synthesis and visualization should be rendered using tools built for human-in-the-loop analysis. The raw data lakes can be imported into **Maltego** for targeted, manual node expansion 26, or, if the dataset is massive, processed through **Gephi** to apply mathematical network algorithms that highlight centrality and structural vulnerabilities.35

In constructing this ecosystem, the selection of the central orchestration framework is critical. Utilizing **sn0int** 17 provides a highly modern, sandbox-driven environment ideal for community-driven updates and secure execution, while **Recon-ng** 11 offers robust, familiar terminal operations for teams accustomed to traditional penetration testing workflows. By carefully balancing the permissive, highly integratable nature of tools like SpiderFoot, Amass, and Sherlock with the powerful, though legally restrictive, copyleft engines like PhoneInfoga, FOCA, and GHunt, security architects can build an exhaustive, legally compliant, and operationally devastating open-source intelligence capability.

#### **Referenzen**

1. jivoi/awesome-osint: :scream \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/jivoi/awesome-osint](https://github.com/jivoi/awesome-osint)  
2. 9 Top OSINT Tools & How to Evaluate Them | Wiz, Zugriff am Februar 28, 2026, [https://www.wiz.io/academy/threat-intel/osint-tools](https://www.wiz.io/academy/threat-intel/osint-tools)  
3. Top 15 Free OSINT Tools To Collect Data From Open Sources \- Recorded Future, Zugriff am Februar 28, 2026, [https://www.recordedfuture.com/threat-intelligence-101/tools-and-technologies/osint-tools](https://www.recordedfuture.com/threat-intelligence-101/tools-and-technologies/osint-tools)  
4. OSINT Tools And Techniques | OSINT Technical Sources \- Neotas, Zugriff am Februar 28, 2026, [https://www.neotas.com/osint-tools-and-techniques/](https://www.neotas.com/osint-tools-and-techniques/)  
5. Open Source License Compliance \- FOSSA, Zugriff am Februar 28, 2026, [https://fossa.com/solutions/oss-license-compliance/](https://fossa.com/solutions/oss-license-compliance/)  
6. Breadcrumbs in the Digital Forest: Tracing Criminals through Torrent Metadata with OSINT, Zugriff am Februar 28, 2026, [https://arxiv.org/html/2601.01492v1](https://arxiv.org/html/2601.01492v1)  
7. osint-tools-list · GitHub Topics, Zugriff am Februar 28, 2026, [https://github.com/topics/osint-tools-list](https://github.com/topics/osint-tools-list)  
8. OSINT Framework, Zugriff am Februar 28, 2026, [https://osintframework.com/](https://osintframework.com/)  
9. tracelabs/awesome-osint: 🕵️ A curated list of awesome ... \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/tracelabs/awesome-osint](https://github.com/tracelabs/awesome-osint)  
10. Top 10 OSINT Tools Everyone Should Know | SMIIT CyberAI, Zugriff am Februar 28, 2026, [https://smiit-cyberai.com/blog15](https://smiit-cyberai.com/blog15)  
11. lanmaster53/recon-ng: Open Source Intelligence gathering ... \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/lanmaster53/recon-ng](https://github.com/lanmaster53/recon-ng)  
12. OWASP Maryam \- Iintroduction to the OSINT tool \- scip AG, Zugriff am Februar 28, 2026, [https://www.scip.ch/en/?labs.20220113](https://www.scip.ch/en/?labs.20220113)  
13. Best OSINT Tools for Intelligence Gathering (2026) Free and Paid \- ShadowDragon, Zugriff am Februar 28, 2026, [https://shadowdragon.io/blog/best-osint-tools/](https://shadowdragon.io/blog/best-osint-tools/)  
14. sn0int Documentation, Zugriff am Februar 28, 2026, [https://sn0int.readthedocs.io/\_/downloads/en/stable/pdf/](https://sn0int.readthedocs.io/_/downloads/en/stable/pdf/)  
15. sn0int | Kali Linux Tools, Zugriff am Februar 28, 2026, [https://www.kali.org/tools/sn0int/](https://www.kali.org/tools/sn0int/)  
16. OSINT Tool: sn0int \- Black Hat Ethical Hacking, Zugriff am Februar 28, 2026, [https://www.blackhatethicalhacking.com/tools/sn0int/](https://www.blackhatethicalhacking.com/tools/sn0int/)  
17. sn0int — sn0int documentation, Zugriff am Februar 28, 2026, [https://sn0int.readthedocs.io/](https://sn0int.readthedocs.io/)  
18. sn0int download | SourceForge.net, Zugriff am Februar 28, 2026, [https://sourceforge.net/projects/sn0int.mirror/](https://sourceforge.net/projects/sn0int.mirror/)  
19. kpcyrd/sn0int: Semi-automatic OSINT framework and package manager \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/kpcyrd/sn0int](https://github.com/kpcyrd/sn0int)  
20. sn0int \- is a semi-automatic OSINT for IT security professionals and bug hunters, Zugriff am Februar 28, 2026, [https://forum.cloudron.io/topic/4509/sn0int-is-a-semi-automatic-osint-for-it-security-professionals-and-bug-hunters](https://forum.cloudron.io/topic/4509/sn0int-is-a-semi-automatic-osint-for-it-security-professionals-and-bug-hunters)  
21. OWASP Maryam, Zugriff am Februar 28, 2026, [https://owasp.org/www-project-maryam/](https://owasp.org/www-project-maryam/)  
22. Home · saeeddhqan/Maryam Wiki \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/saeeddhqan/maryam/wiki](https://github.com/saeeddhqan/maryam/wiki)  
23. OWASP Maryam — Maryam latest documentation, Zugriff am Februar 28, 2026, [https://maryam.readthedocs.io/](https://maryam.readthedocs.io/)  
24. Maryam Tool \- OSINT for data gathering | Briskinfosec \- YouTube, Zugriff am Februar 28, 2026, [https://www.youtube.com/watch?v=dG3gslhDmaM](https://www.youtube.com/watch?v=dG3gslhDmaM)  
25. smicallef/spiderfoot: SpiderFoot automates OSINT for threat ... \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/smicallef/spiderfoot](https://github.com/smicallef/spiderfoot)  
26. Five Essential OSINT Tools Every Penetration Tester Needs to Know | by 0xCh1no \- Medium, Zugriff am Februar 28, 2026, [https://medium.com/@euromarquesrafael/five-essential-osint-tools-every-penetration-tester-needs-to-know-45881400d52b](https://medium.com/@euromarquesrafael/five-essential-osint-tools-every-penetration-tester-needs-to-know-45881400d52b)  
27. Top 15 OSINT Tools in 2025: Leading Platforms OSINT Investigations Platforms \- Medium, Zugriff am Februar 28, 2026, [https://osintbyle.medium.com/top-15-osint-tools-in-2025-leading-platforms-osint-investigations-platforms-2771871b7bb6](https://osintbyle.medium.com/top-15-osint-tools-in-2025-leading-platforms-osint-investigations-platforms-2771871b7bb6)  
28. 13 Best OSINT (Open Source Intelligence) Tools for 2025 \[UPDATED\] \- Talkwalker, Zugriff am Februar 28, 2026, [https://www.talkwalker.com/blog/best-osint-tools](https://www.talkwalker.com/blog/best-osint-tools)  
29. Top 7 Open Source Intelligence Tools Compared: Features, APIs, and Real-World Lessons, Zugriff am Februar 28, 2026, [https://heunify.com/content/product/top-7-open-source-intelligence-tools-compared-features-apis-and-real-world-lessons](https://heunify.com/content/product/top-7-open-source-intelligence-tools-compared-features-apis-and-real-world-lessons)  
30. Professional Standard plan vs Professional Advanced plan \- Maltego, Zugriff am Februar 28, 2026, [https://www.maltego.com/professional-standard-vs-professional-advanced-plan/](https://www.maltego.com/professional-standard-vs-professional-advanced-plan/)  
31. Maltego Products and Plans, Zugriff am Februar 28, 2026, [https://support.maltego.com/en/support/solutions/articles/15000036759-maltego-products-and-plans](https://support.maltego.com/en/support/solutions/articles/15000036759-maltego-products-and-plans)  
32. License Agreement \- Maltego, Zugriff am Februar 28, 2026, [https://www.maltego.com/license-agreement/](https://www.maltego.com/license-agreement/)  
33. Maltego Pricing, Zugriff am Februar 28, 2026, [https://www.maltego.com/pricing/](https://www.maltego.com/pricing/)  
34. Download \- Gephi \- The Open Graph Viz Platform, Zugriff am Februar 28, 2026, [https://gephi.org/desktop/](https://gephi.org/desktop/)  
35. Gephi | Bellingcat's Online Investigation Toolkit \- GitBook, Zugriff am Februar 28, 2026, [https://bellingcat.gitbook.io/toolkit/more/all-tools/gephi](https://bellingcat.gitbook.io/toolkit/more/all-tools/gephi)  
36. Data Driven Intelligence: OSINT Social Network and Graph Analytics \- Digital Marketplace, Zugriff am Februar 28, 2026, [https://www.applytosupply.digitalmarketplace.service.gov.uk/g-cloud/services/721084300216677](https://www.applytosupply.digitalmarketplace.service.gov.uk/g-cloud/services/721084300216677)  
37. Gephi \- The Open Graph Viz Platform, Zugriff am Februar 28, 2026, [https://gephi.org/](https://gephi.org/)  
38. gephi/LICENSE.md at master \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/gephi/gephi/blob/master/LICENSE.md](https://github.com/gephi/gephi/blob/master/LICENSE.md)  
39. owasp-amass/amass: In-depth attack surface mapping and ... \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/owasp-amass/amass](https://github.com/owasp-amass/amass)  
40. Projects \- OWASP Foundation, Zugriff am Februar 28, 2026, [https://owasp.org/projects/](https://owasp.org/projects/)  
41. theHarvester is a tool for gathering e-mail accounts, subdomain names, virtual hosts, open ports/ banners, and employee names from different public sources (search engines, pgp key servers). \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/xmppadmin/theHarvester](https://github.com/xmppadmin/theHarvester)  
42. laramies/theHarvester: E-mails, subdomains and names Harvester \- OSINT \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/laramies/theHarvester](https://github.com/laramies/theHarvester)  
43. 3 OSINT tools every officer should master now, Zugriff am Februar 28, 2026, [https://www.police1.com/investigations/3-osint-tools-every-officer-should-master-now](https://www.police1.com/investigations/3-osint-tools-every-officer-should-master-now)  
44. theharvester | Kali Linux Tools, Zugriff am Februar 28, 2026, [https://www.kali.org/tools/theharvester/](https://www.kali.org/tools/theharvester/)  
45. TheHarvester Information Gathering Guide and Examples \- Sohvaxus, Zugriff am Februar 28, 2026, [https://sohvaxus.github.io/content/theharvester.html](https://sohvaxus.github.io/content/theharvester.html)  
46. OSINT tools can be used for Information gathering, Cybersecurity, Reverse searching, bugbounty, trust and safety, red team operations and more. \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/joe-shenouda/osint-tools](https://github.com/joe-shenouda/osint-tools)  
47. thewhiteh4t/FinalRecon: All In One Web Recon \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/thewhiteh4t/FinalRecon](https://github.com/thewhiteh4t/FinalRecon)  
48. enigma-edu/FinalRecon-1: OSINT Tool for All-In-One Web Reconnaissance \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/enigma-edu/FinalRecon-1](https://github.com/enigma-edu/FinalRecon-1)  
49. MIT license \- thewhiteh4t/FinalRecon \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/thewhiteh4t/FinalRecon/blob/master/LICENSE](https://github.com/thewhiteh4t/FinalRecon/blob/master/LICENSE)  
50. sherlock-project/sherlock: Hunt down social media ... \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/sherlock-project/sherlock](https://github.com/sherlock-project/sherlock)  
51. soxoj/maigret: 🕵️‍♂️ Collect a dossier on a person by ... \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/soxoj/maigret](https://github.com/soxoj/maigret)  
52. My Top 10 OSINT Tools for Nickname Investigation | by Igor S. Bederov | Medium, Zugriff am Februar 28, 2026, [https://medium.com/@ibederov\_en/my-top-10-osint-tools-for-nickname-investigation-40e292fa5c84](https://medium.com/@ibederov_en/my-top-10-osint-tools-for-nickname-investigation-40e292fa5c84)  
53. How to automate the analysis of links to user profiles obtained with nickname enumeration tools | by cyb\_detective | OSINT Ambition, Zugriff am Februar 28, 2026, [https://publication.osintambition.org/how-to-automate-the-analysis-of-links-to-user-profiles-obtained-with-nickname-enumeration-tools-1d0abaf22c53](https://publication.osintambition.org/how-to-automate-the-analysis-of-links-to-user-profiles-obtained-with-nickname-enumeration-tools-1d0abaf22c53)  
54. holehe.ipynb \- bellingcat/open-source-research-notebooks \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/bellingcat/open-source-research-notebooks/blob/main/notebooks/community/holehe.ipynb](https://github.com/bellingcat/open-source-research-notebooks/blob/main/notebooks/community/holehe.ipynb)  
55. megadose/holehe-maltego \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/megadose/holehe-maltego](https://github.com/megadose/holehe-maltego)  
56. holehe/LICENSE.md at master \- Megadose \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/megadose/holehe/blob/master/LICENSE.md](https://github.com/megadose/holehe/blob/master/LICENSE.md)  
57. holehe allows you to check if the mail is used on different sites like twitter, instagram and will retrieve information on sites with the forgotten password function. \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/megadose/holehe](https://github.com/megadose/holehe)  
58. Social Mapper \- Correlate social media profiles with facial recognition, Zugriff am Februar 28, 2026, [https://www.cyberdefensemagazine.com/social-mapper-correlate-social-media-profiles-with-facial-recognition/](https://www.cyberdefensemagazine.com/social-mapper-correlate-social-media-profiles-with-facial-recognition/)  
59. Social Mapper: A free tool for automated discovery of targets' social media accounts, Zugriff am Februar 28, 2026, [https://www.helpnetsecurity.com/2018/08/10/automated-discovery-social-media-accounts/](https://www.helpnetsecurity.com/2018/08/10/automated-discovery-social-media-accounts/)  
60. New Facial Recognition Tool Tracks Targets Across Social Networks | The Verge, Zugriff am Februar 28, 2026, [https://mediawell.ssrc.org/news-items/new-facial-recognition-tool-tracks-targets-across-social-networks-the-verge/](https://mediawell.ssrc.org/news-items/new-facial-recognition-tool-tracks-targets-across-social-networks-the-verge/)  
61. social-coding/content/software-licensing.md at main \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/coderefinery/social-coding/blob/main/content/software-licensing.md](https://github.com/coderefinery/social-coding/blob/main/content/software-licensing.md)  
62. sundowndev/phoneinfoga: Information gathering ... \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/sundowndev/phoneinfoga](https://github.com/sundowndev/phoneinfoga)  
63. cree.py v1.1 Geolocation OSINT tool tutorial \- YouTube, Zugriff am Februar 28, 2026, [https://www.youtube.com/watch?v=SU4t-w1SUHc](https://www.youtube.com/watch?v=SU4t-w1SUHc)  
64. Creepy \- Active Defense Harbinger Distribution, Zugriff am Februar 28, 2026, [https://adhdproject.github.io/\#\!Tools/Attribution/Creepy.md](https://adhdproject.github.io/#!Tools/Attribution/Creepy.md)  
65. ilektrojohn/creepy: A geolocation OSINT tool. Offers geolocation information gathering through social networking platforms. \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/ilektrojohn/creepy](https://github.com/ilektrojohn/creepy)  
66. ExifTool by Phil Harvey, Zugriff am Februar 28, 2026, [https://exiftool.org/](https://exiftool.org/)  
67. A Black Box Comparison of Machine Learning Reverse Image Search for Cybersecurity OSINT Applications \- MDPI, Zugriff am Februar 28, 2026, [https://www.mdpi.com/2079-9292/12/23/4822](https://www.mdpi.com/2079-9292/12/23/4822)  
68. ExifTool \- Wikipedia, Zugriff am Februar 28, 2026, [https://en.wikipedia.org/wiki/ExifTool](https://en.wikipedia.org/wiki/ExifTool)  
69. Using FOCA for OSINT Document Metadata Analysis \- Rae Baker: Deep Dive, Zugriff am Februar 28, 2026, [https://www.raebaker.net/blog/2020/09/30/using-foca-for-osint-document-metadata-analysis](https://www.raebaker.net/blog/2020/09/30/using-foca-for-osint-document-metadata-analysis)  
70. Collection of Metadata from websites with FOCA | by Vasileiadis A. (Cyberkid) \- Medium, Zugriff am Februar 28, 2026, [https://medium.com/@redfanatic7/collection-of-metadata-from-websites-with-foca-b86e3082d337](https://medium.com/@redfanatic7/collection-of-metadata-from-websites-with-foca-b86e3082d337)  
71. ElevenPaths/FOCA: Tool to find metadata and hidden information in the documents. \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/ElevenPaths/FOCA](https://github.com/ElevenPaths/FOCA)  
72. mxrch/GHunt: 🕵️‍♂️ Offensive Google framework. \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/mxrch/ghunt](https://github.com/mxrch/ghunt)  
73. Licensing a repository \- GitHub Docs, Zugriff am Februar 28, 2026, [https://docs.github.com/articles/licensing-a-repository](https://docs.github.com/articles/licensing-a-repository)  
74. Mitaka \- Browser Extension For OSINT Search \- GeeksforGeeks, Zugriff am Februar 28, 2026, [https://www.geeksforgeeks.org/blogs/mitaka-browser-extension-for-osint-search/](https://www.geeksforgeeks.org/blogs/mitaka-browser-extension-for-osint-search/)  
75. ninoseki/mitaka: A browser extension for OSINT search \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/ninoseki/mitaka](https://github.com/ninoseki/mitaka)  
76. alex14324/mitaka \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/alex14324/mitaka](https://github.com/alex14324/mitaka)  
77. mitaka/LICENSE at master \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/ninoseki/mitaka/blob/master/LICENSE](https://github.com/ninoseki/mitaka/blob/master/LICENSE)  
78. Mitaka version history \- 25 versions – Add-ons for Firefox (en-US), Zugriff am Februar 28, 2026, [https://addons.mozilla.org/en-US/firefox/addon/mitaka/versions/](https://addons.mozilla.org/en-US/firefox/addon/mitaka/versions/)  
79. jwbizz08/OSINT-Projects-and-Tools: A collection of tools, scripts, and projects for open-source intelligence (OSINT). This repository includes automation scripts, data collection tools, visualization dashboards, and practical examples for OSINT research and analysis. \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/jwbizz08/OSINT-Projects-and-Tools](https://github.com/jwbizz08/OSINT-Projects-and-Tools)  
80. OnionScan is a free and open source tool for investigating the Dark Web. \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/s-rah/onionscan](https://github.com/s-rah/onionscan)  
81. awareseven/OSINTsources: This is a repo containing several osint sources \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/awareseven/OSINTsources](https://github.com/awareseven/OSINTsources)  
82. onionscan/LICENSE at master · s-rah/onionscan · GitHub, Zugriff am Februar 28, 2026, [https://github.com/s-rah/onionscan/blob/master/LICENSE](https://github.com/s-rah/onionscan/blob/master/LICENSE)  
83. exiftool/LICENSE at master \- GitHub, Zugriff am Februar 28, 2026, [https://github.com/exiftool/exiftool/blob/master/LICENSE](https://github.com/exiftool/exiftool/blob/master/LICENSE)