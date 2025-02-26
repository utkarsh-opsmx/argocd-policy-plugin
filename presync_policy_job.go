package main

import (
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func createPresyncPolicyJobManifest(payloads []string) (unstructured.Unstructured, error) {

	// payloads, err := preparePayloads(images)
	// if err != nil {
	// 	return unstructured.Unstructured{}, err
	// }

	command := preparePresyncCommand(payloads)

	var backOffLimit int32 = 0
	jobSpec := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "presync-job-",
			Annotations: map[string]string{
				"argocd.argoproj.io/hook": "PreSync",
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
							Name:    "presync-job",
							Image:   imagePolicyJob,
							Command: command,
						},
					},
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



func splitImageName(image string) (imageName, imageTag string) {

	imageName = image
	imageTag = "latest"

	ix := strings.LastIndex(image, ":")
	if ix != -1 {
		imageName = image[:ix]
		imageTag = image[ix+1:]
	}
	return
}

func preparePresyncCommand(payloads []string) []string {
	command := []string{"policy-job", "-u", releaseCheckUrl, "-t", token, "-s", servicenowCheckUrl, "-c", gitMessage, "-b", gitBranch, "--sync-type", "presync"}
	for _, payload := range payloads {
		command = append(command, "-p", payload)
	}

	return command
}
