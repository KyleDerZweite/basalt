# Basalt OSINT toolkit research

Observed on November 15, 2026. This document records research and open decisions. It does not authorize features, a rewrite, removal of the web application, or a license change.

## Direction and recommendation

The intended product is a broad OSINT toolkit with one consistent way to find and use the required tools. Account discovery is one task within that scope. Existing choices about zero-auth sources, handwritten CSS, language, and interface are inputs to review rather than requirements for the future product.

Recommendation: decide the operating model before changing languages. Distinguish source collection, specialist-tool execution, evidence normalization, and analysis. The research should establish which responsibilities Basalt should own and which established tools it should use. This is a product and maintenance decision, not a module-count contest.

## Repository consolidation

The maintenance branch was fast-forwarded to the fetched `origin/main`. Their file trees were already identical. Previously committed profiles, evidence-list UI, relationship handling, and dependency changes are in the default branch; there is no separate local code patch to integrate.

Removed the project `AGENTS.md` and its `CLAUDE.md` symlink as requested. Removed the completed April and July audit lists and the superseded March v2 design and execution documents. Git retains those tracked documents. Updated the contribution guide to point to the current operating documentation instead of the deleted agent file.

Removed the untracked handoff and the two untracked Dzumas source lists at the user's explicit request. Those files are not part of this research's evidence. The old module-ideas inventory is retired rather than treated as an approved backlog. The separate map-integration note remains a deferred research idea.

As observed through GitHub on November 15, seven dependency PRs remain open: [Vite #36](https://github.com/KyleDerZweite/basalt/pull/36), [React #38](https://github.com/KyleDerZweite/basalt/pull/38), [Vite React plugin #39](https://github.com/KyleDerZweite/basalt/pull/39), [Pretext #40](https://github.com/KyleDerZweite/basalt/pull/40), [React Router #41](https://github.com/KyleDerZweite/basalt/pull/41), [Lucide #42](https://github.com/KyleDerZweite/basalt/pull/42), and [SQLite #43](https://github.com/KyleDerZweite/basalt/pull/43). They are separate dependency work, not uncommitted local changes. This cleanup does not merge, close, or absorb them.

## Basalt's current operating model

The [CLI scan command](../cli/cmd/scan.go) and [HTTP API](../cli/internal/api/server.go) share the [application service](../cli/internal/app/service.go). The service owns scans, cancellation, SQLite persistence, and events. The browser is a client of the same backend.

The [module contract](../cli/internal/modules/module.go) accepts a graph node and a concrete HTTP client and returns graph nodes and edges. [Registration](../cli/internal/app/modules.go) imports and constructs Go packages explicitly. This is a compiled extractor system, not runtime plugin discovery or a general external-tool runner. The [walker](../cli/internal/walker/walker.go) dispatches typed nodes and controls concurrency, timeouts, deduplication, and progress.

Username, email, and domain seeds are supported. Existing sources include identity platforms, DNS/certificate transparency, WHOIS, Shodan, Wayback, and IP information. The [graph types](../cli/internal/graph/node.go) also describe phone, organization, and other entities. A defined type does not establish a complete workflow for it.

For a broader toolkit, the question is whether a graph-node extractor is the right contract for every task. File metadata and external executable jobs can have different inputs and outputs. This observation does not authorize adding those capabilities.

## Frameworks and investigation platforms

| Tool | How it operates | Access, persistence, and license | Lesson for Basalt |
| --- | --- | --- | --- |
| [SpiderFoot](https://github.com/smicallef/spiderfoot) | Python plugins consume and emit named events. CLI and web scans can recursively discover related information. | SQLite stores scans, results, and logs. Events retain parent, source module, time, and confidence. MIT engine; sources can require credentials or subscriptions. | A broad collector can keep discovery lineage independently of its UI. |
| [Recon-ng](https://github.com/lanmaster53/recon-ng/wiki/Features) | Operator selects marketplace modules inside a workspace. Modules query shared SQL tables and insert findings. Interactive CLI is central; a web application also exists. | Workspaces isolate database, configuration, reports, and files. Marketplace metadata lists dependencies and credentials. GPL-3.0 framework; marketplace modules need separate checks. | Make tool requirements and reusable results explicit. |
| [BBOT](https://www.blacklanternsecurity.com/bbot/Stable/scanning/events/) | Recursive engine with typed events, selectable modules, presets, output consumers, CLI, and Python integration. | Events carry parentage, source, tags, and scope. JSON scan artifacts retain results. AGPL-3.0; provider access remains separate. | Explicit input/output types and scope make automatic chaining understandable. |
| [Maltego](https://docs.maltego.com/support/solutions/articles/15000008829-glossary-for-maltego-graph-desktop-) | Analyst selects entities and applies Transforms. Machines automate sequences of graph operations. | Proprietary graph client and licensed integrations. Saved graphs do not inherently archive their underlying sources. Free and paid access have different [entitlements](https://www.maltego.com/pricing/). | Deliberate pivots can be more useful than indiscriminate recursion. |
| [IntelOwl](https://github.com/intelowlproject/IntelOwl) | File and observable analysis jobs use analyzers, pivots, connectors, ingestors, and playbooks. REST API, GUI, and clients share the platform. | Persistent analysis jobs and investigations. AGPL-3.0 with local, free, authenticated, and premium analyzers. | Collection, analysis, and export have distinct responsibilities. |
| [OpenCTI](https://docs.opencti.io/latest/deployment/overview/) | Threat-intelligence knowledge platform. Connectors import/enrich/export STIX objects; queues and workers process writes. GraphQL API and web UI support analysis. | Durable relationships, references, markings, authorship, and history. [Community Apache-2.0 and separate Enterprise terms](https://docs.opencti.io/latest/administration/enterprise/). | Attribution and durable knowledge matter, but a threat ontology does not cover general OSINT by itself. |

SpiderFoot's [plugin contract](https://raw.githubusercontent.com/smicallef/spiderfoot/master/spiderfoot/plugin.py) declares watched and produced event names. Its [event model](https://raw.githubusercontent.com/smicallef/spiderfoot/master/spiderfoot/event.py) records discovery context. Recon-ng's [development guide](https://github.com/lanmaster53/recon-ng/wiki/Development-Guide) describes SQL-fed modules, a different form of composition. BBOT's [event documentation](https://github.com/blacklanternsecurity/bbot/blob/stable/docs/scanning/events.md) describes typed routing and output consumers. These are different operating models, not interchangeable plugin formats.

IntelOwl and OpenCTI are useful specialist references for enrichment and knowledge management. They should not define Basalt's whole scope. IntelOwl's [DataModels documentation](https://intelowlproject.github.io/docs/IntelOwl/usage/#datamodels) and current releases can differ in how much normalized data drives further operations. OpenCTI's [connector contract](https://docs.opencti.io/latest/development/connectors/) is built around threat-intelligence objects and platform jobs. Neither establishes a universal interface for all OSINT tasks.

## Specialist tools by task

| Task and tool | Operating mechanism | Validation and composition | Declared license |
| --- | --- | --- | --- |
| Username discovery, [Sherlock](https://github.com/sherlock-project/sherlock) | Python HTTP engine plus JSON site rules for URLs, markers, status, and redirect handling. | Explicit unknown/WAF states; successful requests alone do not establish identity. CLI exports include text and CSV. Core checks do not need a universal API key. | MIT |
| Profile discovery, [Maigret](https://github.com/soxoj/maigret) | Python checker and extractor plus a site database with positive/negative markers and examples. | Structured exports and reports. Site-specific inference can become stale. Core discovery and optional credentialed analysis have different access requirements. | MIT |
| Username detection data, [WhatsMyName](https://github.com/WebBreacher/WhatsMyName) | Dataset and schema with check/display URLs, response codes, markers, and examples. It is not a complete engine. | Consumer must supply networking, classification, failures, and output. Archived November 9, 2026, with a final 691-site dataset. | [CC BY-SA 4.0 dataset](https://github.com/WebBreacher/WhatsMyName/blob/main/LICENSE.md) |
| Email association, [Holehe](https://github.com/megadose/holehe) | Service-specific Python modules interpret account-related responses, including recovery flows. | Structured existence and rate-limit results are inferences, not proof of ownership. CLI and importable modules. Maintenance and provider behavior need fresh checks. | [GPL-3.0](https://github.com/megadose/holehe/blob/master/LICENSE.md) |
| Domain/organization collection, [theHarvester](https://github.com/laramies/theHarvester/blob/master/README.md) | Provider collectors query search, CT, DNS, repositories, and intelligence sources. Current source describes CLI, REST API, and local review. | Keyless and credentialed sources. JSONL/SQLite retain richer evidence than compatibility exports. Passive collection and optional active actions are distinct. | GPL-2.0-only as declared in README |
| Asset mapping, [Amass](https://github.com/owasp-amass/amass/blob/main/README.md) | Public-source collection and active reconnaissance use data sources, transformations, an asset database, and the Open Asset Model. | Covers relationships beyond subdomain names. Interfaces and deployment requirements depend on the selected version. | Apache-2.0 with separately licensed components |
| Passive subdomains, [Subfinder](https://github.com/projectdiscovery/subfinder/blob/dev/README.md) | Go provider adapters and YAML execution/credential configuration. | JSONL source attribution, stdin/stdout, and Go library. Optional resolution/wildcard handling is distinct from passive discovery. Many providers need keys. | [MIT](https://github.com/projectdiscovery/subfinder/blob/dev/LICENSE.md) |
| Website extraction, [Photon](https://github.com/s0md3v/Photon) | Python crawler extracts links, files, JavaScript references, email addresses, subdomains, and regex matches. | Runtime depth, scope, supplied cookies/headers, JSON/CSV output. Extracted references do not prove that an account or endpoint exists. | GPL-3.0 |
| Local file metadata, [ExifTool](https://github.com/exiftool/exiftool/blob/master/README) | Perl library and CLI with extensive format parsers and tag definitions. | JSON/CSV, batches, and persistent-process operation. No service account needed for local extraction. Metadata values do not authenticate their author. | Same terms as Perl, Artistic License or GPL |

Sherlock's [checker](https://github.com/sherlock-project/sherlock/blob/master/sherlock_project/sherlock.py) and [site loader](https://github.com/sherlock-project/sherlock/blob/master/sherlock_project/sites.py) show the separation between engine and data. Maigret's [site database](https://github.com/soxoj/maigret/blob/main/maigret/resources/data.json) and WhatsMyName's [schema](https://github.com/WebBreacher/WhatsMyName/blob/main/wmn-data-schema.json) demonstrate configurable positive and negative checks. Schema validity does not establish detector accuracy against a live provider.

A generated candidate, a source observation, a successful existence check, and an identity attribution are different claims. For infrastructure, historical discovery, current DNS resolution, service reachability, and organizational ownership are also different claims. No reviewed tool's platform count establishes accuracy or complete OSINT coverage.

WhatsMyName's closure is a concrete maintenance concern, not merely a missing release. The archived dataset remains a useful reference but is not a maintained dependency. Holehe's [repository metadata](https://api.github.com/repos/megadose/holehe) reported its latest push in September 2024 when checked. Other projects' recent commits likewise do not prove individual detectors work. ExifTool's [changelog](https://github.com/exiftool/exiftool/blob/master/Changes) distinguishes development and production releases, so default-branch documentation is not necessarily the packaged behavior.

## What Rust examples actually demonstrate

[RustScan](https://github.com/bee-san/RustScan/blob/master/README.md) combines native port discovery with Nmap and external scripts. It demonstrates coordination of specialist tools rather than reproducing every capability. Its [manifest](https://github.com/bee-san/RustScan/blob/master/Cargo.toml) declares GPL-3.0-only.

[feroxbuster](https://github.com/epi052/feroxbuster) is a Rust web-content discovery tool with configuration, structured output, and response filtering. It demonstrates the validation work behind a network tool, including soft-404 filtering. Its [manifest](https://github.com/epi052/feroxbuster/blob/main/Cargo.toml) declares MIT.

Both concern active reconnaissance. They are engineering references, not passive OSINT equivalents or evidence that Rust supplies replacements for the full tool inventory.

## Product choices the research leaves open

The strongest candidate for later evaluation is a hybrid: Basalt owns a consistent user experience and evidence handling, uses selected native collectors, and integrates specialist engines where reproducing them has little value. This is an assessment, not a selected architecture or adapter backlog.

A fully native implementation offers direct control and simpler runtime packaging. It also inherits site detection, API churn, DNS validation, crawling, file-format parsing, and authentication work. Wrappers retain upstream expertise but introduce executable installation, version compatibility, process cancellation, output parsing, and failure translation. Neither approach removes source subscriptions or data-use restrictions.

The next product discussion should settle these questions before any implementation:

- Does "one tool" mean one self-contained binary, or one consistent interface that can manage external dependencies?
- Should the normal workflow run a selected task or automatically expand discoveries? What scope and evidence should authorize another pivot?
- Are credentialed and paid sources optional parts of the toolkit? The former zero-auth preference is not assumed to remain a requirement.
- Is the primary result an automation-friendly export, an interactive evidence workspace, or both? That answer should decide CLI, TUI, and web priorities.
- Which concrete requirement would make a Rust migration worth the compatibility and validation work?
- Does broad downstream reuse justify MIT or Apache-2.0 after the rights review?

This survey is representative, not exhaustive. Corporate registries, geospatial analysis, media verification, and other OSINT disciplines need additional task-specific research before claiming broad coverage. "Ultimate" describes the ambition; it is not a verified capability of the current application.

## Stack and interface choices

Keep Go pending evidence of a requirement it cannot meet. A Rust rewrite would port module behavior, the walker, SQLite compatibility, cancellation, security controls, commands, and tests. Rust's [ownership model](https://doc.rust-lang.org/book/ch04-00-understanding-ownership.html) avoids garbage collection; [Tokio](https://tokio.rs/tokio/tutorial) provides async infrastructure with explicit [shutdown handling](https://tokio.rs/tokio/topics/shutdown). Go already provides [context cancellation](https://pkg.go.dev/context) and [asset embedding](https://pkg.go.dev/embed). No benchmark or runtime measurement in this research establishes a Rust performance benefit for Basalt.

A migration decision should name its actual benefit, such as an unmet memory constraint, a required Rust integration, or a maintenance preference that justifies porting existing behavior. [clap](https://docs.rs/clap/latest/clap/) supplies CLI parsing, not command compatibility or the OSINT engine.

CLI-only would remove frontend build and browser maintenance, but also remove interactive evidence inspection. A TUI is another application to build and maintain. [Ratatui backends](https://ratatui.rs/concepts/backends/) and [event handling](https://ratatui.rs/concepts/event-handling/) address terminal mechanics, not a finished investigation workflow. A TUI does not inherently require rewriting the backend or using Rust.

The current [React client](../web/src/lib/api.ts) consumes JSON and [SSE events](../web/src/hooks/useScanEvents.ts). [Vite](../web/vite.config.ts) outputs assets into the embedded Go directory. These contracts allow interface and backend choices to be evaluated separately. Keeping the web UI is a recommendation pending a task-based comparison, not a settled product constraint.

## License decision

The [repository license](../LICENSE) is AGPLv3, and inspected Go headers use `AGPL-3.0-or-later`. The contribution guide licenses submissions under AGPLv3; it does not expressly assign copyright or grant unrestricted relicensing authority. Git author names, including bot and tool identities, do not prove ownership or assignments.

[AGPLv3](https://www.gnu.org/licenses/agpl-3.0.html) addresses distribution in sections 5 and 6 and modified network-interactive versions in section 13. [MIT](https://opensource.org/license/mit) permits broad reuse with preservation of its copyright and permission notices. [Apache-2.0](https://www.apache.org/licenses/LICENSE-2.0.html) includes explicit patent terms and redistribution obligations for notices and modified files. These summaries are comparisons of the license texts, not a determination of the repository's relicensing eligibility.

A permissive license is worth considering if broad reuse and embedding are priorities. MIT is the simpler candidate; Apache-2.0 adds explicit patent terms. Before changing either the license file or source headers, establish rights for retained contributions, copied source, datasets, fonts, bundled assets, and dependencies. [FSF guidance](https://www.fsf.org/blogs/licensing/relicensing-versus-compatibility) distinguishes compatibility from authority to relicense. Fetching that page and the GNU AGPL web page timed out during verification; the local AGPL text and the successfully retrieved MIT and Apache texts were available.

Copying source, linking libraries, invoking executables, and redistributing binaries are different integration choices. GNU's [aggregation FAQ](https://www.gnu.org/licenses/gpl-faq.en.html#MereAggregation) considers both the communication mechanism and its semantics. A subprocess is not a blanket exemption from licensing obligations. No license change was made.

## Evidence limits

Research uses official documentation and repository inspection. Tools were not installed, benchmarked, or run against people or third-party targets. Advertised platform counts, maintenance activity, and detection behavior are not independent accuracy measurements. Authentication, costs, availability, and licenses can change and must be rechecked for a concrete integration.
