# Contributing to Gopherize.me

[gopherize.me](https://gopherize.me) is an open source project. Your help is very welcome!

## Filing issues

Sensitive security-related issues should be reported to [Mat Ryer directly](https://twitter.com/matryer).

For any other problem fill up a new issue describing your feature request or problem.

## Contributing code
To use Github you will need to install [Git](https://git-scm.com/downloads) and (optional) [Github credentials setup](https://help.github.com/articles/connecting-to-github-with-ssh/)

1. File up a new issue (or use an existing open one)
2. [Fork this repository](https://help.github.com/articles/fork-a-repo/)
3. [Download your repository](https://help.github.com/articles/cloning-a-repository/) on your computer
4. Make a new branch `git checkout -b myNewBranch`
5. Update your repository and [make a pull request](https://help.github.com/articles/creating-a-pull-request/)


## Installing the project
In order to make the project work on your local computer you need the following:

* [Google Cloud SDK](https://cloud.google.com/sdk/install) with the **App Engine Go Extensions** component installed.
* [Go](https://golang.org/doc/install) the language and tools
* the local copy of your repository must be in the [GOPATH folder](https://github.com/golang/go/wiki/SettingGOPATH)
* run `go get ./...` in the project main folder, in order to install its dependencies

## Run the project
The project runs using the local [Google App Engine development server](https://cloud.google.com/appengine/docs/standard/go/tools/using-local-server). The command that runs it is already written in a [bash file](./gae/run.sh). 

```bash
./gae/run.sh
```
Windows/others see the [Development server docs](https://cloud.google.com/appengine/docs/standard/go/tools/using-local-server#running_the_local_development_server).

> Note: The config for the app engine server is `/gae/app.yaml`.

After the server starts, you can visit [http://localhost:8080/](http://localhost:8080/) in your web browser to see the app in action.
