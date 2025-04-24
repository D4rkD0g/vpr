# Actions Reference

Actions are operations performed during PoC execution. This document details all supported action types and their parameters.

## HTTP Actions

### http_request

Sends HTTP requests to target systems.

```yaml
action:
  type: "http_request"
  request:
    method: "POST"  # GET, POST, PUT, DELETE, etc.
    url: "{{ target_base_url }}/api/endpoint"
    headers:
      Content-Type: "application/json"
      Authorization: "Bearer {{ user_token }}"
    body: |
      {"key": "value"}
  response_actions:
    - extract:
        target: "csrf_token"
        source: "body"
        using: "regex"
        pattern: "name=\"csrf\" value=\"([^\"]+)\""
```

Parameters:
- `request`: HTTP request details
  - `method`: HTTP method (GET, POST, PUT, DELETE, etc.)
  - `url`: Target URL (supports variable substitution)
  - `headers`: Key-value pairs for request headers
  - `body`: Request body content (string or object)
  - `form`: Form data (alternative to body)
  - `files`: File uploads
- `response_actions`: Optional operations on the response
  - `extract`: Extract data from response (see Extractors documentation)

## Data Generation

### generate_data

Generates dynamic data for use in subsequent steps.

```yaml
action:
  type: "generate_data"
  data:
    random_id: "{{ random(10) }}"
    timestamp: "{{ now() }}"
  output_variable: "generated"  # Stores to context.variables.generated
```

Parameters:
- `data`: Key-value pairs to generate
- `output_variable`: Variable to store results

### wait

Pauses execution for a specified duration.

```yaml
action:
  type: "wait"
  duration_ms: 5000  # Wait 5 seconds
```

Parameters:
- `duration_ms`: Time to wait in milliseconds

## Authentication Actions

### authenticate

Handles authentication flows for specified users.

```yaml
action:
  type: "authenticate"
  user: "attacker_user"
  method: "form"  # or "oauth", "basic", "custom"
  credentials: "{{ attacker_user.credentials }}"
  output_variable: "auth_result"
```

Parameters:
- `user`: Reference to user in context
- `method`: Authentication method
- `credentials`: Credentials reference or object
- `output_variable`: Variable to store auth results (tokens, cookies)

## Setup Actions

### ensure_users_exist

Creates or confirms user accounts needed for the PoC.

```yaml
action:
  type: "ensure_users_exist"
  users: ["attacker_user", "victim_user"]
```

Parameters:
- `users`: List of user IDs from context.users

### ensure_resource_exists

Creates or confirms resources needed for the PoC.

```yaml
action:
  type: "ensure_resource_exists"
  resource: "victim_prompt"
  user_context: "victim_user"
  output_variable: "resource_id"
```

Parameters:
- `resource`: Resource ID from context.resources
- `user_context`: Optional user context for creation
- `output_variable`: Variable to store resource identifier

## System Actions

### execute_local_command

Runs commands on the local system (requires security controls).

```yaml
action:
  type: "execute_local_command"
  command: "curl -s {{ target_url }} > output.txt"
  working_directory: "./temp"
  timeout_ms: 10000
  output_variable: "command_result"
```

Parameters:
- `command`: Command to execute
- `working_directory`: Directory to run in
- `timeout_ms`: Maximum execution time
- `output_variable`: Variable to store command output

## Control Flow

### set_variable

Explicitly sets a context variable value.

```yaml
action:
  type: "set_variable"
  variable: "attempt_count"
  value: 0
```

Parameters:
- `variable`: Variable ID to set
- `value`: New value

### branch

Takes different actions based on conditions.

```yaml
action:
  type: "branch"
  conditions:
    - condition: "{{ status_code == 200 }}"
      then:
        type: "set_variable"
        variable: "success"
        value: true
    - condition: "{{ status_code == 403 }}"
      then:
        type: "set_variable"
        variable: "permission_denied"
        value: true
  default:
    type: "set_variable"
    variable: "unknown_response"
    value: true
```

Parameters:
- `conditions`: List of condition/action pairs
- `default`: Action to take if no conditions match
