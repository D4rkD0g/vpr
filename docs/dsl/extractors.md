# Extractors Reference

Extractors capture data from responses and store them in context variables for later use. They are typically used in the `response_actions` of an `http_request` action.

## Overview

Extractors allow you to:
- Extract data from HTTP responses (headers, body, status code)
- Parse structured data (JSON, XML, HTML)
- Match patterns using regular expressions
- Transform extracted values
- Store results in context variables

## Basic Syntax

```yaml
action:
  type: "http_request"
  request:
    # Request details
  response_actions:
    - extract:
        target: "variable_name"  # Variable to store result in
        source: "body"           # Where to extract from
        using: "extraction_type" # Method to use
        # Method-specific parameters
```

## Extraction Sources

The `source` parameter specifies where to extract data from:

- `body`: The full response body (default)
- `header.<name>`: A specific response header (e.g., `header.Content-Type`)
- `status`: The HTTP status code
- `response`: The entire response object

## Extraction Methods

### regex

Extract data using regular expressions.

```yaml
extract:
  target: "session_id"
  source: "body"
  using: "regex"
  pattern: "sessionId=([a-f0-9]+)"  # Capture group becomes the result
  all_matches: false  # If true, returns array of all matches
  match_group: 1  # Which regex capture group to use (default: 1)
```

Parameters:
- `pattern`: Regular expression with at least one capture group
- `all_matches`: If true, captures all matches (returns array)
- `match_group`: Which capture group to use (default: 1)

### jsonpath

Extract data using JSONPath expressions.

```yaml
extract:
  target: "user_ids"
  source: "body"
  using: "jsonpath"
  path: "$.users[*].id"  # JSONPath expression
```

Parameters:
- `path`: JSONPath expression (see [JSONPath syntax](https://goessner.net/articles/JsonPath/))

### xpath

Extract data from XML/HTML using XPath expressions.

```yaml
extract:
  target: "csrf_token"
  source: "body"
  using: "xpath"
  path: "//input[@name='csrf']/@value"  # XPath expression
  all_matches: false  # If true, returns array of all matches
```

Parameters:
- `path`: XPath expression
- `all_matches`: If true, captures all matches (returns array)

### css

Extract data from HTML using CSS selectors.

```yaml
extract:
  target: "page_title"
  source: "body"
  using: "css"
  selector: "h1.title"  # CSS selector
  attribute: "textContent"  # Element attribute to extract (default: textContent)
  all_matches: false  # If true, returns array of all matches
```

Parameters:
- `selector`: CSS selector
- `attribute`: Element attribute to extract
- `all_matches`: If true, captures all matches (returns array)

### split

Extract data by splitting a string.

```yaml
extract:
  target: "api_token"
  source: "header.Authorization"
  using: "split"
  delimiter: " "  # String to split on
  index: 1  # Index to extract (0-based)
```

Parameters:
- `delimiter`: String to split on
- `index`: Index of the part to extract (0-based)

## Transformation

You can transform extracted values before storing them:

```yaml
extract:
  target: "clean_token"
  source: "body"
  using: "regex"
  pattern: "token=([^&]+)"
  transform:
    - type: "trim"  # Trim whitespace
    - type: "replace"  # Replace characters
      pattern: "\\+"
      replacement: " "
    - type: "decode"  # URL decode
      encoding: "url"
```

Supported transformations:
- `trim`: Removes whitespace from beginning and end
- `replace`: String replacement (`pattern` and `replacement` parameters)
- `decode`: Decodes encoded strings (`encoding` parameter: `url`, `base64`, `html`)
- `encode`: Encodes strings (`encoding` parameter: `url`, `base64`)
- `substring`: Extracts substring (`start` and `end` parameters)
- `lowercase`: Converts to lowercase
- `uppercase`: Converts to uppercase

## Conditional Extraction

You can conditionally extract data:

```yaml
extract:
  target: "error_message"
  source: "body"
  if: "{{ last_http_response.status >= 400 }}"
  using: "jsonpath"
  path: "$.error.message"
```

Parameters:
- `if`: Condition for extraction (variable substitution supported)
