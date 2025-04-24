# PoC DSL Structure

This document explains the structure of a PoC file and its components.

## Overall Structure

A PoC file consists of these main sections:

```yaml
poc:
  metadata: {...}         # Information about the PoC
  context: {...}          # Defines resources, variables, users, etc.
  setup: [...]            # GIVEN phase - setup steps
  exploit_scenario: {...} # WHEN phase - actual exploit
  assertions: [...]       # THEN phase - immediate checks
  verification: [...]     # Optional impact verification
```

## Section Descriptions

### Metadata
Descriptive information about the PoC and vulnerability.

```yaml
metadata:
  id: "AppName-VulnType-Date"           # Unique identifier
  title: "Descriptive title"            # Human-readable title
  dsl_version: "1.0"                    # Version of DSL
  source_report:                        # Source of vulnerability
    platform: "HackerOne"               # Reporting platform
    id: "123456"                        # Report ID
    reported_date: "2025-01-15"         # Date reported
  vulnerability:                        # Vulnerability details
    type: "Improper Access Control"     # Type of vulnerability
    cwe: "CWE-284"                      # CWE reference
  target_application:                   # Target details
    name: "Example App"                 # Application name
    version: "1.2.3"                    # Affected version
  severity: "High"                      # Impact severity
  tags: ["api", "authenticated"]        # Categorization tags
```

### Context
Defines the necessary environment, actors, resources, files, and variables.

```yaml
context:
  description: "Prerequisites for this PoC"
  users:                                # User roles
    - id: "attacker_user"               # User identifier
      description: "Logged-in attacker" # Description
      credentials_ref: "attacker_creds" # External credentials reference
  resources:                            # Resources needed
    - id: "victim_prompt"               # Resource identifier
      description: "Target prompt to delete"
      type: "prompt"                    # Type hint
  environment:                          # Target environment
    - id: "target_base_url"             # Environment variable ID
      value: "https://example.com"      # Value
  files:                                # Required files
    - id: "payload_file"                # File identifier
      local_path: "./payloads/xss.js"   # Path on local system
  variables:                            # Dynamic variables
    - id: "csrf_token"                  # Variable ID
      value: null                       # Initial value (will be populated later)
```

### Setup (GIVEN)
Global setup steps executed once before the exploit scenario.

```yaml
setup:
  - step: 1
    dsl: "Ensure attacker user exists and is authenticated"
    action:
      type: "ensure_users_exist"
      users: ["attacker_user"]
  - step: 2
    dsl: "Ensure victim resource exists"
    action:
      type: "ensure_resource_exists"
      resource: "victim_prompt"
      output_variable: "resource_id"
```

### Exploit Scenario (WHEN)
Core attack steps, potentially with scenario-specific setup/teardown.

```yaml
exploit_scenario:
  name: "Attacker bypasses access control to delete victim's data"
  setup:
    - step: 1
      dsl: "Login as attacker"
      action:
        type: "authenticate"
        user: "attacker_user"
        output_variable: "auth_token"
  steps:
    - step: 1
      dsl: "Attacker sends delete request with victim's resource ID"
      action:
        type: "http_request"
        request:
          method: "DELETE"
          url: "{{ target_base_url }}/api/prompts/{{ resource_id }}"
          headers:
            Authorization: "Bearer {{ auth_token }}"
  teardown:
    - step: 1
      dsl: "Logout attacker account"
      action:
        type: "http_request"
        request:
          method: "POST"
          url: "{{ target_base_url }}/logout"
```

### Assertions (THEN)
Immediate checks performed after exploit steps to confirm vulnerability.

```yaml
assertions:
  - step: 1
    dsl: "Verify delete succeeded with 200 status"
    check:
      type: "http_status"
      response: "{{ last_http_response }}"
      expected: 200
  - step: 2
    dsl: "Verify response indicates success"
    check:
      type: "http_body"
      response: "{{ last_http_response }}"
      contains: ["success"]
      not_contains: ["error", "permission denied"]
```

### Verification
Optional steps to confirm long-term impact or persistence.

```yaml
verification:
  - step: 1
    dsl: "Verify resource is actually deleted by checking it no longer exists"
    action:
      type: "http_request"
      request:
        method: "GET"
        url: "{{ target_base_url }}/api/prompts/{{ resource_id }}"
    check:
      type: "http_status"
      response: "{{ last_http_response }}"
      expected: 404
```

## Step Structure

Steps are the basic building blocks across all phases:

```yaml
- step: 1  # Numeric identifier
  dsl: "Human-readable description"
  id: "optional_step_id"  # For references
  if: "{{ condition }}"   # Optional condition
  loop:                   # Optional loop
    variable: "item"
    collection: "{{ items }}"
  action:                 # Either action OR check
    type: "action_type"
    # Action-specific parameters
  check:                  # Either check OR action
    type: "check_type"
    # Check-specific parameters
  manual: false           # If true, requires manual intervention
```

A step may contain either an action (performing an operation) or a check (verifying a condition), and can optionally include conditional logic (`if`) or loop constructs (`loop`).
