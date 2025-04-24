// Package poc defines the Go data structures representing the v1.0 PoC DSL format.
// These structs map directly to the YAML/JSON structure defined in the specification,
// enabling parsing, validation (using jsonschema tags), and manipulation of PoC definitions.
// Key components include Metadata, Context (Users, Resources, etc.), Steps (Action/Check),
// and the overall Poc structure as defined in specification_v1.0.md.
package poc

// PocWrapper represents the top-level root containing a Poc definition
// This is necessary since YAML/JSON files typically have a top-level 'poc:' key
type PocWrapper struct {
	Poc Poc `yaml:"poc" json:"poc" jsonschema:"required"`
}

// Poc represents the top-level poc object as described in Section 3
type Poc struct {
	Metadata     Metadata        `yaml:"metadata" json:"metadata" jsonschema:"required"`
	Context      Context         `yaml:"context" json:"context" jsonschema:"required"`
	Setup        []Step          `yaml:"setup,omitempty" json:"setup,omitempty"`
	Exploit      ExploitScenario `yaml:"exploit_scenario" json:"exploit_scenario" jsonschema:"required"`
	Assertions   []Step          `yaml:"assertions" json:"assertions" jsonschema:"required"`
	Verification []Step          `yaml:"verification,omitempty" json:"verification,omitempty"`
}

// Metadata contains descriptive information about the PoC and vulnerability as described in Section 4
type Metadata struct {
	ID                string             `yaml:"id" json:"id" jsonschema:"required"`
	Title             string             `yaml:"title" json:"title" jsonschema:"required"`
	DslVersion        string             `yaml:"dsl_version" json:"dsl_version" jsonschema:"required,pattern=^1\\.0$"`
	SourceReport      *SourceReport      `yaml:"source_report,omitempty" json:"source_report,omitempty"`
	Vulnerability     *Vulnerability     `yaml:"vulnerability,omitempty" json:"vulnerability,omitempty"`
	TargetApplication *TargetApplication `yaml:"target_application,omitempty" json:"target_application,omitempty"`
	Severity          string             `yaml:"severity,omitempty" json:"severity,omitempty" jsonschema:"enum=Info,enum=Low,enum=Medium,enum=High,enum=Critical"`
	Tags              []string           `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// SourceReport contains information about the vulnerability report as described in Section 4
type SourceReport struct {
	Platform     string `yaml:"platform,omitempty" json:"platform,omitempty"`
	ID           string `yaml:"id,omitempty" json:"id,omitempty"`
	ReportedDate string `yaml:"reported_date,omitempty" json:"reported_date,omitempty" jsonschema:"pattern=^\\d{4}-\\d{2}-\\d{2}$"`
}

// Vulnerability contains details about the vulnerability as described in Section 4
type Vulnerability struct {
	Type string      `yaml:"type,omitempty" json:"type,omitempty"`
	CWE  interface{} `yaml:"cwe,omitempty" json:"cwe,omitempty"` // Can be string or integer
}

// TargetApplication contains details about the affected application as described in Section 4
type TargetApplication struct {
	Name    string `yaml:"name,omitempty" json:"name,omitempty"`
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
}

// Context defines the necessary environment, actors, resources, files, and variables as described in Section 5
type Context struct {
	Description string               `yaml:"description,omitempty" json:"description,omitempty"`
	Users       []ContextUser        `yaml:"users,omitempty" json:"users,omitempty"`
	Resources   []ContextResource    `yaml:"resources,omitempty" json:"resources,omitempty"`
	Environment []ContextEnvironment `yaml:"environment,omitempty" json:"environment,omitempty"`
	Files       []ContextFile        `yaml:"files,omitempty" json:"files,omitempty"`
	Variables   []ContextVariable    `yaml:"variables,omitempty" json:"variables,omitempty"`
}

// ContextUser defines a user context as described in Section 5.1
type ContextUser struct {
	ID             string                 `yaml:"id" json:"id" jsonschema:"required"`
	Description    string                 `yaml:"description,omitempty" json:"description,omitempty"`
	CredentialsRef string                 `yaml:"credentials_ref,omitempty" json:"credentials_ref,omitempty"`
	Credentials    map[string]interface{} `yaml:"credentials,omitempty" json:"credentials,omitempty"`
}

// ContextResource defines a resource as described in Section 5.2
type ContextResource struct {
	ID          string      `yaml:"id" json:"id" jsonschema:"required"`
	Description string      `yaml:"description,omitempty" json:"description,omitempty"`
	Identifier  interface{} `yaml:"identifier,omitempty" json:"identifier,omitempty"`
	Type        string      `yaml:"type,omitempty" json:"type,omitempty"`
}

// ContextEnvironment defines an environment setting as described in Section 5.3
type ContextEnvironment struct {
	ID    string      `yaml:"id" json:"id" jsonschema:"required"`
	Value interface{} `yaml:"value" json:"value" jsonschema:"required"`
}

// ContextFile defines a file as described in Section 5.4
type ContextFile struct {
	ID          string `yaml:"id" json:"id" jsonschema:"required"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	LocalPath   string `yaml:"local_path" json:"local_path" jsonschema:"required"`
}

// ContextVariable defines a variable as described in Section 5.5
type ContextVariable struct {
	ID    string      `yaml:"id" json:"id" jsonschema:"required"`
	Value interface{} `yaml:"value,omitempty" json:"value,omitempty"`
}

// Step represents a step within a phase as described in Section 10.1
type Step struct {
	Step   interface{} `yaml:"step,omitempty" json:"step,omitempty"`
	DSL    string      `yaml:"dsl" json:"dsl" jsonschema:"required"`
	ID     string      `yaml:"id,omitempty" json:"id,omitempty"`
	If     string      `yaml:"if,omitempty" json:"if,omitempty"`
	Loop   *Loop       `yaml:"loop,omitempty" json:"loop,omitempty"`
	Manual bool        `yaml:"manual,omitempty" json:"manual,omitempty"`
	Action *Action     `yaml:"action,omitempty" json:"action,omitempty"`
	Check  *Check      `yaml:"check,omitempty" json:"check,omitempty"`
}

// Loop defines loop behavior for a step as described in Section 10.1
type Loop struct {
	Over         string `yaml:"over" json:"over" jsonschema:"required"`
	VariableName string `yaml:"variable_name" json:"variable_name" jsonschema:"required"`
	Steps        []Step `yaml:"steps,omitempty" json:"steps,omitempty"`
}

// Action defines an operation to be performed as described in Section 10.2
type Action struct {
	Type                  string                  `yaml:"type" json:"type" jsonschema:"required"`
	Description           string                  `yaml:"description,omitempty" json:"description,omitempty"`
	AuthenticationContext string                  `yaml:"authentication_context,omitempty" json:"authentication_context,omitempty"`
	Timeout               string                  `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Retries               int                     `yaml:"retries,omitempty" json:"retries,omitempty"`
	RetryDelay            string                  `yaml:"retry_delay,omitempty" json:"retry_delay,omitempty"`
	Request               *HTTPRequest            `yaml:"request,omitempty" json:"request,omitempty"`
	ResponseActions       []HTTPResponseAction    `yaml:"response_actions,omitempty" json:"response_actions,omitempty"`
	TargetVariable        string                  `yaml:"target_variable,omitempty" json:"target_variable,omitempty"`
	Commands              []string                `yaml:"commands,omitempty" json:"commands,omitempty"`
	Users                 []string                `yaml:"users,omitempty" json:"users,omitempty"`
	Resource              string                  `yaml:"resource,omitempty" json:"resource,omitempty"`
	UserContext           string                  `yaml:"user_context,omitempty" json:"user_context,omitempty"`
	Generator             string                  `yaml:"generator,omitempty" json:"generator,omitempty"`
	Parameters            map[string]interface{}  `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	Duration              string                  `yaml:"duration,omitempty" json:"duration,omitempty"`
	AuthType              string                  `yaml:"auth_type,omitempty" json:"auth_type,omitempty"`
	URL                   string                  `yaml:"url,omitempty" json:"url,omitempty"`
}

// HTTPRequest defines an HTTP request as described in Section 10.4
type HTTPRequest struct {
	Method      string                 `yaml:"method" json:"method" jsonschema:"required"`
	URL         string                 `yaml:"url" json:"url" jsonschema:"required"`
	Headers     map[string]string      `yaml:"headers,omitempty" json:"headers,omitempty"`
	BodyType    string                 `yaml:"body_type,omitempty" json:"body_type,omitempty"`
	Body        interface{}            `yaml:"body,omitempty" json:"body,omitempty"`
	Multipart   *MultipartRequest      `yaml:"multipart,omitempty" json:"multipart,omitempty"`
	Redirects   *bool                  `yaml:"redirects,omitempty" json:"redirects,omitempty"`
	MaxRedirects *int                  `yaml:"max_redirects,omitempty" json:"max_redirects,omitempty"`
}

// MultipartRequest defines multipart form data for file uploads as described in Section 10.4
type MultipartRequest struct {
	Files []FileUpload          `yaml:"files,omitempty" json:"files,omitempty"`
	Data  map[string]string     `yaml:"data,omitempty" json:"data,omitempty"`
}

// FileUpload defines a file to be uploaded as described in Section 10.4
type FileUpload struct {
	ParameterName string `yaml:"parameter_name" json:"parameter_name" jsonschema:"required"`
	Filename      string `yaml:"filename" json:"filename" jsonschema:"required"`
	LocalPath     string `yaml:"local_path" json:"local_path" jsonschema:"required"`
	ContentType   string `yaml:"content_type,omitempty" json:"content_type,omitempty"`
}

// HTTPResponseAction defines actions performed on an HTTP response as described in Section 10.5
type HTTPResponseAction struct {
	Type            string  `yaml:"type" json:"type" jsonschema:"required"`
	Description     string  `yaml:"description,omitempty" json:"description,omitempty"`
	Source          string  `yaml:"source,omitempty" json:"source,omitempty"`
	TargetVariable  string  `yaml:"target_variable" json:"target_variable" jsonschema:"required"`
	JSONPath        string  `yaml:"json_path,omitempty" json:"json_path,omitempty"`
	HeaderName      string  `yaml:"header_name,omitempty" json:"header_name,omitempty"`
	Regex           string  `yaml:"regex,omitempty" json:"regex,omitempty"`
	Group           *int    `yaml:"group,omitempty" json:"group,omitempty"`
	CSSSelector     string  `yaml:"css_selector,omitempty" json:"css_selector,omitempty"`
	XPath           string  `yaml:"xpath,omitempty" json:"xpath,omitempty"`
	Attribute       string  `yaml:"attribute,omitempty" json:"attribute,omitempty"`
	ExtractAll      bool    `yaml:"extract_all,omitempty" json:"extract_all,omitempty"`
}

// Check defines a condition or state to be verified as described in Section 10.3
type Check struct {
	Type               string                 `yaml:"type" json:"type" jsonschema:"required"`
	Description        string                 `yaml:"description,omitempty" json:"description,omitempty"`
	RetryInterval      string                 `yaml:"retry_interval,omitempty" json:"retry_interval,omitempty"`
	MaxAttempts        int                    `yaml:"max_attempts,omitempty" json:"max_attempts,omitempty"`
	ExpectedStatus     interface{}            `yaml:"expected_status,omitempty" json:"expected_status,omitempty"`
	Contains           interface{}            `yaml:"contains,omitempty" json:"contains,omitempty"`
	Equals             interface{}            `yaml:"equals,omitempty" json:"equals,omitempty"`
	EqualsJSON         interface{}            `yaml:"equals_json,omitempty" json:"equals_json,omitempty"`
	JSONPath           *JSONPathCheck         `yaml:"json_path,omitempty" json:"json_path,omitempty"`
	Regex              string                 `yaml:"regex,omitempty" json:"regex,omitempty"`
	HeaderName         string                 `yaml:"header_name,omitempty" json:"header_name,omitempty"`
	ResourceType       string                 `yaml:"resource_type,omitempty" json:"resource_type,omitempty"`
	Path               string                 `yaml:"path,omitempty" json:"path,omitempty"`
	State              string                 `yaml:"state,omitempty" json:"state,omitempty"`
	ContentContains    string                 `yaml:"content_contains,omitempty" json:"content_contains,omitempty"`
	ContentEquals      string                 `yaml:"content_equals,omitempty" json:"content_equals,omitempty"`
	ExpectedError      *ExpectedError         `yaml:"expected_error,omitempty" json:"expected_error,omitempty"`
}

// JSONPathCheck defines a check using JSONPath as described in Section 10.3
type JSONPathCheck struct {
	Path         string      `yaml:"path" json:"path" jsonschema:"required"`
	Value        interface{} `yaml:"value,omitempty" json:"value,omitempty"`
	ValueMatches string      `yaml:"value_matches,omitempty" json:"value_matches,omitempty"`
}

// ExpectedError defines expected error conditions as described in Section 10.3
type ExpectedError struct {
	StatusMatches    string `yaml:"status_matches,omitempty" json:"status_matches,omitempty"`
	MessageContains  string `yaml:"message_contains,omitempty" json:"message_contains,omitempty"`
	ErrorTypeMatches string `yaml:"error_type_matches,omitempty" json:"error_type_matches,omitempty"`
}

// ExploitScenario defines the core exploit steps and scenario-specific setup/teardown as described in Section 7
type ExploitScenario struct {
	Name     string `yaml:"name,omitempty" json:"name,omitempty"`
	Setup    []Step `yaml:"setup,omitempty" json:"setup,omitempty"`
	Steps    []Step `yaml:"steps" json:"steps" jsonschema:"required"`
	Teardown []Step `yaml:"teardown,omitempty" json:"teardown,omitempty"`
}
