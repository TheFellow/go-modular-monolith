# arch-lint

`arch-lint` is a static analysis tool for Go projects that enforces architectural rules by analyzing import paths and package structures.
It helps maintain clean and consistent codebases by preventing unwanted dependencies and enforcing modular boundaries.

## Features

- Use glob patterns to include or exclude packages for analysis.
- Define custom rules to forbid specific imports.
- Support exceptions to allow forbidden imports in restricted contexts.

## Installation

```
go install github.com/TheFellow/arch-lint@latest
```

## Usage

Run the linter with a configuration file:

```bash
./arch-lint -config=path/to/rules.yml
```

### Configuration

The linter uses a `rules.yml` file to define the rules for your project.
Below is an example configuration:

```yaml
specs:
  - name: no-experimental-imports
    packages:
      include:
        - "example/alpha/**"
      exclude:
        - "example/alpha/internal/exception/**"
    rules:
      forbid:
        - "example/alpha/experimental"
      except:
        - "example/alpha/internal/excluded"
      exempt:
        - "example/alpha/common"
```

Note: By default test packages are excluded. This can be changed by setting `include_tests: true` on the configuration.

Configuration files are validated against a built-in YAML schema before the linter runs. Invalid files will cause arch-lint to exit with an error.

### Fields

- **name**: A descriptive name for the rule.
- **include**: Glob patterns specifying packages to include in the analysis.
- **exclude**: Glob patterns specifying packages to exclude from the analysis.
- **forbid**: Import paths that are forbidden.
- **except**: Import paths that are exceptions to the forbidden rules.
- **exempt**: Import paths that are exempt from `forbid` rules.

A `forbid` pattern supports a few special cases:
- `*`: Matches a single path segment.
- `**`: Matches multiple path segments, including none.
- `{variable}`: Matches a single path segment and captures it as a named variable.

An `except` pattern supports the same special cases as `forbid`, and one more
- `*`: Matches a single path segment.
- `**`: Matches multiple path segments, including none.
- `{variable}`: Matches this path segment when its value matches the one captured in the `forbid` pattern.
- `{!variable}`: Matches this path segment when its value **does not** match the one captured in the `forbid` pattern.

An `exempt` pattern supports the same special cases as `forbid`, and one more
- `*`: Matches a single path segment.
- `**`: Matches multiple path segments, including none.
- `{variable}`: Matches a single path segment and captures it as a named variable.
- `{!variable}`: Matches this path segment when its value **does not** match the one captured in the `forbid` pattern.

### How it works

First all packages in scope for analysis are collected.
That is, all packages that match the `include` glob patterns and do not match the `exclude` glob patterns.

Then each package in scope is analyzed.
During analysis there are two packages under consideration:
- The package being analyzed (the `current` package).
- The package being imported (the `imported` package).

The `current` package is forbidden from importing the `imported` package
if the `imported` package matches a `forbid` pattern.

Once forbidden, the `imported` package will be allowed if:
- The `current` package matches an `except` pattern.
- The `imported` package matches an `exempt` pattern.

This provides the flexibility to allow certain imports based on either the importer or the importee.

## Output

On the happy path the linter will output
```
✔ arch-lint: no forbidden imports found.
```
and exit with code 0.

On the unhappy path the linter will output

```
arch-lint: [<rule name>] package "path/to"  imports "forbidden/package"
```

and exit with code 1.

## Development

### Prerequisites

- Go 1.23 or later

### Running Tests

```
go test ./...
```

## Contributing

Contributions are welcome! Please open an issue or submit a pull request with your changes.

## License

This project is licensed under the MIT License. See the `LICENSE` file for details.