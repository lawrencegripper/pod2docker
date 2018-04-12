# Pod-2-Docker

Simple utility which takes a Kubernetes Pod definition and generates a Bash script which emulates the Pods behavior on a VM using docker commands. 

## Testing

Docker is required.

Run `./ci.sh` at the root directory. The tests will create and execute docker commands to validate pod behavior (Volumes, IPC and Network). 

## Supported Volumes

- EmptyDir
- HostDir

## Supported Configuration

Note many features of pods aren't included. Currently the following settings are supported:

- ImagePullCredentials
- ImagePullPolicy
- Volumes
- VolumeMounts
- Command
- Args

See [Unit](pod2docker_test.go) and [Integrations](pod2docker_integration_test.go) tests to understand more on usage.