# Leaderlog API

A lightweight API to manage the leader log of a stake pool such that
it can be safely used by a Web application, i.e. the exact minting 
time of an assigned block isn't exposed in advance. At the moment,
this application makes use of the [Blockfrost API](https://blockfrost.io),
which is why an API key is needed for this service.

## Build

This application can easily be built with the following command. 

```bash
$ go mod vendor && go build
```

## Usage

```
Usage: leaderlog-api -pool-id <pool-id> [options]
  -db-path string
        path to the directory with the leader log db. (default ".db")
  -hostname string
        location at which the API shall be served. (default "localhost")
  -level string
        level of logging. (default "info")
  -pool-id string
        pool ID in hex format.
  -port int
        port on which the API shall be served. (default 9001)
```

The application expects some values to be specified in your environment.

| Name                    | Usage                                             |
|-------------------------|---------------------------------------------------|
| BLU_BLOCKFROST_API_KEY  | Specifies the API key that shall be used for Blockfrost |
| BLU_AUTH_USERNAME | Specifies the username for access control |
| BLU_AUTH_PASSWORD | Specifies the password for access control |

## Contact

* [Kevin Haller](kevin.haller@blockbllu.io) (Operator of the SOBIT stake pool)