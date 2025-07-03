# passwd

A standalone password hashing tool for buildah tests.

## Purpose

This tool generates bcrypt password hashes and is used exclusively for testing purposes. It was previously part of the main buildah command as a hidden `passwd` subcommand but has been split out into a separate tool to:

- Keep the main buildah command clean of test-only functionality
- Allow tests to use password hashing independently
- Follow the same pattern as other test tools like `imgtype`

## Usage

```bash
passwd <password>
```

## Example

```bash
$ passwd testpassword
$2a$10$ZamosnV9dfpTJn4Uk.Xix.5nwbKNiLw8xpP/6g2z83jhY.WKZuRjG
```

The tool outputs a bcrypt hash of the input password to stdout, which can be used in test scenarios that require password hashing (such as setting up test registries with HTTP basic authentication).

## Building

The tool is built automatically when running `make all` or can be built individually with:

```bash
make bin/passwd
```