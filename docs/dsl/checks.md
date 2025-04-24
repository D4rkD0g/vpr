# Checks Reference

Checks verify conditions during PoC execution. This document details all supported check types and their parameters.

## HTTP Response Checks

### http_status

Verifies HTTP response status codes.

```yaml
check:
  type: "http_status"
  expected: 200  # Can be single value or list [200, 201, 204]
  response: "{{ last_http_response }}"
  max_attempts: 3
  retry_interval: 1000  # milliseconds
```

Parameters:
- `expected`: Expected status code(s), can be a single value or an array
- `response`: Response object to check (typically from previous HTTP request)
- `max_attempts`: Optional retry count
- `retry_interval`: Delay between retries in milliseconds

### http_body

Verifies HTTP response body contents.

```yaml
check:
  type: "http_body"
  response: "{{ last_http_response }}"
  contains: ["success", "token"]  # Strings that must be present
  not_contains: ["error", "failed"]  # Strings that must NOT be present
  regex: "token: ([A-Za-z0-9]+)"  # Regex pattern to match
```

Parameters:
- `response`: Response object to check
- `contains`: Strings that must be present in the response body
- `not_contains`: Strings that must NOT be present in the response body
- `regex`: Regular expression pattern to match against the body
- `max_attempts`: Optional retry count
- `retry_interval`: Delay between retries in milliseconds

### http_header

Verifies HTTP response headers.

```yaml
check:
  type: "http_header"
  response: "{{ last_http_response }}"
  header: "Content-Type"  # Header name
  value: "application/json"  # Expected value
  regex: "^application/json.*"  # Alternative regex pattern
```

Parameters:
- `response`: Response object to check
- `header`: HTTP header name to verify
- `value`: Expected header value (exact match)
- `regex`: Alternative regular expression pattern for header value
- `max_attempts`: Optional retry count
- `retry_interval`: Delay between retries in milliseconds

## Variable Checks

### variable_equals

Compares variable value to expected value.

```yaml
check:
  type: "variable_equals"
  variable: "response_count"
  expected: 5
```

Parameters:
- `variable`: Variable from context to check
- `expected`: Expected value (supports basic types: string, number, boolean)
- `max_attempts`: Optional retry count
- `retry_interval`: Delay between retries in milliseconds

### variable_contains

Checks if variable contains a specific value.

```yaml
check:
  type: "variable_contains"
  variable: "user_response.items"
  value: "target_id"
  not_contains: ["error"]
```

Parameters:
- `variable`: Variable from context to check
- `value`: Value that should be contained (can check arrays or strings)
- `not_contains`: Values that should NOT be contained
- `max_attempts`: Optional retry count
- `retry_interval`: Delay between retries in milliseconds

## Resource Checks

### resource_exists

Verifies that a resource exists in the target system.

```yaml
check:
  type: "resource_exists"
  resource: "victim_prompt"
  user_context: "victim_user"
```

Parameters:
- `resource`: Resource ID from context.resources
- `user_context`: Optional user context for verification
- `max_attempts`: Optional retry count
- `retry_interval`: Delay between retries in milliseconds

### resource_not_exists

Verifies that a resource does NOT exist in the target system.

```yaml
check:
  type: "resource_not_exists"
  resource: "deleted_file"
  user_context: "victim_user"
```

Parameters:
- `resource`: Resource ID from context.resources
- `user_context`: Optional user context for verification
- `max_attempts`: Optional retry count
- `retry_interval`: Delay between retries in milliseconds

## Advanced Checks

### custom_check

Implements a custom check using a JavaScript expression.

```yaml
check:
  type: "custom_check"
  expression: "{{ response.body.count > 0 && response.body.items.length === response.body.count }}"
  context:
    response: "{{ last_http_response }}"
```

Parameters:
- `expression`: JavaScript expression that evaluates to boolean
- `context`: Variable mappings for the expression
- `max_attempts`: Optional retry count
- `retry_interval`: Delay between retries in milliseconds

## Error Handling

All checks support `expected_error` for handling expected failures:

```yaml
check:
  type: "http_status"
  expected: 200
  response: "{{ last_http_response }}"
  expected_error:
    message: "Status check failed after retries"
    severity: "warning"  # or "error", "info"
```

The `expected_error` object has these fields:
- `message`: Error message to display
- `severity`: Error severity level
- `continue`: If true, execution continues despite error
