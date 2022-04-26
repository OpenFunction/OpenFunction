# Development Workflow

## Step 1. Fork

1. Visit https://github.com/OpenFunction/OpenFunction
2. Click `Fork` button to create a fork of the project to your GitHub account.

## Step 2. Clone fork to local storage

Per Go's [workspace instructions](https://golang.org/doc/code.html#Workspaces), place OpenFunction code on your `GOPATH` using the following cloning procedure.

Define a local working directory:

```shell
export working_dir=$GOPATH/src/openfunction.io
export user={your github profile name}
```

Create your clone locally:

```shell
mkdir -p $working_dir
cd $working_dir
git clone https://github.com/$user/OpenFunction.git
cd $working_dir/OpenFunction
git remote add upstream https://github.com/OpenFunction/OpenFunction.git

# Never push to upstream master
git remote set-url --push upstream no_push

# Confirm your remotes make sense:
git remote -v
```

## Step 3. Keep your branch in sync

```shell
git fetch upstream
git checkout main
git rebase upstream/main
```

## Step 4. Add new features or fix issues

Create a branch from master:

```shell
git checkout -b myfeature
```

Then edit code on the `myfeature` branch. You can refer to [effective_go](https://golang.org/doc/effective_go.html) while writing code.

### Test and build

Currently, the make rules only contain simple checks such as vet, unit test, will add e2e tests soon.

### Using KubeBuilder

- For Linux OS, you can download and execute this [KubeBuilder script](https://raw.githubusercontent.com/kubesphere/kubesphere/master/hack/install_kubebuilder.sh).
- For MacOS, you can install KubeBuilder by following this [guide](https://book.kubebuilder.io/quick-start.html).

### Run and test

> Run `make help` for additional information on these make targets.

```shell
make all
# Run every unit test
make test
```

### E2E test

We recommend using the following commands for e2e testing:

You need to first look at the `test/e2e_env` file, which serves to provide the necessary environment variables for the e2e testing process. You **must complete** the configuration of these environment variables before you can run the e2e test.

```shell
# Your dockerhub username
DOCKERHUB_USERNAME=
# Your dockerhub password
DOCKERHUB_PASSWORD=
# Your dockerhub repository name
DOCKERHUB_REPO_PREFIX=
```

You can run all e2e tests with a single command:

```shell
make e2e
```

You can also run e2e test on individual functions:

```shell
# e2e testing for knative-runtime
make e2e-knative
# e2e testing for async-runtime
make e2e-async
# e2e testing for plugin functionality
make e2e-plugin
# e2e testing for events-framework functionality
make e2e-events
```

## Step 5. Development in new branch

### Sync with upstream

After the test is completed, it is a good practice to keep your local in sync with upstream to avoid conflicts.

```shell
# Rebase your master branch of your local repo.
git checkout main
git rebase upstream/main

# Then make your development branch in sync with master branch
git checkout new_feature
git rebase -i main
```

> ***Note**:* A script is provided in the `hack` directory to generate the certificate information required by the webhook and you can execute it using the following command. This script can generate certificates for a lifetime of 3650 days, so we recommend not running it until OpenFunction's certificate has expired.

```shell
cd hack && bash generate-cert.sh
```

### Commit local changes

```shell
git add <file>
git commit -s -m "add your description"
```

## Step 6. Push to your fork

When ready to review (or just to establish an offsite backup of your work), push your branch to your fork on GitHub:

```shell
git push -f ${your_remote_name} myfeature
```

## Step 7. Create a PR

- Visit your fork at https://github.com/$user/OpenFunction
- Click the` Compare & Pull Request` button next to your myfeature branch.
- Check out the [pull request process](pull-requests.md) for more details and advice.