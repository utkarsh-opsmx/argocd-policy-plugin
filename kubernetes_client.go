package main

import (
	"encoding/json"
    "os/exec"
)

func getDeploymentIdAndSealId() error {
	cmd := "kubectl"
	//kubectl get app <appname> -o jsonpath='{.metadata.labels}'
	arg0 := "get app"
	arg1 := argocdAppName
	arg2 := "-o jsonpath"
	arg3 := "'{.metadata.labels}'"
	labelsJson, err := exec.Command(cmd, arg0,arg1,arg2,arg3).Output()
	if err != nil {
		return err;
	}
	var labels map[string]string
	if err := json.Unmarshal(labelsJson, &labels); err != nil {
		return err
	}
	sealId = labels["sealId"]
	deploymentId = labels["deploymentId"]
	return nil
}

// func getImageNameToBranchMapping() (map[string]string, error) {
// 	cmd := "kubectl"
// 	//kubectl get app <appname> -o jsonpath='{.metadata.labels}'
// 	arg0 := "get app"
// 	arg1 := argocdAppName
// 	arg2 := "-o jsonpath"
// 	arg3 := "'{.spec.sources}'"
// 	sourcesJson, err := exec.Command(cmd, arg0,arg1,arg2,arg3).Output()
// 	if err != nil {
// 		return map[string]string{}, err;
// 	}
// 	var sources []map[string]string
// 	if err := json.Unmarshal(sourcesJson, &sources); err != nil {
// 		return map[string]string{}, err;
// 	}
// 	var results map[string]string
// 	for source := range sources {
// 		imageName := extractImageNamefromRepoUrl(source["repoURL"])
// 		results[imageName] = source["targetRevision"]
// 	}
// 	return results, nil
// }

// func extractImageNamefromRepoUrl() string {

// }