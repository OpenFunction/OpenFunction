# Development Workflow

## Step 1. Fork

1. Visit https://github.com/OpenFunction/OpenFunction
2. Click `Fork` button to create a fork of the project to your GitHub account.

## Step 2. Clone fork to local storage

Per Go's [workspace instructions](https://golang.org/doc/code.html#Workspaces), place OpenFunction code on your `GOPATH` using the following cloning procedure.

Define a local working directory:

```
export working_dir=$GOPATH/src/openfunction.io
export user={your github profile name}
```

Create your clone locally:

```
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

```
git fetch upstream
git checkout main
git rebase upstream/main
```

## Step 4. Add new features or fix issues

Create a branch from master:

```
git checkout -b myfeature
```

Then edit code on the `myfeature` branch. You can refer to [effective_go](https://golang.org/doc/effective_go.html) while writing code.

### Test and build

Currently, the make rules only contain simple checks such as vet, unit test, will add e2e tests soon.

### Using KubeBuilder

- For Linux OS, you can download and execute this [KubeBuilder script](https://raw.githubusercontent.com/kubesphere/kubesphere/master/hack/install_kubebuilder.sh).
- For MacOS, you can install KubeBuilder by following this [guide](https://book.kubebuilder.io/quick-start.html).

### Run and test

```
make all
# Run every unit test
make test
```

Run `make help` for additional information on these make targets.

## Step 5. Development in new branch

### Sync with upstream

After the test is completed, it is a good practice to keep your local in sync with upstream to avoid conflicts.

```
# Rebase your master branch of your local repo.
git checkout main
git rebase upstream/main

# Then make your development branch in sync with master branch
git checkout new_feature
git rebase -i main
```

### Commit local changes

```
git add <file>
git commit -s -m "add your description"
```

## Step 6. Push to your fork

When ready to review (or just to establish an offsite backup of your work), push your branch to your fork on GitHub:

```
git push -f ${your_remote_name} myfeature
```

## Step 7. Create a PR

- Visit your fork at https://github.com/$user/OpenFunction
- Click the` Compare & Pull Request` button next to your myfeature branch.
- Check out the [pull request process](pull-requests.md) for more details and advice.