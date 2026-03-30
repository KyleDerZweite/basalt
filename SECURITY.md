# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 0.2.x   | Yes       |
| < 0.2   | No        |

## Reporting a Vulnerability

If you discover a security vulnerability in Basalt, please report it responsibly.

**Do not open a public issue.** Instead, email security concerns to the maintainer or use [GitHub's private vulnerability reporting](https://github.com/KyleDerZweite/basalt/security/advisories/new).

Please include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

You can expect an initial response within 72 hours. We will work with you to understand the issue and coordinate a fix before any public disclosure.

## Scope

Basalt is a local CLI tool. Security concerns most relevant to this project include:

- **Command injection** -- unsanitized input passed to shell commands or system calls
- **SSRF** -- modules making requests to unintended internal targets
- **Credential leakage** -- API keys or tokens logged, cached, or written to output files
- **Path traversal** -- config/site file loading accessing unintended paths
- **Dependency vulnerabilities** -- known CVEs in Go dependencies

## Design Principles

- API keys are read from config files (`~/.basalt/config`), never hardcoded
- All HTTP requests go through the shared `httpclient` package with timeouts
- Module output is structured data (graph nodes/edges), not raw shell output
- No user input is passed to `exec` or shell commands
