# git-security

## Project description

git-security is an application that displays many git repositories information in a table view with ecommerce-like filters. It also supports changing different branch protection rule options in bulk. In order to pull in more information from other services, you can also configure custom hooks using docker images to enrich the data besides GitHub.

Architecture diagram:

![alt text](https://github.com/PaloAltoNetworks/git-security/blob/main/architecture.png?raw=true)

Screenshot of the UI:
![alt text](https://github.com/PaloAltoNetworks/git-security/blob/main/ui.png?raw=true)

## Features

- Ecommerce-like contextual filters
- Columns are configurable
- Custom data is supported using docker images
- Bulk action on changing the branch protection rule options in many repositories

## How to try it out

Pre-requisite: Docker, GitHub personal access token (from https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens)

```sh
export GITHUB_HOST=example.com
export GH_TOKEN=ghp_xyz
docker run -d -e "GH_TOKEN=$GH_TOKEN" -e "GITHUB_HOST=$GITHUB_HOST" -p 8080:8080 matthewkwong/git-security:latest
```

## How to develop

Pre-requisite: Go 1.22+, npm

1. Run go backend server

```sh
go run github.com/PaloAltoNetworks/git-security/cmd/git-security
```

2. Run UI

```sh
cd cmd/git-security/ui
npm install
npm run dev -- --open
```

## How to build the image

Pre-requisite: Docker

```sh
make image
```

or if you want to deploy to k8s with amd64 arch

```sh
make image-amd64
```

# App options

```
COMMANDS:
   generate-key  generate a random encryption key for GIT_SECURITY_KEY
   help, h       Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --github-host host                  GitHub host (default: "github.com") [$GITHUB_HOST]
   --github-pat PAT                    GitHub PAT [$GITHUB_PAT, $GH_TOKEN]
   --http-port port                    HTTP port (default: 8080) [$HTTP_PORT]
   --https-ssl-cert-location location  HTTPS SSL cert location [$HTTPS_SSL_CERT_LOCATION]
   --https-ssl-key-location location   HTTPS SSL key location [$HTTPS_SSL_KEY_LOCATION]
   --https-port port                   HTTPS port (default: 443) [$HTTPS_PORT]
   --psql-host host                    PostgreSQL host (default: "localhost") [$PSQL_HOST]
   --psql-port port                    PostgreSQL port (default: 5432) [$PSQL_PORT]
   --psql-username username            PostgreSQL username (default: "postgres") [$PSQL_USERNAME]
   --psql-password password            PostgreSQL password (default: "password") [$PSQL_PASSWORD]
   --psql-dbname dbname                PostgreSQL dbname (default: "postgres") [$PSQL_DBNAME]
   --mongo-host host                   Mongo host (default: "localhost") [$MONGO_HOST]
   --mongo-port port                   Mongo port (default: 27017) [$MONGO_PORT]
   --mongo-username username           Mongo username (default: "admin") [$MONGO_USERNAME]
   --mongo-password password           Mongo password (default: "password") [$MONGO_PASSWORD]
   --debug                             debug mode (default: false) [$GIT_SECURITY_DEBUG]
   --key value                         key for encrypting the env variable values in DB [$GIT_SECURITY_KEY]
   --cacert value                      cacert for accessing the GitHub [$GIT_SECURITY_CACERT]
   --admin-username value              basic auth admin username (default: "admin") [$GIT_SECURITY_ADMIN_USERNAME]
   --admin-password value              basic auth admin password (default: "changeme") [$GIT_SECURITY_ADMIN_PASSWORD]
   --db value                          Sqlite (sqlite), PostgreSQL (pg) or Mongo (mongo) as database backend (default: "sqlite") [$GIT_SECURITY_DB]
   --help, -h                          show help
   --version, -v                       print the version
```

In order to encrypt the custom logic envs provided by the users, we need to configure --key option, use generate-key command to randomly create one if not existed

```sh
go run github.com/PaloAltoNetworks/git-security/cmd/git-security generate-key
```

For backend database, MongoDB is recommended. PostgreSQL and Sqlite are supported through FerretDB (https://github.com/FerretDB/FerretDB)

# Columns configuration

# Custom hooks configuration

# Automations

## Pre-Receive Hook Enforcement

Using this automation the pre-receive hooks of a github repository can be enabled or disabled.

The following environment variables must be set to enforce the automation:

```
GITHUB_HOST                     Github Host
GH_TOKEN                        Github PAT
GIT_PRE_RECEIVE_HOOKS_ENABLE    Pre-receive hooks to be enabled
GIT_PRE_RECEIVE_HOOKS_DISABLE   Pre-receive hooks to be disabled
```

The user can specify the hooks to enable or disable or both

## License

MIT
