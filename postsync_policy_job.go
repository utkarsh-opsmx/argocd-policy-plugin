package main

import (
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func createPostsyncPolicyJobManifest(payloads []string) (unstructured.Unstructured, error) {

	// payloads, err := preparePayloads(images)
	// if err != nil {
	// 	return unstructured.Unstructured{}, err
	// }

	command := preparePostsyncCommand(payloads)

	var backOffLimit int32 = 0
	jobSpec := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "postsync-job-",
			Annotations: map[string]string{
				"argocd.argoproj.io/hook": "PostSync",
			},
		},
		Spec: batchv1.JobSpec{
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"opsmx.io/resource-owner": "opsmx",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:    "postsync-job",
							Image:   imagePolicyJob,
							Command: command,
						},
					},
					ServiceAccountName: "policy-job-account-name",
					RestartPolicy: v1.RestartPolicyNever,
				},
			},
			BackoffLimit: &backOffLimit,
		},
	}

	unstructuredManifest, err := runtime.DefaultUnstructuredConverter.ToUnstructured(jobSpec)
	if err != nil {
		return unstructured.Unstructured{}, err
	}

	return unstructured.Unstructured{Object: unstructuredManifest}, nil
}

func preparePostsyncCommand(payloads []string) []string {
	command := []string{"policy-job", "-t", token, "--submit-deployment-url", submitDeploymentUrl, "--sync-type", "postsync", "--repo-url", repoUrl, "--git-last-commitId", gitLastCommitId, "--target-environment", targetEnvironment}
	for _, payload := range payloads {
		command = append(command, "-p", payload)
	}

	return command
}
