package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8yaml "k8s.io/apimachinery/pkg/util/yaml"
)

type ManifestKind string

const (
	ManifestKindDeployment ManifestKind = "Deployment"
)

type ManifestVersion string

const (
	ManifestVersionAppsV1 ManifestVersion = "apps/v1"
)

func getFilesOfTypeYamlJson(root string) ([]string, error) {
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) == ".yaml" || filepath.Ext(path) == ".yml" || filepath.Ext(path) == ".json" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return files, err
	}

	return files, nil
}

func readFilesAsManifests(paths []string) (result []unstructured.Unstructured, errs []error) {
	for _, path := range paths {
		rawdata, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Errorf("could not read file: %s from disk: %s", path, err))
		}

		manifest, err := readManifestData(bytes.NewReader(rawdata))
		if err != nil {
			errs = append(errs, fmt.Errorf("could not read file: %s from disk: %s", path, err))
		}
		result = append(result, manifest...)
	}

	return result, errs
}

func readManifestData(yamlData io.Reader) ([]unstructured.Unstructured, error) {
	decoder := k8yaml.NewYAMLOrJSONDecoder(yamlData, 1)

	var manifests []unstructured.Unstructured
	for {
		nxtManifest := unstructured.Unstructured{}
		err := decoder.Decode(&nxtManifest)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		// Skip empty manifests
		if len(nxtManifest.Object) > 0 {
			manifests = append(manifests, nxtManifest)
		}
	}

	return manifests, nil
}

func getImagesFromUnstructuredData(unstructured []unstructured.Unstructured) ([]string, error) {
	images := make([]string, 0)

	for _, manifest := range unstructured {

		var obj appsv1.Deployment
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(manifest.Object, &obj); err != nil {
			return []string{}, err
		}

		if obj.Kind == string(ManifestKindDeployment) && obj.APIVersion == string(ManifestVersionAppsV1) {
			images = append(images, getImagesFromDeploymentManifest(obj)...)
		}

		//we can add more manifest kinds here if need be
	}

	return images, nil
}

func getImagesFromDeploymentManifest(deployment appsv1.Deployment) []string {
	images := make([]string, 0)
	for _, image := range deployment.Spec.Template.Spec.Containers {
		images = append(images, image.Image)
	}

	return images
}

func toYAML(data unstructured.Unstructured) (string, error) {
	res, err := yaml.Marshal(data.Object)
	if err != nil {
		return "", fmt.Errorf("toYAML: could not export into YAML: %s", err)
	}
	return string(res), nil
}
