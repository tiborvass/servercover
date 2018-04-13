# Usecase: coverage on integration tests against a server

Imagine there is a long running server to test with integration tests that are client invocations.
Servercover allows you to gather coverage information on the server for the duration of the integration tests' run.

# Example

In the `example/` folder there are two folders:
- `program/` where the program code lives
- `tests/` where the integration tests live

The program is itself composed of `server/` and `client/` components, both of which need to be compiled first.
The integration tests in `tests/` can then be run as follows:
`EXAMPLE_SOCK=example.sock go test -coverprofile=cover.out ./tests -cover.addr cover.sock` where `EXAMPLE_SOCK` is an
application-specific parameter, in this case the socket needed to communicate with the server.

`example/run.sh` contains an entire example.
