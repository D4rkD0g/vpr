# Variables and Functions Reference

This document describes the variable substitution syntax used in the PoC DSL and the built-in functions available.

## Variable Substitution

The PoC DSL uses a double-curly brace syntax (`{{ ... }}`) for variable substitution and function calls. This allows dynamic values to be inserted into strings, objects, and parameters.

### Basic Variable Syntax

```yaml
# Simple variable reference
url: "{{ target_base_url }}/api/endpoint"

# Nested property access
user_id: "{{ context.users.attacker_user.id }}"

# Array indexing
first_item: "{{ items[0] }}"
```

Variables can be inserted into strings, used as entire property values, or embedded within complex objects.

### Variable Sources

Variables can come from several sources:

1. **Context Variables**: Defined in the `context` section or added via `set_variable` actions
2. **Step Results**: Results from previous steps (e.g., `last_http_response`)
3. **Loop Variables**: Values from the current loop iteration
4. **Extracted Data**: Values captured by extractors
5. **Function Results**: Return values from built-in functions

## Built-in Functions

Functions can be called within the variable substitution syntax:

```yaml
# Basic function call
random_value: "{{ random(8) }}"

# Function with multiple arguments
encoded: "{{ base64_encode(payload) }}"

# Function with complex arguments
trimmed: "{{ regex_replace(input, '\\s+', ' ') }}"
```

### String Functions

#### `concat(...args)`
Concatenates multiple strings.

```yaml
full_name: "{{ concat(first_name, ' ', last_name) }}"
```

#### `trim(str)`
Removes whitespace from the beginning and end of a string.

```yaml
clean: "{{ trim(input_value) }}"
```

#### `substring(str, start, end)`
Extracts a portion of a string.

```yaml
first_four: "{{ substring(token, 0, 4) }}"
```

#### `uppercase(str)`
Converts a string to uppercase.

```yaml
upper: "{{ uppercase(name) }}"
```

#### `lowercase(str)`
Converts a string to lowercase.

```yaml
lower: "{{ lowercase(name) }}"
```

#### `replace(str, pattern, replacement)`
Replaces all occurrences of a pattern with a replacement.

```yaml
fixed: "{{ replace(text, 'old', 'new') }}"
```

### Encoding Functions

#### `base64_encode(str)`
Encodes a string to base64.

```yaml
encoded: "{{ base64_encode(password) }}"
```

#### `base64_decode(str)`
Decodes a base64 string.

```yaml
decoded: "{{ base64_decode(encoded_data) }}"
```

#### `url_encode(str)`
URL-encodes a string.

```yaml
safe_url: "{{ url_encode(param) }}"
```

#### `url_decode(str)`
Decodes a URL-encoded string.

```yaml
decoded: "{{ url_decode(param) }}"
```

#### `json_encode(obj)`
Converts an object to a JSON string.

```yaml
json_body: "{{ json_encode(payload) }}"
```

#### `json_decode(str)`
Parses a JSON string into an object.

```yaml
parsed: "{{ json_decode(response_body) }}"
```

### Data Generation Functions

#### `random(length, [charset])`
Generates a random string of specified length.

```yaml
# Random alphanumeric string
id: "{{ random(10) }}"

# Random string with custom charset
pin: "{{ random(4, '0123456789') }}"
```

#### `uuid()`
Generates a random UUID.

```yaml
request_id: "{{ uuid() }}"
```

#### `now([format])`
Returns the current timestamp, optionally in the specified format.

```yaml
# ISO format (default)
timestamp: "{{ now() }}"

# Custom format
date: "{{ now('YYYY-MM-DD') }}"
```

#### `timestamp()`
Returns the current Unix timestamp (seconds since epoch).

```yaml
unix_time: "{{ timestamp() }}"
```

### Math Functions

#### `add(a, b)`
Adds two numbers.

```yaml
sum: "{{ add(value, 10) }}"
```

#### `subtract(a, b)`
Subtracts b from a.

```yaml
difference: "{{ subtract(total, discount) }}"
```

#### `multiply(a, b)`
Multiplies two numbers.

```yaml
product: "{{ multiply(price, quantity) }}"
```

#### `divide(a, b)`
Divides a by b.

```yaml
quotient: "{{ divide(total, count) }}"
```

#### `round(num, [precision])`
Rounds a number to the specified precision.

```yaml
rounded: "{{ round(value, 2) }}"
```

### Array Functions

#### `length(arr_or_str)`
Returns the length of an array or string.

```yaml
count: "{{ length(items) }}"
```

#### `join(arr, [separator])`
Joins array elements into a string.

```yaml
csv: "{{ join(values, ',') }}"
```

#### `split(str, separator)`
Splits a string into an array.

```yaml
parts: "{{ split(input, ',') }}"
```

### Conditional Functions

#### `if(condition, true_value, false_value)`
Returns one of two values based on a condition.

```yaml
status: "{{ if(count > 0, 'Found', 'Not found') }}"
```

#### `coalesce(...args)`
Returns the first non-null argument.

```yaml
value: "{{ coalesce(primary_value, backup_value, default_value) }}"
```

### Regular Expression Functions

#### `regex_match(str, pattern)`
Returns true if the string matches the pattern.

```yaml
is_valid: "{{ regex_match(email, '^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$') }}"
```

#### `regex_replace(str, pattern, replacement)`
Replaces text matching a pattern.

```yaml
clean_html: "{{ regex_replace(body, '<[^>]*>', '') }}"
```

#### `regex_extract(str, pattern, [group])`
Extracts text matching a pattern.

```yaml
domain: "{{ regex_extract(url, '^https?://([^/]+)', 1) }}"
```

## Function Chaining

Functions can be chained together:

```yaml
processed: "{{ base64_encode(trim(concat(prefix, value))) }}"
```

## Complex Expressions

More complex expressions can be constructed using JavaScript syntax within the variable substitution:

```yaml
# Arithmetic
calculated: "{{ price * quantity * (1 - discount) }}"

# Conditionals
display: "{{ count > 0 ? 'Items: ' + count : 'Empty' }}"

# Function composition
transformed: "{{ uppercase(trim(input)) }}"
```

Note: The full JavaScript expression capabilities depend on the implementation details of the PoC executor.
