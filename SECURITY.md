# Security Policy

## Supported Versions

| main    | :white_check_mark: |

mtgo is currently in active development. Security fixes are applied to the `main` branch.

## Reporting a Vulnerability

**Do not report security vulnerabilities through public GitHub issues.**

Instead, please report them via:

- **GitHub Security Advisories**: [Report a vulnerability](https://github.com/mtgo-labs/mtgo/security/advisories/new)
- **Email**: Send a detailed report to the repository maintainers

Please include:

1. **Description** of the vulnerability
2. **Affected component** (e.g., `internal/crypto`, `telegram/client`, `internal/session`)
3. **Reproduction steps** or proof-of-concept
4. **Impact assessment** — what an attacker could achieve
5. **Suggested fix** (if you have one)

### Response Timeline

- **Acknowledgment**: Within 48 hours
- **Initial assessment**: Within 5 business days
- **Status updates**: Every 7 days until resolved
- **Fix**: Depends on severity and complexity

### Disclosure Policy

- We follow **responsible disclosure**.
- We request that you do not publicly disclose the vulnerability until a fix is available.
- We will credit reporters in the security advisory (unless you prefer to remain anonymous).

## Security Considerations

mtgo implements the MTProto 2.0 protocol for Telegram client communication. Key security areas:

- **Cryptographic operations** (`internal/crypto/`) — AES-IGE, RSA, DH key exchange
- **Session management** (`internal/session/`) — authorization keys, session persistence
- **Transport layer** (`internal/transport/`) — obfuscated connections, ABR

### Best Practices When Using mtgo

- Never hardcode `apiHash` or session credentials in source code
- Use environment variables or secure storage for sensitive configuration
- Keep session files (`.session`) protected — they contain authorization keys
- Rotate session files if you suspect compromise
- Do not expose the `Client` instance across untrusted boundaries

## Scope

### In Scope

- Vulnerabilities in mtgo's core library code
- Cryptographic implementation issues in `internal/crypto/`
- Authentication or session management flaws in `internal/session/`
- Transport layer issues in `internal/transport/`
- Input validation issues in the TL parser or type system

### Out of Scope

- Issues in Telegram's server-side infrastructure
- Social engineering attacks
- Denial of service via Telegram's API rate limiting
- Issues in dependencies (report upstream, but let us know too)
