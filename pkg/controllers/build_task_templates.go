package controllers

const (
	// Tekton build template from https://raw.githubusercontent.com/tektoncd/catalog/main/task/buildpacks/0.3/buildpacks.yaml
	tmplBuild = `
---
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: buildpacks
  labels:
    app.kubernetes.io/version: "0.3"
  annotations:
    tekton.dev/pipelines.minVersion: "0.17.0"
    tekton.dev/tags: image-build
    tekton.dev/displayName: "Buildpacks"
spec:
  description: >-
    The Buildpacks task builds source into a container image and pushes it to a registry,
    using Cloud Native Buildpacks.

  workspaces:
    - name: source
      description: Directory where application source is located.
    - name: cache
      description: Directory where cache is stored (when no cache image is provided).
      optional: true

  params:
    - name: APP_IMAGE
      description: The name of where to store the app image.
    - name: BUILDER_IMAGE
      description: The image on which builds will run (must include lifecycle and compatible buildpacks).
    - name: SOURCE_SUBPATH
      description: A subpath within the source input where the source to build is located.
      default: ""
    - name: ENV_VARS
      type: array
      description: Environment variables to set during _build-time_.
      default: []
    - name: PROCESS_TYPE
      description: The default process type to set on the image.
      default: "web"
    - name: RUN_IMAGE
      description: Reference to a run image to use.
      default: ""
    - name: CACHE_IMAGE
      description: The name of the persistent app cache image (if no cache workspace is provided).
      default: ""
    - name: SKIP_RESTORE
      description: Do not write layer metadata or restore cached layers.
      default: "false"
    - name: USER_ID
      description: The user ID of the builder image user.
      default: "1000"
    - name: GROUP_ID
      description: The group ID of the builder image user.
      default: "1000"
    - name: PLATFORM_DIR
      description: The name of the platform directory.
      default: empty-dir

  results:
    - name: APP_IMAGE_DIGEST
      description: The digest of the built APP_IMAGE.

  stepTemplate:
    env:
      - name: CNB_PLATFORM_API
        value: "0.4"

  steps:
    - name: prepare
      image: docker.io/library/bash:5.1.4@sha256:b208215a4655538be652b2769d82e576bc4d0a2bb132144c060efc5be8c3f5d6
      args:
        - "--env-vars"
        - "$(params.ENV_VARS[*])"
      script: |
        #!/usr/bin/env bash
        set -e

        if [[ "$(workspaces.cache.bound)" == "true" ]]; then
          echo "> Setting permissions on '$(workspaces.cache.path)'..."
          chown -R "$(params.USER_ID):$(params.GROUP_ID)" "$(workspaces.cache.path)"
        fi

        for path in "/tekton/home" "/layers" "$(workspaces.source.path)"; do
          echo "> Setting permissions on '$path'..."
          chown -R "$(params.USER_ID):$(params.GROUP_ID)" "$path"
        done

        echo "> Parsing additional configuration..."
        parsing_flag=""
        envs=()
        for arg in "$@"; do
            if [[ "$arg" == "--env-vars" ]]; then
                echo "-> Parsing env variables..."
                parsing_flag="env-vars"
            elif [[ "$parsing_flag" == "env-vars" ]]; then
                envs+=("$arg")
            fi
        done

        echo "> Processing any environment variables..."
        ENV_DIR="/platform/env"

        echo "--> Creating 'env' directory: $ENV_DIR"
        mkdir -p "$ENV_DIR"

        for env in "${envs[@]}"; do
            IFS='=' read -r key value string <<< "$env"
            if [[ "$key" != "" && "$value" != "" ]]; then
                path="${ENV_DIR}/${key}"
                echo "--> Writing ${path}..."
                echo -n "$value" > "$path"
            fi
        done
      volumeMounts:
        - name: layers-dir
          mountPath: /layers
        - name: $(params.PLATFORM_DIR)
          mountPath: /platform
      securityContext:
        privileged: true

    - name: create
      image: $(params.BUILDER_IMAGE)
      imagePullPolicy: Always
      command: ["/cnb/lifecycle/creator"]
      args:
        - "-app=$(workspaces.source.path)/$(params.SOURCE_SUBPATH)"
        - "-cache-dir=$(workspaces.cache.path)"
        - "-cache-image=$(params.CACHE_IMAGE)"
        - "-uid=$(params.USER_ID)"
        - "-gid=$(params.GROUP_ID)"
        - "-layers=/layers"
        - "-platform=/platform"
        - "-report=/layers/report.toml"
        - "-process-type=$(params.PROCESS_TYPE)"
        - "-skip-restore=$(params.SKIP_RESTORE)"
        - "-previous-image=$(params.APP_IMAGE)"
        - "-run-image=$(params.RUN_IMAGE)"
        - "$(params.APP_IMAGE)"
      volumeMounts:
        - name: layers-dir
          mountPath: /layers
        - name: $(params.PLATFORM_DIR)
          mountPath: /platform
      securityContext:
        runAsUser: 1000
        runAsGroup: 1000

    - name: results
      image: docker.io/library/bash:5.1.4@sha256:b208215a4655538be652b2769d82e576bc4d0a2bb132144c060efc5be8c3f5d6
      script: |
        #!/usr/bin/env bash
        set -e
        cat /layers/report.toml | grep "digest" | cut -d'"' -f2 | cut -d'"' -f2 | tr -d '\n' | tee $(results.APP_IMAGE_DIGEST.path)
      volumeMounts:
        - name: layers-dir
          mountPath: /layers

  volumes:
    - name: empty-dir
      emptyDir: {}
    - name: layers-dir
      emptyDir: {}
`
	// Tekton GitClone task https://raw.githubusercontent.com/tektoncd/catalog/main/task/git-clone/0.3/git-clone.yaml
	tmplGitClone = `
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: git-clone
  labels:
    app.kubernetes.io/version: "0.3"
  annotations:
    tekton.dev/pipelines.minVersion: "0.21.0"
    tekton.dev/tags: git
    tekton.dev/displayName: "git clone"
spec:
  description: >-
    These Tasks are Git tasks to work with repositories used by other tasks
    in your Pipeline.

    The git-clone Task will clone a repo from the provided url into the
    output Workspace. By default the repo will be cloned into the root of
    your Workspace. You can clone into a subdirectory by setting this Task's
    subdirectory param. This Task also supports sparse checkouts. To perform
    a sparse checkout, pass a list of comma separated directory patterns to
    this Task's sparseCheckoutDirectories param.

  workspaces:
    - name: output
      description: The git repo will be cloned onto the volume backing this workspace
  params:
    - name: url
      description: git url to clone
      type: string
    - name: revision
      description: git revision to checkout (branch, tag, sha, ref…)
      type: string
      default: ""
    - name: refspec
      description: (optional) git refspec to fetch before checking out revision
      default: ""
    - name: submodules
      description: defines if the resource should initialize and fetch the submodules
      type: string
      default: "true"
    - name: depth
      description: performs a shallow clone where only the most recent commit(s) will be fetched
      type: string
      default: "1"
    - name: sslVerify
      description: defines if http.sslVerify should be set to true or false in the global git config
      type: string
      default: "true"
    - name: subdirectory
      description: subdirectory inside the "output" workspace to clone the git repo into
      type: string
      default: ""
    - name: sparseCheckoutDirectories
      description: defines which directories patterns to match or exclude when performing a sparse checkout
      type: string
      default: ""
    - name: deleteExisting
      description: clean out the contents of the repo's destination directory (if it already exists) before trying to clone the repo there
      type: string
      default: "true"
    - name: httpProxy
      description: git HTTP proxy server for non-SSL requests
      type: string
      default: ""
    - name: httpsProxy
      description: git HTTPS proxy server for SSL requests
      type: string
      default: ""
    - name: noProxy
      description: git no proxy - opt out of proxying HTTP/HTTPS requests
      type: string
      default: ""
    - name: verbose
      description: log the commands used during execution
      type: string
      default: "true"
    - name: gitInitImage
      description: the image used where the git-init binary is
      type: string
      default: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init:v0.21.0"
  results:
    - name: commit
      description: The precise commit SHA that was fetched by this Task
    - name: url
      description: The precise URL that was fetched by this Task
  steps:
    - name: clone
      image: $(params.gitInitImage)
      script: |
        #!/bin/sh
        set -eu -o pipefail

        if [[ "$(params.verbose)" == "true" ]] ; then
          set -x
        fi

        CHECKOUT_DIR="$(workspaces.output.path)/$(params.subdirectory)"

        cleandir() {
          # Delete any existing contents of the repo directory if it exists.
          #
          # We don't just "rm -rf $CHECKOUT_DIR" because $CHECKOUT_DIR might be "/"
          # or the root of a mounted volume.
          if [[ -d "$CHECKOUT_DIR" ]] ; then
            # Delete non-hidden files and directories
            rm -rf "$CHECKOUT_DIR"/*
            # Delete files and directories starting with . but excluding ..
            rm -rf "$CHECKOUT_DIR"/.[!.]*
            # Delete files and directories starting with .. plus any other character
            rm -rf "$CHECKOUT_DIR"/..?*
          fi
        }

        if [[ "$(params.deleteExisting)" == "true" ]] ; then
          cleandir
        fi

        test -z "$(params.httpProxy)" || export HTTP_PROXY=$(params.httpProxy)
        test -z "$(params.httpsProxy)" || export HTTPS_PROXY=$(params.httpsProxy)
        test -z "$(params.noProxy)" || export NO_PROXY=$(params.noProxy)

        /ko-app/git-init \
          -url "$(params.url)" \
          -revision "$(params.revision)" \
          -refspec "$(params.refspec)" \
          -path "$CHECKOUT_DIR" \
          -sslVerify="$(params.sslVerify)" \
          -submodules="$(params.submodules)" \
          -depth "$(params.depth)" \
          -sparseCheckoutDirectories "$(params.sparseCheckoutDirectories)"
        cd "$CHECKOUT_DIR"
        RESULT_SHA="$(git rev-parse HEAD)"
        EXIT_CODE="$?"
        if [ "$EXIT_CODE" != 0 ] ; then
          exit $EXIT_CODE
        fi
        # ensure we don't add a trailing newline to the result
        echo -n "$RESULT_SHA" > $(results.commit.path)
        echo -n "$(params.url)" > $(results.url.path)
`
)
