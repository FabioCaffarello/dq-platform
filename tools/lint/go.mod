// path: tools/lint/go.mod

module dq-platform/tools/lint

go 1.22

require (
	dq-platform/tools/pathsafe v0.0.0-00010101000000-000000000000
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1
	gopkg.in/yaml.v3 v3.0.1
)

// pathsafe is a sibling module in the same monorepo; the
// replace makes `cd tools/lint && go build` work outside the
// go.work workspace (e.g., for future Dockerfile builds).
replace dq-platform/tools/pathsafe => ../pathsafe
