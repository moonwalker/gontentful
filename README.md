# Gontentful

Contentful client library for Go with a command line interface for schema export and data sync.

## CLI:

Install:

```sh
$ go get -u github.com/moonwalker/gontentful/cmd/gfl
```

### Usage

Schema export:

```sh
# generate postgres schema and print to stdout
$ gfl schema pg --space <spaceid> --token <token>

# generate postgres schema and execute on the specified database
$ gfl schema pg --space <spaceid> --token <token> --url postgres://user:pass@host:port/db

# generate graphql schema and print to stdout
$ gfl schema gql --space <spaceid> --token <token>
```

Data sync:

```sh
# sync data to postgres (init sync first then incremental)
$ gfl sync pg --space <spaceid> --token <token> --url postgres://user:pass@host:port/db

# sync data to postgres (init sync always start from scratch)
$ gfl sync pg --space <spaceid> --token <token> --url postgres://user:pass@host:port/db --init
```

## Dependencies

Using Go modules:

```sh
$ go mod vendor
```

## License

Licensed under the [MIT License](LICENSE)

### Acknowledgements

Utilize code from [contentful-go](https://github.com/contentful-labs/contentful-go)
