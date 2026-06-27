# Development

Onyx is deliberately boring Go. The local contract is the same one enforced by
CI:

```sh
files="$(gofmt -l $(git ls-files '*.go'))"
test -z "$files" || { echo "$files"; exit 1; }
go vet ./...
go test ./...
```

The project is stdlib-only. Do not add third-party requirements or `go.sum`
unless the no-dependency contract is being changed deliberately.

When changing generated output, preserve relative URLs, the `.nojekyll` behavior,
the conservative overwrite guards, and the generated-file markers. Update tests
alongside behavior changes.

## Rebuilding this site

Because the website is itself an Onyx site, regenerate it from the repository
root after editing anything under `docs/`:

```sh
go run ./onyx .
```

That rewrites `index.html` and `public/`. Commit the regenerated files together
with your Markdown changes. See [[deploying|Deploying]] for how the output is
served.

Onyx lives on [GitHub](https://github.com/jwd83/onyx).
