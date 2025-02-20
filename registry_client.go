package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
	"io"
)

var httpClient *http.Client

const (
	timeout    = 5
)

type Creds struct {
	auths				map[string]map[string]string
}

type DockerLabels struct {
	docker				Labels `json:"docker"`
}

type Labels struct {
	labels				ArtifactData `json:"labels"`
}

type ArtifactData struct {
	jetId				string `json:"jet-id"`
	sealId				string `json:"seal_Id"`
	projectName 		string `json:"project_key"`
}



func init() {
	//TODO: This may have to be changed in the future
	httpClient = NewHTTPClient(timeout)
}

func NewHTTPClient(clientTimeout int) *http.Client {
	return &http.Client{
		Timeout: time.Duration(clientTimeout) * time.Second,
	}
}

func getImageMetaDatas(images []string) ([]ImageMetaData, error) {
	imageMetaDatas := []ImageMetaData{}
	for _, image := range images {
		imageMetaData, err := getImageMetaDataFromImageName(image)
		if err != nil {
			return nil, err
		}
		imageMetaDatas = append(imageMetaDatas, imageMetaData)
	}
	return imageMetaDatas, nil
}


func getImageMetaDataFromImageName(image string) (ImageMetaData, error) {
	credsFromEnv := os.Getenv("REGISTRY_CREDS")
	regUrl := ""
	regToken := ""
	if credsFromEnv == "" {
		return ImageMetaData{}, fmt.Errorf("docker registry creds secret not found!")
	}

	creds :=  Creds{}
	err := json.Unmarshal([]byte(credsFromEnv), &creds)
	if err != nil {
		return ImageMetaData{}, err
	}
	for key, value := range creds.auths {
		regUrl = key
		regToken = value["auth"]
	}

	imageName, imageTag := splitImageName(image)
	regUrl=regUrl+"/artifactory"+"/"+imageName+"/"+imageTag+"/manifest.json"
	request, err := http.NewRequest(http.MethodGet, regUrl, nil)
	if err != nil {
		return ImageMetaData{}, err
	}

	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Authorization", "Bearer "+regToken)

	resp, err := httpClient.Do(request)
	if err != nil {
		return ImageMetaData{}, err
	}

	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return ImageMetaData{}, fmt.Errorf("request to fetch metadata for the image: %s failed with status code: %s",image, resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return ImageMetaData{}, err
	}

	data := DockerLabels{}
	if err := json.Unmarshal(content, &data); err != nil {
		return ImageMetaData{}, nil
	}
	result := ImageMetaData{
		ArtifactId					:	"???",
		ArtifactCreateDate			:	"???",
		JetId						:	data.docker.labels.jetId,
		SealId						:	data.docker.labels.sealId,
		DeploymentId				:	"???",
		ProjectName					:	data.docker.labels.projectName,
		artifactLocation			:	"artifactory/"+imageName,
	}
	return result, nil
}