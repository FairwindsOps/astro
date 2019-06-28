# Contributing

Issues, whether bugs, tasks, or feature requests are essential for keeping dd-manager great. We believe it should be as easy as possible to contribute changes that get things working in your environment. There are a few guidelines that we need contributors to follow so that we can keep on top of things.

## Code of Conduct

This project adheres to a [code of conduct](CODE_OF_CONDUCT.md). Please review this document before contributing to this project.

## Project Structure

Dd-manager is built using the [Kubernetes Go client](https://github.com/kubernetes/client-go) that makes use of the [Kubernetes Api](https://kubernetes.io/docs/reference/using-api/api-concepts/).  The project consists of a collection of controllers that watch for Kubernetes object updates, and sends these updates as events to handlers, which interact with the [Datadog Api](https://docs.datadoghq.com/api/) to manage the lifecycle of monitors.

## Getting Started

We label issues with the ["good first issue" tag](https://github.com/reactiveops/dd-manager/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22) if we believe they'll be a good starting point for new contributors. If you're interested in working on an issue, please start a conversation on that issue, and we can help answer any questions as they come up.

## Setting Up Your Development Environment
### Prerequisites
* A properly configured Golang environment with Go 1.11 or higher
* Access to a Kubernetes cluster defined in `~/.kube/config` or `$KUBECONFIG`.

### Installation
* Install the project with `go get github.com/reactiveops/dd-manager`
* Change into the dd-manager directory which is installed at `$GOPATH/src/github.com/reactiveops/dd-manager`
* Run the tool with `go run main.go`.

## Running Tests

The following commands are all required to pass as part of dd-manager testing:

```
golint ./...
go fmt ./...
go test -v --bench --benchmem ./pkg/...
```

### Datadog Mocking
We mock the interface for the Datadog API client library in `./pkg/datadog/datadog.go`.
If you're adding a new function to the interface there, you'll need to regenerate the
mocks using
```
go install github.com/golang/mock/mockgen
mockgen -source=pkg/datadog/datadog.go -destination=pkg/mocks/datadog_mock.go
```

## Creating a New Issue

If you've encountered an issue that is not already reported, please create an issue that contains the following:

- Clear description of the issue
- Steps to reproduce it
- Appropriate labels

## Creating a Pull Request

Each new pull request should:

- Reference any related issues
- Add tests that show the issues have been solved
- Pass existing tests and linting
- Contain a clear indication of if they're ready for review or a work in progress
- Be up to date and/or rebased on the master branch

## Creating a new release

The steps are:
1. Create a PR for this repo
    1. Bump the version number in:
        1. README.md
    2. Update CHANGELOG.md
    3. Merge your PR
2. Tag the latest branch for this repo
    1. Pull the latest for the `master` branch
    2. Run `git tag $VERSION && git push --tags`
    3. Wait for CircleCI to finish the build for the tag, which pushes images to quay.io and creates a release in github
