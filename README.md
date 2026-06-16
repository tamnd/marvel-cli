# marvel

Read the Marvel Comics API

`marvel` is a single pure-Go binary. It reads from the official Marvel Comics
developer API, shapes data into clean records, and prints output that pipes
into the rest of your tools. Requires `MARVEL_PUBLIC_KEY` and `MARVEL_PRIVATE_KEY`
(free registration at developer.marvel.com).

The same package is also a [resource-URI driver](#use-it-as-a-resource-uri-driver),
so a host program like [ant](https://github.com/tamnd/ant) can address
marvel as `marvel://` URIs.

## Install

```bash
go install github.com/tamnd/marvel-cli/cmd/marvel@latest
```

Or grab a prebuilt binary from the [releases](https://github.com/tamnd/marvel-cli/releases), or run
the container image:

```bash
docker run --rm ghcr.io/tamnd/marvel:latest --help
```

## Setup

```bash
export MARVEL_PUBLIC_KEY=your-public-key
export MARVEL_PRIVATE_KEY=your-private-key
```

## Usage

```bash
marvel characters                     # list characters
marvel characters -n 20               # first 20 characters
marvel character <id>                 # fetch one character by id
marvel comics                         # list comics
marvel comic <id>                     # fetch one comic by id
marvel search <query>                 # search characters and comics
marvel character <id> -o json         # as JSON, ready for jq
marvel --help                         # the whole command tree
```

Every command shares one output contract: `-o table|json|jsonl|csv|tsv|url|raw`,
`--fields` to pick columns, `--template` for a custom line, and `-n` to limit.
The default adapts to where output goes (a table on a terminal, JSONL in a
pipe), so the same command reads well by hand and parses cleanly downstream.

## Commands

| Command | Description |
|---------|-------------|
| `marvel characters` | List all Marvel characters |
| `marvel character <id>` | Fetch a single character by numeric id |
| `marvel comics` | List Marvel comics |
| `marvel comic <id>` | Fetch a single comic by numeric id |
| `marvel search <query>` | Search characters and comics |

## Serve it

The same operations are available over HTTP and as an MCP tool set for agents,
with no extra code:

```bash
marvel serve --addr :7777    # GET /v1/character/<id>  returns NDJSON
marvel mcp                   # speak MCP over stdio
```

## Use it as a resource-URI driver

`marvel` registers a `marvel` domain the way a program registers a
database driver with `database/sql`. A host enables it with one blank import:

```go
import _ "github.com/tamnd/marvel-cli/marvel"
```

Then [ant](https://github.com/tamnd/ant) (or any program that links the package)
dereferences `marvel://` URIs without knowing anything about Marvel:

```bash
ant get marvel://character/<id>   # fetch the record
ant get marvel://comic/<id>       # fetch a comic
ant url marvel://character/<id>   # the live https URL
```

## Development

```
cmd/marvel/   thin main: hands cli.NewApp to kit.Run
cli/                 assembles the kit App from the marvel domain
marvel/                the library: HTTP client, data models, and domain.go (the driver)
docs/                tago documentation site
```

```bash
make build      # ./bin/marvel
make test       # go test ./...
make vet        # go vet ./...
```

## Releasing

Push a version tag and GitHub Actions runs GoReleaser, which builds the
archives, Linux packages, the multi-arch GHCR image, checksums, SBOMs, and a
cosign signature:

```bash
git tag v0.1.0
git push --tags
```

The Homebrew and Scoop steps self-disable until their tokens exist, so the first
release works with no extra secrets.

## License

Apache-2.0. See [LICENSE](LICENSE).
