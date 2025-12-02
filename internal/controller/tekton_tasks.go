/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// GitCloneTaskName is the name of the git-clone Task
	GitCloneTaskName = "git-clone"
	// BuildpacksPhasesTaskName is the name of the buildpacks-phases Task
	BuildpacksPhasesTaskName = "buildpacks-phases"
	// TaskVersionLabel is the label key for the task version
	TaskVersionLabel = "app.kubernetes.io/version"
	// ManagedByLabel is the label key for managed-by
	ManagedByLabel = "app.kubernetes.io/managed-by"
	// ManagedByValue is the value for managed-by label
	ManagedByValue = "zenith-operator"
)

// ensureTektonTasks ensures that the required Tekton Tasks exist in the given namespace.
// This is called before creating a PipelineRun to guarantee the Tasks are available.
func (r *FunctionReconciler) ensureTektonTasks(ctx context.Context, namespace string) error {
	logger := log.FromContext(ctx)

	// Ensure git-clone Task exists
	if err := r.ensureGitCloneTask(ctx, namespace); err != nil {
		logger.Error(err, "Failed to ensure git-clone Task", "namespace", namespace)
		return err
	}

	// Ensure buildpacks-phases Task exists
	if err := r.ensureBuildpacksPhasesTask(ctx, namespace); err != nil {
		logger.Error(err, "Failed to ensure buildpacks-phases Task", "namespace", namespace)
		return err
	}

	logger.Info("Tekton Tasks ensured successfully", "namespace", namespace)
	return nil
}

// ensureGitCloneTask ensures the git-clone Task exists in the namespace
func (r *FunctionReconciler) ensureGitCloneTask(ctx context.Context, namespace string) error {
	logger := log.FromContext(ctx)

	// Check if Task already exists
	existingTask := &tektonv1.Task{}
	err := r.Get(ctx, types.NamespacedName{Name: GitCloneTaskName, Namespace: namespace}, existingTask)
	if err == nil {
		// Task already exists, check if it's managed by us
		if existingTask.Labels[ManagedByLabel] == ManagedByValue {
			logger.V(1).Info("git-clone Task already exists and is managed by operator", "namespace", namespace)
			return nil
		}
		// Task exists but not managed by us, don't overwrite
		logger.Info("git-clone Task exists but not managed by operator, skipping", "namespace", namespace)
		return nil
	}

	if !errors.IsNotFound(err) {
		return err
	}

	// Create the Task
	task := r.buildGitCloneTask(namespace)
	if err := r.Create(ctx, task); err != nil {
		return err
	}

	logger.Info("Created git-clone Task", "namespace", namespace)
	return nil
}

// ensureBuildpacksPhasesTask ensures the buildpacks-phases Task exists in the namespace
func (r *FunctionReconciler) ensureBuildpacksPhasesTask(ctx context.Context, namespace string) error {
	logger := log.FromContext(ctx)

	// Check if Task already exists
	existingTask := &tektonv1.Task{}
	err := r.Get(ctx, types.NamespacedName{Name: BuildpacksPhasesTaskName, Namespace: namespace}, existingTask)
	if err == nil {
		// Task already exists, check if it's managed by us
		if existingTask.Labels[ManagedByLabel] == ManagedByValue {
			logger.V(1).Info("buildpacks-phases Task already exists and is managed by operator", "namespace", namespace)
			return nil
		}
		// Task exists but not managed by us, don't overwrite
		logger.Info("buildpacks-phases Task exists but not managed by operator, skipping", "namespace", namespace)
		return nil
	}

	if !errors.IsNotFound(err) {
		return err
	}

	// Create the Task
	task := r.buildBuildpacksPhasesTask(namespace)
	if err := r.Create(ctx, task); err != nil {
		return err
	}

	logger.Info("Created buildpacks-phases Task", "namespace", namespace)
	return nil
}

// buildGitCloneTask builds the git-clone Task definition
func (r *FunctionReconciler) buildGitCloneTask(namespace string) *tektonv1.Task {
	return &tektonv1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GitCloneTaskName,
			Namespace: namespace,
			Labels: map[string]string{
				TaskVersionLabel: "0.9",
				ManagedByLabel:   ManagedByValue,
			},
			Annotations: map[string]string{
				"tekton.dev/pipelines.minVersion": "0.38.0",
				"tekton.dev/categories":           "Git",
				"tekton.dev/tags":                 "git",
				"tekton.dev/displayName":          "git clone",
				"tekton.dev/platforms":            "linux/amd64,linux/s390x,linux/ppc64le,linux/arm64",
			},
		},
		Spec: tektonv1.TaskSpec{
			Description: `These Tasks are Git tasks to work with repositories used by other tasks in your Pipeline.

The git-clone Task will clone a repo from the provided url into the output Workspace. By default the repo will be cloned into the root of your Workspace. You can clone into a subdirectory by setting this Task's subdirectory param. This Task also supports sparse checkouts. To perform a sparse checkout, pass a list of comma separated directory patterns to this Task's sparseCheckoutDirectories param.`,
			Workspaces: []tektonv1.WorkspaceDeclaration{
				{Name: "output", Description: "The git repo will be cloned onto the volume backing this Workspace."},
				{Name: "ssh-directory", Optional: true, Description: "A .ssh directory with private key, known_hosts, config, etc. Copied to the user's home before git commands are executed. Used to authenticate with the git remote when performing the clone. Binding a Secret to this Workspace is strongly recommended over other volume types."},
				{Name: "basic-auth", Optional: true, Description: "A Workspace containing a .gitconfig and .git-credentials file. These will be copied to the user's home before any git commands are run. Any other files in this Workspace are ignored. It is strongly recommended to use ssh-directory over basic-auth whenever possible and to bind a Secret to this Workspace over other volume types."},
				{Name: "ssl-ca-directory", Optional: true, Description: "A workspace containing CA certificates, this will be used by Git to verify the peer with when fetching or pushing over HTTPS."},
			},
			Params: tektonv1.ParamSpecs{
				{Name: "url", Type: tektonv1.ParamTypeString, Description: "Repository URL to clone from."},
				{Name: "revision", Type: tektonv1.ParamTypeString, Description: "Revision to checkout. (branch, tag, sha, ref, etc...)", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: ""}},
				{Name: "refspec", Type: tektonv1.ParamTypeString, Description: "Refspec to fetch before checking out revision.", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: ""}},
				{Name: "submodules", Type: tektonv1.ParamTypeString, Description: "Initialize and fetch git submodules.", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "true"}},
				{Name: "depth", Type: tektonv1.ParamTypeString, Description: "Perform a shallow clone, fetching only the most recent N commits.", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "1"}},
				{Name: "sslVerify", Type: tektonv1.ParamTypeString, Description: "Set the `http.sslVerify` global git config. Setting this to `false` is not advised unless you are sure that you trust your git remote.", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "true"}},
				{Name: "crtFileName", Type: tektonv1.ParamTypeString, Description: "file name of mounted crt using ssl-ca-directory workspace. default value is ca-bundle.crt.", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "ca-bundle.crt"}},
				{Name: "subdirectory", Type: tektonv1.ParamTypeString, Description: "Subdirectory inside the `output` Workspace to clone the repo into.", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: ""}},
				{Name: "sparseCheckoutDirectories", Type: tektonv1.ParamTypeString, Description: "Define the directory patterns to match or exclude when performing a sparse checkout.", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: ""}},
				{Name: "deleteExisting", Type: tektonv1.ParamTypeString, Description: "Clean out the contents of the destination directory if it already exists before cloning.", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "true"}},
				{Name: "httpProxy", Type: tektonv1.ParamTypeString, Description: "HTTP proxy server for non-SSL requests.", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: ""}},
				{Name: "httpsProxy", Type: tektonv1.ParamTypeString, Description: "HTTPS proxy server for SSL requests.", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: ""}},
				{Name: "noProxy", Type: tektonv1.ParamTypeString, Description: "Opt out of proxying HTTP/HTTPS requests.", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: ""}},
				{Name: "verbose", Type: tektonv1.ParamTypeString, Description: "Log the commands that are executed during `git-clone`'s operation.", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "true"}},
				{Name: "gitInitImage", Type: tektonv1.ParamTypeString, Description: "The image providing the git-init binary that this Task runs.", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "ghcr.io/tektoncd/github.com/tektoncd/pipeline/cmd/git-init:v0.40.2"}},
				{Name: "userHome", Type: tektonv1.ParamTypeString, Description: "Absolute path to the user's home directory.", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "/home/git"}},
			},
			Results: []tektonv1.TaskResult{
				{Name: "commit", Description: "The precise commit SHA that was fetched by this Task."},
				{Name: "url", Description: "The precise URL that was fetched by this Task."},
				{Name: "committer-date", Description: "The epoch timestamp of the commit that was fetched by this Task."},
			},
			Steps: []tektonv1.Step{
				{
					Name:  "clone",
					Image: "$(params.gitInitImage)",
					Env: []corev1.EnvVar{
						{Name: "HOME", Value: "$(params.userHome)"},
						{Name: "PARAM_URL", Value: "$(params.url)"},
						{Name: "PARAM_REVISION", Value: "$(params.revision)"},
						{Name: "PARAM_REFSPEC", Value: "$(params.refspec)"},
						{Name: "PARAM_SUBMODULES", Value: "$(params.submodules)"},
						{Name: "PARAM_DEPTH", Value: "$(params.depth)"},
						{Name: "PARAM_SSL_VERIFY", Value: "$(params.sslVerify)"},
						{Name: "PARAM_CRT_FILENAME", Value: "$(params.crtFileName)"},
						{Name: "PARAM_SUBDIRECTORY", Value: "$(params.subdirectory)"},
						{Name: "PARAM_DELETE_EXISTING", Value: "$(params.deleteExisting)"},
						{Name: "PARAM_HTTP_PROXY", Value: "$(params.httpProxy)"},
						{Name: "PARAM_HTTPS_PROXY", Value: "$(params.httpsProxy)"},
						{Name: "PARAM_NO_PROXY", Value: "$(params.noProxy)"},
						{Name: "PARAM_VERBOSE", Value: "$(params.verbose)"},
						{Name: "PARAM_SPARSE_CHECKOUT_DIRECTORIES", Value: "$(params.sparseCheckoutDirectories)"},
						{Name: "PARAM_USER_HOME", Value: "$(params.userHome)"},
						{Name: "WORKSPACE_OUTPUT_PATH", Value: "$(workspaces.output.path)"},
						{Name: "WORKSPACE_SSH_DIRECTORY_BOUND", Value: "$(workspaces.ssh-directory.bound)"},
						{Name: "WORKSPACE_SSH_DIRECTORY_PATH", Value: "$(workspaces.ssh-directory.path)"},
						{Name: "WORKSPACE_BASIC_AUTH_DIRECTORY_BOUND", Value: "$(workspaces.basic-auth.bound)"},
						{Name: "WORKSPACE_BASIC_AUTH_DIRECTORY_PATH", Value: "$(workspaces.basic-auth.path)"},
						{Name: "WORKSPACE_SSL_CA_DIRECTORY_BOUND", Value: "$(workspaces.ssl-ca-directory.bound)"},
						{Name: "WORKSPACE_SSL_CA_DIRECTORY_PATH", Value: "$(workspaces.ssl-ca-directory.path)"},
					},
					SecurityContext: &corev1.SecurityContext{
						RunAsNonRoot: boolPtr(true),
						RunAsUser:    int64Ptr(65532),
					},
					Script: gitCloneScript,
				},
			},
		},
	}
}

// gitCloneScript is the shell script for the git-clone step
// This script uses the Tekton git-init binary (/ko-app/git-init) for cloning
const gitCloneScript = `#!/usr/bin/env sh
set -eu

if [ "${PARAM_VERBOSE}" = "true" ] ; then
  set -x
fi

if [ "${WORKSPACE_BASIC_AUTH_DIRECTORY_BOUND}" = "true" ] ; then
  cp "${WORKSPACE_BASIC_AUTH_DIRECTORY_PATH}/.git-credentials" "${PARAM_USER_HOME}/.git-credentials"
  cp "${WORKSPACE_BASIC_AUTH_DIRECTORY_PATH}/.gitconfig" "${PARAM_USER_HOME}/.gitconfig"
  chmod 400 "${PARAM_USER_HOME}/.git-credentials"
  chmod 400 "${PARAM_USER_HOME}/.gitconfig"
fi

if [ "${WORKSPACE_SSH_DIRECTORY_BOUND}" = "true" ] ; then
  cp -R "${WORKSPACE_SSH_DIRECTORY_PATH}" "${PARAM_USER_HOME}"/.ssh
  chmod 700 "${PARAM_USER_HOME}"/.ssh
  chmod -R 400 "${PARAM_USER_HOME}"/.ssh/*
fi

if [ "${WORKSPACE_SSL_CA_DIRECTORY_BOUND}" = "true" ] ; then
   export GIT_SSL_CAPATH="${WORKSPACE_SSL_CA_DIRECTORY_PATH}"
   if [ "${PARAM_CRT_FILENAME}" != "" ] ; then
      export GIT_SSL_CAINFO="${WORKSPACE_SSL_CA_DIRECTORY_PATH}/${PARAM_CRT_FILENAME}"
   fi
fi
CHECKOUT_DIR="${WORKSPACE_OUTPUT_PATH}/${PARAM_SUBDIRECTORY}"

cleandir() {
  # Delete any existing contents of the repo directory if it exists.
  #
  # We don't just "rm -rf ${CHECKOUT_DIR}" because ${CHECKOUT_DIR} might be "/"
  # or the root of a mounted volume.
  if [ -d "${CHECKOUT_DIR}" ] ; then
    # Delete non-hidden files and directories
    rm -rf "${CHECKOUT_DIR:?}"/*
    # Delete files and directories starting with . but excluding ..
    rm -rf "${CHECKOUT_DIR}"/.[!.]*
    # Delete files and directories starting with .. plus any other character
    rm -rf "${CHECKOUT_DIR}"/..?*
  fi
}

if [ "${PARAM_DELETE_EXISTING}" = "true" ] ; then
  cleandir || true
fi

test -z "${PARAM_HTTP_PROXY}" || export HTTP_PROXY="${PARAM_HTTP_PROXY}"
test -z "${PARAM_HTTPS_PROXY}" || export HTTPS_PROXY="${PARAM_HTTPS_PROXY}"
test -z "${PARAM_NO_PROXY}" || export NO_PROXY="${PARAM_NO_PROXY}"

git config --global --add safe.directory "${WORKSPACE_OUTPUT_PATH}"
/ko-app/git-init \
  -url="${PARAM_URL}" \
  -revision="${PARAM_REVISION}" \
  -refspec="${PARAM_REFSPEC}" \
  -path="${CHECKOUT_DIR}" \
  -sslVerify="${PARAM_SSL_VERIFY}" \
  -submodules="${PARAM_SUBMODULES}" \
  -depth="${PARAM_DEPTH}" \
  -sparseCheckoutDirectories="${PARAM_SPARSE_CHECKOUT_DIRECTORIES}"
cd "${CHECKOUT_DIR}"
RESULT_SHA="$(git rev-parse HEAD)"
EXIT_CODE="$?"
if [ "${EXIT_CODE}" != 0 ] ; then
  exit "${EXIT_CODE}"
fi
RESULT_COMMITTER_DATE="$(git log -1 --pretty=%ct)"
printf "%s" "${RESULT_COMMITTER_DATE}" > "$(results.committer-date.path)"
printf "%s" "${RESULT_SHA}" > "$(results.commit.path)"
printf "%s" "${PARAM_URL}" > "$(results.url.path)"
`

// buildBuildpacksPhasesTask builds the buildpacks-phases Task definition
func (r *FunctionReconciler) buildBuildpacksPhasesTask(namespace string) *tektonv1.Task {
	return &tektonv1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      BuildpacksPhasesTaskName,
			Namespace: namespace,
			Labels: map[string]string{
				TaskVersionLabel: "0.4",
				ManagedByLabel:   ManagedByValue,
			},
			Annotations: map[string]string{
				"tekton.dev/categories":           "Image Build, Security",
				"tekton.dev/pipelines.minVersion": "0.62.0",
				"tekton.dev/tags":                 "image-build",
				"tekton.dev/displayName":          "Buildpacks phases",
				"tekton.dev/platforms":            "linux/amd64",
			},
		},
		Spec: tektonv1.TaskSpec{
			Description: `The Buildpacks-Phases task builds source into a container image and pushes it to a registry, using Cloud Native Buildpacks - https://buildpacks.io/. This task separately calls the aspects of the Cloud Native Buildpacks lifecycle, to provide increased security via container isolation.

When the builder image includes extensions (= Dockerfiles), then this task will execute them. That allows to by example install packages, rpm, etc and to customize the build process according to your needs.

This task supports the Platform spec 0.13: https://github.com/buildpacks/spec/blob/platform/v0.13/platform.md`,
			Workspaces: []tektonv1.WorkspaceDeclaration{
				{Name: "source", Description: "Directory where application source is located."},
				{Name: "cache", Optional: true, Description: "Directory where cache is stored (when no cache image is provided)."},
			},
			Params: tektonv1.ParamSpecs{
				{Name: "CNB_BUILD_IMAGE", Type: tektonv1.ParamTypeString, Description: "Reference to the current build image in an OCI registry (if used <kaniko-dir> must be provided)", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: ""}},
				{Name: "CNB_BUILDER_IMAGE", Type: tektonv1.ParamTypeString, Description: "The Builder image which includes the lifecycle tool, the buildpacks and metadata."},
				{Name: "CNB_CACHE_IMAGE", Type: tektonv1.ParamTypeString, Description: "Reference to a cache image in an OCI registry (if no cache workspace is provided).", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: ""}},
				{Name: "CNB_ENV_VARS", Type: tektonv1.ParamTypeArray, Description: "Environment variables to set during _build-time_.", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeArray, ArrayVal: []string{}}},
				{Name: "CNB_EXPERIMENTAL_MODE", Type: tektonv1.ParamTypeString, Description: "Control the lifecycle's execution according to the mode silent, warn, error for the experimental features.", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "silent"}},
				{Name: "CNB_GROUP_ID", Type: tektonv1.ParamTypeString, Description: "The group ID of the builder image user.", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: ""}},
				{Name: "CNB_INSECURE_REGISTRIES", Type: tektonv1.ParamTypeString, Description: "List of registries separated by a comma having a self-signed certificate where TLS verification will be skipped.", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: ""}},
				{Name: "CNB_LAYERS_DIR", Type: tektonv1.ParamTypeString, Description: "Path to layers directory", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "/layers"}},
				{Name: "CNB_LOG_LEVEL", Type: tektonv1.ParamTypeString, Description: "Logging level values info, warning, error, debug", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "info"}},
				{Name: "CNB_PLATFORM_API_SUPPORTED", Type: tektonv1.ParamTypeString, Description: "Buildpack Platform API supported by the Tekton task", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "0.13"}},
				{Name: "CNB_PLATFORM_API", Type: tektonv1.ParamTypeString, Description: "User's Buildpack Platform API", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: ""}},
				{Name: "CNB_PLATFORM_DIR", Type: tektonv1.ParamTypeString, Description: "Path to the platform directory", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "/platform"}},
				{Name: "CNB_PROCESS_TYPE", Type: tektonv1.ParamTypeString, Description: "Default process type to set in the exported image", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "web"}},
				{Name: "CNB_RUN_IMAGE", Type: tektonv1.ParamTypeString, Description: "Reference to an image which is packaging the application runtime to be launched.", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: ""}},
				{Name: "CNB_SKIP_LAYERS", Type: tektonv1.ParamTypeString, Description: "Do not restore SBOM layer from previous image", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "false"}},
				{Name: "CNB_USER_ID", Type: tektonv1.ParamTypeString, Description: "The user ID of the builder image user.", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: ""}},
				{Name: "APP_IMAGE", Type: tektonv1.ParamTypeString, Description: "The name of the container image for your application."},
				{Name: "SOURCE_SUBPATH", Type: tektonv1.ParamTypeString, Description: "A subpath within the `source` input where the source to build is located.", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: ""}},
				{Name: "TAGS", Type: tektonv1.ParamTypeString, Description: "Additional tag to apply to the exported image", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: ""}},
				{Name: "USER_HOME", Type: tektonv1.ParamTypeString, Description: "Absolute path to the user's home directory.", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "/tekton/home"}},
				{Name: "INSPECT_TOOLS_IMAGE", Type: tektonv1.ParamTypeString, Description: "Image packaging tools like skopeo and jq to inspect the builder images", Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "quay.io/halkyonio/skopeo-jq:0.1.3@sha256:1b3d21ad541227dc9d3e793d18cef9eb00a969c0c01eb09cab88997bc63680c6"}},
			},
			Results: []tektonv1.TaskResult{
				{Name: "APP_IMAGE_DIGEST", Description: "The digest of the built `APP_IMAGE`."},
			},
			StepTemplate: &tektonv1.StepTemplate{
				Env: []corev1.EnvVar{
					{Name: "CNB_EXPERIMENTAL_MODE", Value: "$(params.CNB_EXPERIMENTAL_MODE)"},
					{Name: "HOME", Value: "$(params.USER_HOME)"},
				},
			},
			Volumes: []corev1.Volume{
				{Name: "tekton-home-dir", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
				{Name: "layers-dir", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
				{Name: "kaniko-dir", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
				{Name: "platform-dir", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
			},
			Steps: buildBuildpacksSteps(),
		},
	}
}

// buildBuildpacksSteps returns the steps for the buildpacks-phases Task
func buildBuildpacksSteps() []tektonv1.Step {
	return []tektonv1.Step{
		{
			Name:  "get-labels-and-env",
			Image: "$(params.INSPECT_TOOLS_IMAGE)",
			Env: []corev1.EnvVar{
				{Name: "PARAM_VERBOSE", Value: "$(params.CNB_LOG_LEVEL)"},
				{Name: "PARAM_BUILDER_IMAGE", Value: "$(params.CNB_BUILDER_IMAGE)"},
				{Name: "PARAM_CNB_PLATFORM_API", Value: "$(params.CNB_PLATFORM_API)"},
				{Name: "PARAM_CNB_PLATFORM_API_SUPPORTED", Value: "$(params.CNB_PLATFORM_API_SUPPORTED)"},
			},
			Results: []tektonv1.StepResult{
				{Name: "UID", Description: "UID of the user specified in the Builder image"},
				{Name: "GID", Description: "GID of the user specified in the Builder image"},
				{Name: "EXTENSION_LABELS", Description: "Extensions labels: io.buildpacks.extension.layers defined in the Builder image"},
				{Name: "CNB_PLATFORM_API", Description: "The CNB_PLATFORM_API to be used by lifecycle and verified against the one supported by this task"},
			},
			Script: getLabelsAndEnvScript,
		},
		{
			Name:  "prepare",
			Image: "registry.access.redhat.com/ubi8/ubi-minimal@sha256:b2a1bec3dfbc7a14a1d84d98934dfe8fdde6eb822a211286601cf109cbccb075",
			Args:  []string{"--env-vars", "$(params.CNB_ENV_VARS[*])"},
			Env: []corev1.EnvVar{
				{Name: "CNB_USER_ID", Value: "$(steps.get-labels-and-env.results.UID)"},
				{Name: "CNB_GROUP_ID", Value: "$(steps.get-labels-and-env.results.GID)"},
			},
			Script: prepareScript,
			VolumeMounts: []corev1.VolumeMount{
				{Name: "layers-dir", MountPath: "/layers"},
				{Name: "platform-dir", MountPath: "$(params.CNB_PLATFORM_DIR)"},
			},
		},
		{
			Name:            "analyze",
			Image:           "$(params.CNB_BUILDER_IMAGE)",
			ImagePullPolicy: corev1.PullAlways,
			Command:         []string{"/cnb/lifecycle/analyzer"},
			Env: []corev1.EnvVar{
				{Name: "CNB_PLATFORM_API", Value: "$(steps.get-labels-and-env.results.CNB_PLATFORM_API)"},
			},
			Args: []string{
				"-log-level=$(params.CNB_LOG_LEVEL)",
				"-layers=$(params.CNB_LAYERS_DIR)",
				"-run-image=$(params.CNB_RUN_IMAGE)",
				"-cache-image=$(params.CNB_CACHE_IMAGE)",
				"-uid=$(steps.get-labels-and-env.results.UID)",
				"-gid=$(steps.get-labels-and-env.results.GID)",
				"-insecure-registry=$(params.CNB_INSECURE_REGISTRIES)",
				"-tag=$(params.TAGS)",
				"-skip-layers=$(params.CNB_SKIP_LAYERS)",
				"$(params.APP_IMAGE)",
			},
			VolumeMounts: []corev1.VolumeMount{
				{Name: "layers-dir", MountPath: "/layers"},
			},
		},
		{
			Name:            "detect",
			Image:           "$(params.CNB_BUILDER_IMAGE)",
			ImagePullPolicy: corev1.PullAlways,
			Command:         []string{"/cnb/lifecycle/detector"},
			Env: []corev1.EnvVar{
				{Name: "CNB_PLATFORM_API", Value: "$(steps.get-labels-and-env.results.CNB_PLATFORM_API)"},
			},
			Args: []string{
				"-log-level=$(params.CNB_LOG_LEVEL)",
				"-app=$(workspaces.source.path)/$(params.SOURCE_SUBPATH)",
				"-group=/layers/group.toml",
				"-plan=/layers/plan.toml",
				"-layers=$(params.CNB_LAYERS_DIR)",
				"-platform=$(params.CNB_PLATFORM_DIR)",
			},
			VolumeMounts: []corev1.VolumeMount{
				{Name: "layers-dir", MountPath: "/layers"},
				{Name: "platform-dir", MountPath: "$(params.CNB_PLATFORM_DIR)"},
				{Name: "tekton-home-dir", MountPath: "/tekton/home"},
			},
		},
		{
			Name:            "restore",
			Image:           "$(params.CNB_BUILDER_IMAGE)",
			ImagePullPolicy: corev1.PullAlways,
			Env: []corev1.EnvVar{
				{Name: "UID", Value: "$(steps.get-labels-and-env.results.UID)"},
				{Name: "GID", Value: "$(steps.get-labels-and-env.results.GID)"},
				{Name: "CNB_LOG_LEVEL", Value: "$(params.CNB_LOG_LEVEL)"},
				{Name: "CNB_BUILD_IMAGE", Value: "$(params.CNB_BUILD_IMAGE)"},
				{Name: "CNB_BUILDER_IMAGE", Value: "$(params.CNB_BUILDER_IMAGE)"},
				{Name: "CNB_CACHE_IMAGE", Value: "$(params.CNB_CACHE_IMAGE)"},
				{Name: "CNB_INSECURE_REGISTRIES", Value: "$(params.CNB_INSECURE_REGISTRIES)"},
				{Name: "CNB_SKIP_LAYERS", Value: "$(params.CNB_SKIP_LAYERS)"},
				{Name: "CNB_PLATFORM_API", Value: "$(steps.get-labels-and-env.results.CNB_PLATFORM_API)"},
			},
			Script: restoreScript,
			VolumeMounts: []corev1.VolumeMount{
				{Name: "layers-dir", MountPath: "/layers"},
				{Name: "kaniko-dir", MountPath: "/kaniko"},
			},
		},
		{
			Name:            "extender",
			Image:           "$(params.CNB_BUILDER_IMAGE)",
			ImagePullPolicy: corev1.PullAlways,
			Command:         []string{"/cnb/lifecycle/extender"},
			// Only run extender when builder image includes extensions (EXTENSION_LABELS is NOT "empty")
			When: tektonv1.StepWhenExpressions{
				{
					Input:    "$(steps.get-labels-and-env.results.EXTENSION_LABELS)",
					Operator: selection.NotIn,
					Values:   []string{"empty"},
				},
			},
			Env: []corev1.EnvVar{
				{Name: "CNB_PLATFORM_API", Value: "$(steps.get-labels-and-env.results.CNB_PLATFORM_API)"},
			},
			Args: []string{
				"-log-level=$(params.CNB_LOG_LEVEL)",
				"-app=$(workspaces.source.path)/$(params.SOURCE_SUBPATH)",
				"-generated=/layers/generated",
				"-uid=$(steps.get-labels-and-env.results.UID)",
				"-gid=$(steps.get-labels-and-env.results.GID)",
				"-platform=$(params.CNB_PLATFORM_DIR)",
			},
			SecurityContext: &corev1.SecurityContext{
				RunAsUser:  int64Ptr(0),
				RunAsGroup: int64Ptr(0),
				Capabilities: &corev1.Capabilities{
					Add: []corev1.Capability{"SYS_ADMIN", "SETFCAP"},
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{Name: "layers-dir", MountPath: "/layers"},
				{Name: "kaniko-dir", MountPath: "/kaniko"},
				{Name: "tekton-home-dir", MountPath: "/tekton/home"},
				{Name: "platform-dir", MountPath: "$(params.CNB_PLATFORM_DIR)"},
			},
		},
		{
			Name:            "build",
			Image:           "$(params.CNB_BUILDER_IMAGE)",
			ImagePullPolicy: corev1.PullAlways,
			Command:         []string{"/cnb/lifecycle/builder"},
			// Only run build when builder image does NOT include extensions (EXTENSION_LABELS is "empty")
			When: tektonv1.StepWhenExpressions{
				{
					Input:    "$(steps.get-labels-and-env.results.EXTENSION_LABELS)",
					Operator: selection.In,
					Values:   []string{"empty"},
				},
			},
			Env: []corev1.EnvVar{
				{Name: "CNB_PLATFORM_API", Value: "$(steps.get-labels-and-env.results.CNB_PLATFORM_API)"},
			},
			Args: []string{
				"-log-level=$(params.CNB_LOG_LEVEL)",
				"-app=$(workspaces.source.path)/$(params.SOURCE_SUBPATH)",
				"-layers=$(params.CNB_LAYERS_DIR)",
				"-group=/layers/group.toml",
				"-plan=/layers/plan.toml",
				"-platform=$(params.CNB_PLATFORM_DIR)",
			},
			VolumeMounts: []corev1.VolumeMount{
				{Name: "layers-dir", MountPath: "/layers"},
				{Name: "platform-dir", MountPath: "$(params.CNB_PLATFORM_DIR)"},
				{Name: "tekton-home-dir", MountPath: "/tekton/home"},
			},
		},
		{
			Name:            "export",
			Image:           "$(params.CNB_BUILDER_IMAGE)",
			ImagePullPolicy: corev1.PullAlways,
			Command:         []string{"/cnb/lifecycle/exporter"},
			Env: []corev1.EnvVar{
				{Name: "CNB_PLATFORM_API", Value: "$(steps.get-labels-and-env.results.CNB_PLATFORM_API)"},
			},
			Args: []string{
				"-log-level=$(params.CNB_LOG_LEVEL)",
				"-app=$(workspaces.source.path)/$(params.SOURCE_SUBPATH)",
				"-layers=$(params.CNB_LAYERS_DIR)",
				"-group=/layers/group.toml",
				"-cache-dir=$(workspaces.cache.path)",
				"-cache-image=$(params.CNB_CACHE_IMAGE)",
				"-report=/layers/report.toml",
				"-process-type=$(params.CNB_PROCESS_TYPE)",
				"-uid=$(steps.get-labels-and-env.results.UID)",
				"-gid=$(steps.get-labels-and-env.results.GID)",
				"-insecure-registry=$(params.CNB_INSECURE_REGISTRIES)",
				"$(params.APP_IMAGE)",
			},
			VolumeMounts: []corev1.VolumeMount{
				{Name: "layers-dir", MountPath: "/layers"},
			},
		},
		{
			Name:   "results",
			Image:  "registry.access.redhat.com/ubi8/python-311@sha256:43605cb2491ef2297a7acf4b4bf0b7f54f0c91b96daf12ae41c49cc7f192b153",
			Script: resultsScript,
			VolumeMounts: []corev1.VolumeMount{
				{Name: "layers-dir", MountPath: "/layers"},
			},
		},
	}
}

// int64Ptr returns a pointer to an int64
func int64Ptr(i int64) *int64 {
	return &i
}

// boolPtr returns a pointer to a bool
func boolPtr(b bool) *bool {
	return &b
}

// getLabelsAndEnvScript is the script for the get-labels-and-env step
const getLabelsAndEnvScript = `#!/usr/bin/env bash
set -eu

if [ "${PARAM_VERBOSE}" = "debug" ] ; then
  set -x
fi

echo # Check if registry creds docker file has been mounted from a secret"
if [[ -f "$HOME/.docker/config.json" ]]; then
  printf %"s\n" "The docker config.json file exists !"
else
  printf %"s\n" "!!!!! Warning: No registry credentials file exist. So it could be possible that the task will fail due to docker rate limit, etc !!!"
fi

printf %"s\n" "Remove the @sha from the image as not supported by skopeo to inspect an image"
CLEANED_IMAGE="${PARAM_BUILDER_IMAGE%@*}"

EXT_LABEL_1="io.buildpacks.extension.layers"
EXT_LABEL_2="io.buildpacks.buildpack.order-extensions"
BUILDER_LABEL="io.buildpacks.builder.metadata"

IMG_MANIFEST=$(skopeo inspect --authfile $HOME/.docker/config.json "docker://${CLEANED_IMAGE}")

IMG_LABELS=$(echo $IMG_MANIFEST | jq -e '.Labels')

if [[ $(echo "$IMG_LABELS" | jq -r '.["'${BUILDER_LABEL}'"]') != "{}" ]] > /dev/null; then
  printf %"s\n" "## The builder image ${PARAM_BUILDER_IMAGE} includes the label: \"${BUILDER_LABEL}\" :"

  builderLabel=$(echo -n "$IMG_LABELS" | jq -r '.["'${BUILDER_LABEL}'"]')
  platforms=($(echo $builderLabel | jq -r '.lifecycle.apis.platform.supported'))
  printf %"s\n" "Lifecycle platforms API supported: ${platforms[@]}"

  CNB_PLATFORM_API=${PARAM_CNB_PLATFORM_API:-$PARAM_CNB_PLATFORM_API_SUPPORTED}
  echo "Platform API selected: $CNB_PLATFORM_API"
  printf %"s\n" "Platform API supported by this task: $PARAM_CNB_PLATFORM_API_SUPPORTED"

  if [[ "${platforms[@]}" =~ "$CNB_PLATFORM_API" && "$CNB_PLATFORM_API" == "$PARAM_CNB_PLATFORM_API_SUPPORTED" ]]; then
      echo -n "$CNB_PLATFORM_API" > "$(step.results.CNB_PLATFORM_API.path)"
      printf %"s\n" "$CNB_PLATFORM_API is in the list of the platform supported by lifecycle like also this Tekton task :-)"
  else
      echo "$PARAM_CNB_PLATFORM_API is not in the list of the supported platform by lifecycle or is not supported by this tekton task: ${PARAM_CNB_PLATFORM_API_SUPPORTED} !"
      exit 1
  fi
fi

if [[ $(echo "$IMG_LABELS" | jq -r '.["'${EXT_LABEL_1}'"]') != "{}" ]] > /dev/null; then
  echo "## The builder image ${PARAM_BUILDER_IMAGE} includes some extensions as the extension label \"${EXT_LABEL_1}\" is NOT empty:"
  echo -n "$IMG_LABELS" | jq -r '.["'${EXT_LABEL_1}'"]' | tee "$(step.results.EXTENSION_LABELS.path)"
  echo ""
else
  echo "## The builder image ${PARAM_BUILDER_IMAGE} dot not include extensions as the extension label \"${EXT_LABEL_1}\" is empty !"
  echo -n "empty" | tee "$(step.results.EXTENSION_LABELS.path)"
fi

CNB_USER_ID=$(echo $IMG_MANIFEST | jq -r '.Env' | jq -r '.[] | select(test("^CNB_USER_ID="))'  | cut -d '=' -f 2)
CNB_GROUP_ID=$(echo $IMG_MANIFEST | jq -r '.Env' | jq -r '.[] | select(test("^CNB_GROUP_ID="))' | cut -d '=' -f 2)

echo "## The CNB_USER_ID & CNB_GROUP_ID defined within the builder image: ${PARAM_BUILDER_IMAGE} are:"
echo -n "$CNB_USER_ID"  | tee "$(step.results.UID.path)"
echo ""
echo -n "$CNB_GROUP_ID" | tee "$(step.results.GID.path)"
`

// prepareScript is the script for the prepare step
const prepareScript = `#!/usr/bin/env bash
set -eu

echo "CNB UID: $CNB_USER_ID"
echo "CNB GID: $CNB_GROUP_ID"

if [[ "$(workspaces.cache.bound)" == "true" ]]; then
  echo "--> Setting permissions on '$(workspaces.cache.path)'..."
  chown -R "$CNB_USER_ID:$CNB_GROUP_ID" "$(workspaces.cache.path)"
fi

echo "--> Creating .docker folder"
mkdir -p "/tekton/home/.docker"

for path in "/tekton/home" "/tekton/home/.docker" "/tekton/creds" "/layers" "$(workspaces.source.path)"; do
  echo "--> Setting permissions on '$path'..."
  chown -R "$CNB_USER_ID:$CNB_GROUP_ID" "$path"
done

echo "--> Parsing additional configuration..."
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

echo "--> Processing any environment variables..."
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
echo "--> Content of $(params.CNB_PLATFORM_DIR)/env"
ls -la $(params.CNB_PLATFORM_DIR)/env

echo "--> Show the project cloned within the workspace ..."
ls -la $(workspaces.source.path)/$(params.SOURCE_SUBPATH)
`

// restoreScript is the script for the restore step
const restoreScript = `#!/usr/bin/env bash
export BUILD_IMAGE=${CNB_BUILD_IMAGE:-${CNB_BUILDER_IMAGE}}
/cnb/lifecycle/restorer \
  -log-level=${CNB_LOG_LEVEL} \
  -build-image=${BUILD_IMAGE} \
  -group=/layers/group.toml \
  -layers=${CNB_LAYERS_DIR} \
  -cache-dir=$(workspaces.cache.path) \
  -cache-image=${CNB_CACHE_IMAGE} \
  -uid=${UID} \
  -gid=${GID} \
  -insecure-registry=${CNB_INSECURE_REGISTRIES} \
  -skip-layers=${CNB_SKIP_LAYERS}
`

// resultsScript is the script for the results step
const resultsScript = `#!/usr/bin/env python3

import tomllib

def write_to_file(filename, content):
  with open(filename, "w") as f:
    f.write(content)

with open("/layers/report.toml", "rb") as f:
    data = tomllib.load(f)

img_data = data.get("image")

tags = img_data.get("tags")
digest = img_data.get("digest")
image_id = img_data.get("image_id")
manifest_size = img_data.get("manifest_size")

print("#### Image data ####")
print(f"tags: {tags}")
print(f"Digest: {digest}")

if None not in (image_id, manifest_size):
  print(f"image container id (when using daemon): {image_id}, manifest size: {manifest_size}")

write_to_file('$(results.APP_IMAGE_DIGEST.path)',digest)
`
