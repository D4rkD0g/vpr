# Credential Management

This document explains how VPR securely manages credentials for PoC execution.

## Core Principles

1. **Never store secrets in PoC files**: The DSL specification explicitly forbids storing actual credentials in the PoC YAML/JSON files.
2. **Use indirection for credentials**: All authentication should use credential references or placeholders.
3. **Separation of concerns**: Execution logic is separate from credential storage.
4. **Secure by default**: Default configuration should be secure with no credential exposure.

## Credential Reference Mechanism

The PoC DSL uses a reference system for credentials:

```yaml
context:
  users:
    - id: "admin_user"
      credentials_ref: "admin_user_credentials"
```

The `credentials_ref` value acts as a lookup key that the execution engine resolves externally.

## Credential Resolution Methods

VPR supports the following credential resolution methods:

### 1. Environment Variables

Variables following the pattern `VPR_CRED_{credential_ref}_{field}` are used to supply credential fields:

```bash
# For credentials_ref: "admin_user_credentials"
export VPR_CRED_admin_user_credentials_username=admin
export VPR_CRED_admin_user_credentials_password=s3cret
```

### 2. Credential Files

JSON files in a designated credentials directory:

```json
// ~/.vpr/credentials/admin_user_credentials.json
{
  "username": "admin",
  "password": "s3cret",
  "api_key": "1234567890abcdef"
}
```

### 3. External Secret Managers

Integration with external secret managers:

```yaml
# ~/.vpr/config.yaml
credential_providers:
  - type: "vault"
    url: "https://vault.example.com"
    auth_method: "token"
    path_prefix: "secret/vpr/"
  - type: "aws_secrets_manager"
    region: "us-west-2"
```

## Implementing Credential Resolvers

For developers extending VPR, implement the `CredentialResolver` interface:

```go
type CredentialResolver interface {
    ResolveCredential(ref string) (map[string]interface{}, error)
    SupportsCredential(ref string) bool
}
```

## Security Considerations

1. **Memory handling**: Credentials should be treated as sensitive in memory.
2. **Logging**: Ensure credentials are never logged, even in debug mode.
3. **Error messages**: Error messages should not expose credential values.
4. **Timeouts**: Consider auto-expiry of credentials after usage.
5. **Permissions**: Use least-privilege credentials for actions.

## Runtime Prompting

For interactive usage, VPR can prompt users for credentials at runtime:

```
> vpr run example.yaml --interactive-creds
Credential 'admin_user_credentials' required:
Username: admin
Password: *******
```

## Best Practices

1. Use dedicated test accounts for PoCs, never production credentials.
2. Rotate credentials regularly.
3. Use environment-specific credential sets.
4. Validate minimum required privileges before execution.
5. Consider creating time-limited credentials for PoC execution.
