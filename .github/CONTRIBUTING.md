# Contributing to OnlyLogins

Thank you for your interest in contributing! OnlyLogins is an open source
password manager, and we welcome contributions from its user base.

## Before You Start Coding

Please open an issue before you begin working on a change. A good issue includes:

- Details on the feature you'd like to see and the problem you hope to solve,
  including how you envision using the feature.
- Your wish to contribute the code, your proposed solution, and your test plan.

Creating the issue opens a discussion with us so that we can collect feedback,
help with design, track the feature request, and better review your pull
request. This initial discussion and collaboration ensures that you have the
support you need to implement and test your enhancement.

## Working on an Existing Issue

If you'd like to contribute to an existing issue, please post in that issue
before starting. Let us know you'd like to work on it and your proposed
solution.

## Pull Request Checklist

- Run the linters and tests before opening a pull request (see below).
- Keep changes focused on a single problem or feature.
- Add or update tests for any behavior you change.
- Ensure your code follows the existing style and uses the SPDX license header:

  ```
  // Copyright (c) 2026 Cattle Cloud LLC
  // SPDX-License-Identifier: AGPL-3.0
  ```

## AI Policy

Please do not submit pure AI-generated pull requests. Using agents for
assistance in code exploration and development is permissible, but do be
courteous and avoid submitting code that you yourself haven't reviewed. If you
can't be bothered to think critically about your changes, why should we?

## Development Setup

Assuming you have a working Go toolchain, getting started is as easy as running
a few [Justfile](https://just.systems/man/en/) targets.

```shell
# Install local tooling, run linters and tests
just init lint tests
```

```shell
# Run the instance and listen on localhost:3000
just run ./hack/dev.toml
```

### Accessing the database

```shell
# Open the SQLite file
just db-open ./hack/dev.toml

# Completely delete the SQLite file
just db-reset ./hack/dev.toml
```

### Using memcached

A [memcached](https://www.memcached.org) instance is expected to be available
and is used for storing session state. For local development a Justfile target
exists for convenience, and works with the `hack/dev.toml` configuration.

```shell
# Start a local memcached listening on /tmp/mc.sock
just mc-run

# Show stats output of the local memcached
# tip: STAT curr_items is the interesting metric
just mc-stats
```

### Stripe integration

Note: it is not necessary to integrate with Stripe for local development or for
running your own instance. If the Stripe configuration is left unset,
`onlylogins` will automatically exclude the subscription features.

## Architecture

When working on OnlyLogins, there are a few key packages you should be familiar
with before making changes:

- `agent/` - The main entrypoint of the long-lived web service
  - `agent/cronjobs/` - Creates background tasks that run periodically
  - `agent/web/` - Creates the HTTP, HTMX, and API web handlers
- `database/` - Storage layer backed by SQLite
- `hack/` - Contains example configuration and mock packages
- `pkg/` - Contains numerous helper packages
  - `pkg/encrypt/` - Where the security happens
  - `pkg/sitecontent/` - Models that bridge storage and web layers
- `ui/` - Defines HTML templates and CSS, Javascript files

## License

OnlyLogins is made available under the [AGPL-3.0](LICENSE) license. By
contributing, you agree that your contributions will be licensed under the same
terms.
