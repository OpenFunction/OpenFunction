package controllers

const (
	// Tekton build template from https://raw.githubusercontent.com/tektoncd/catalog/master/task/buildpacks/0.2/buildpacks.yaml
	tmplBuild = `
---
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: buildpacks
  labels:
    app.kubernetes.io/version: "0.2"
  annotations:
    tekton.dev/pipelines.minVersion: "0.12.1"
    tekton.dev/tags: image-build
    tekton.dev/displayName: "buildpacks"
spec:
  description: >-
    The Buildpacks task builds source into a container image and pushes it to a registry,
    using Cloud Native Buildpacks.
    Cloud Native Buildpacks are pluggable, modular tools that transform application source code
    into OCI images. They replace Dockerfiles in the app development lifecycle, and allow for swift
    rebasing of images, and give modular control over images through the use of builders, among other
    benefits. This command uses a builder to construct the image, and pushes it to the registry provided.
  params:
    - name: BUILDER_IMAGE
      description: The image on which builds will run (must include lifecycle and compatible buildpacks).
    - name: CACHE
      description: The name of the persistent app cache volume.
      default: empty-dir
    - name: CACHE_IMAGE
      description: The name of the persistent app cache image.
      default: ""
    - name: PLATFORM_DIR
      description: The name of the platform directory.
      default: empty-dir
    - name: USER_ID
      description: The user ID of the builder image user.
      default: "1000"
    - name: GROUP_ID
      description: The group ID of the builder image user.
      default: "1000"
    - name: PROCESS_TYPE
      description: The default process type to set on the image.
      default: "web"
    - name: SOURCE_SUBPATH
      description: A subpath within the source input where the source to build is located.
      default: ""
    - name: SKIP_RESTORE
      description: Do not write layer metadata or restore cached layers
      default: "false"
    - name: RUN_IMAGE
      description: Reference to a run image to use
      default: ""

  resources:
    outputs:
      - name: image
        type: image

  workspaces:
    - name: source

  stepTemplate:
    env:
      - name: CNB_PLATFORM_API
        value: "0.3"

  steps:
    - name: prepare
      # Latest alpine as of Oct 22, 2020
      image: docker.io/alpine@sha256:203ee936961c0f491f72ce9d3c3c67d9440cdb1d61b9783cf340baa09308ffc1
      imagePullPolicy: Always
      command: ["/bin/sh"]
      args:
        - "-c"
        - |-
          chown -R "$(params.USER_ID):$(params.GROUP_ID)" "/tekton/home" &&
          chown -R "$(params.USER_ID):$(params.GROUP_ID)" "/layers" &&
          chown -R "$(params.USER_ID):$(params.GROUP_ID)" "/cache" &&
          chown -R "$(params.USER_ID):$(params.GROUP_ID)" "$(workspaces.source.path)"
      volumeMounts:
        - name: layers-dir
          mountPath: /layers
        - name: $(params.CACHE)
          mountPath: /cache

    - name: create
      image: $(params.BUILDER_IMAGE)
      imagePullPolicy: Always
      command: ["/cnb/lifecycle/creator"]
      args:
        - "-app=$(workspaces.source.path)/$(params.SOURCE_SUBPATH)"
        - "-cache-dir=/cache"
        - "-cache-image=$(params.CACHE_IMAGE)"
        - "-gid=$(params.GROUP_ID)"
        - "-layers=/layers"
        - "-platform=/platform"
        - "-process-type=$(params.PROCESS_TYPE)"
        - "-skip-restore=$(params.SKIP_RESTORE)"
        - "-previous-image=$(resources.outputs.image.url)"
        - "-run-image=$(params.RUN_IMAGE)"
        - "-uid=$(params.USER_ID)"
        - "$(resources.outputs.image.url)"
      volumeMounts:
        - name: layers-dir
          mountPath: /layers
        - name: $(params.CACHE)
          mountPath: /cache
        - name: $(params.PLATFORM_DIR)
          mountPath: /platform
      securityContext:
        runAsUser: 1000
        runAsGroup: 1000

  volumes:
    - name: empty-dir
      emptyDir: {}
    - name: layers-dir
      emptyDir: {}
`
	// Tekton GitClone task https://raw.githubusercontent.com/tektoncd/catalog/master/task/git-clone/0.2/git-clone.yaml
	tmplGitClone = `
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: git-clone
  labels:
    app.kubernetes.io/version: "0.2"
  annotations:
    tekton.dev/pipelines.minVersion: "0.12.1"
    tekton.dev/tags: git
    tekton.dev/displayName: "git clone"
spec:
  description: >-
    These Tasks are Git tasks to work with repositories used by other tasks
    in your Pipeline.

    The git-clone Task will clone a repo from the provided url into the
    output Workspace. By default the repo will be cloned into the root of
    your Workspace. You can clone into a subdirectory by setting this Task's
    subdirectory param.

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
      default: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init:v0.18.1"
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
          -depth "$(params.depth)"
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
