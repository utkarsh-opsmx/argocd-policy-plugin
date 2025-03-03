package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"encoding/json"
	"github.com/spf13/cobra"
)

//Code ref - https://github.com/argoproj-labs/argocd-vault-plugin/tree/main

const (
	StdIn      = "-"
	dummyToken = "dummy_token"
)

type JobPayload struct {
	OrganizationName 			string `json:"organizationName,omitempty"`
	ArtifactName  				string `json:"artifactName"`
	ArtifactTag     			string `json:"artifactTag"`
	ArtifactId					string `json:"artifactId"`
	ArtifactCreateDate			string `json:"artifactCreateDate"`
	JetId						string `json:"jetId"`
	SealId						string `json:"sealId"`
	DeploymentId				string `json:"deploymentId"`
	ProjectName					string `json:"projectName"`
	artifactLocation			string `json:"artifactLocation"`
}

type ImageMetaData struct {
	ArtifactId					string `json:"artifactId"`
	ArtifactCreateDate			string `json:"artifactCreateDate"`
	JetId						string `json:"jetId"`
	SealId						string `json:"sealId"`
	ProjectName					string `json:"projectName"`
	artifactLocation			string `json:"artifactLocation"`
}

var releaseCheckUrl, servicenowCheckUrl, organization, token, gitMessage, gitBranch, imagePolicyJob, argocdAppName string
var submitDeploymentUrl, repoUrl, gitLastCommitId, targetEnvironment string
var custom, deploymentId, sealId string

var rootCmd = &cobra.Command{
	Use:   "argocd-policy-plugin <path>",
	Short: "This is a plugin that adds a presync and postsync job to every application to communicate with evidence store",
	RunE: func(cmd *cobra.Command, args []string) error {
		var manifests []unstructured.Unstructured
		var err error
		var images []string

		if strings.TrimSpace(custom) != "" {
			return fmt.Errorf("custom : %s\n", custom)
		}

		if strings.TrimSpace(releaseCheckUrl) == "" {
			return errors.New("release-check-url flag has not been set for the argocd-policy-plugin binary")
		}

		if strings.TrimSpace(releaseCheckUrl) == "" {
			return errors.New("servicenow-check-url flag has not been set for the argocd-policy-plugin binary")
		}

		if strings.TrimSpace(organization) == "" {
			return errors.New("organization-name flag has not been set for the argocd-policy-plugin binary")
		}

		if strings.TrimSpace(token) == "" {
			token = dummyToken
		}

		if strings.TrimSpace(gitMessage) == "" {
			return errors.New("git-last-commit-message flag has not been set for the argocd-policy-plugin binary")
		}

		if strings.TrimSpace(gitBranch) == "" {
			return errors.New("git-branch flag has not been set for the argocd-policy-plugin binary")
		}

		if strings.TrimSpace(imagePolicyJob) == "" {
			return errors.New("image-policy-job flag has not been set for the image policy job binary")
		}

		path := args[0]
		if path == StdIn {
			manifests, err = readManifestData(cmd.InOrStdin())
			if err != nil {
				return err
			}
		} else {
			// This is supposed to work for simple kubernetes objects files with images but wont work for application spec since it  contains sources not images 
			files, err := getFilesOfTypeYamlJson(path)
			if len(files) < 1 {
				return fmt.Errorf("no YAML or JSON files were found in %s", path)
			}
			if err != nil {
				return err
			}

			var errs []error
			manifests, errs = readFilesAsManifests(files)
			if len(errs) != 0 {
				errMessages := make([]string, len(errs))
				for idx, err := range errs {
					errMessages[idx] = err.Error()
				}
				return fmt.Errorf("could not read YAML/JSON files:\n%s", strings.Join(errMessages, "\n"))
			}
		}

		images, err = getImagesFromUnstructuredData(manifests)
		if err != nil {
			return fmt.Errorf("error in argocd-policy-plugin - getImagesFromUnstructuredData %v", err)
		}

		imageMetaDatas, err := getImageMetaDatas(images)
		if err != nil {
			return fmt.Errorf("error while fetching image metadata from docker registry %v", err)
		}
		if err := getDeploymentIdAndSealId(); err != nil {
			return fmt.Errorf("error while fetching deploymentId and sealId from application manifest: %v", err)
		}
		payloads, err := preparePayloads(images, imageMetaDatas)
		if err != nil {
			return fmt.Errorf("error while preparing payload for the job: %v", err)
		}

		if len(images) > 0 {
			presyncJobManifest, err := createPresyncPolicyJobManifest(payloads)
			if err != nil {
				return err
			}
			postSyncJobManifest, err := createPostsyncPolicyJobManifest(payloads)
			if err != nil {
				return err
			}
			manifests = append(manifests, presyncJobManifest, postSyncJobManifest)
		}

		for _, manifest := range manifests {
			output, err := toYAML(manifest)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s---\n", output)
		}

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&releaseCheckUrl, "release-check-url", "u", "", "release check url")
	rootCmd.Flags().StringVarP(&servicenowCheckUrl, "servicenow-check-url", "s", "", "servicenow check url")
	rootCmd.Flags().StringVarP(&submitDeploymentUrl, "submit-deployment-url", "d", "", "submit deployment url")
	rootCmd.Flags().StringVarP(&organization, "organization-name", "n", "", "organization name")
	rootCmd.Flags().StringVarP(&token, "service-token", "t", "", "service token")
	rootCmd.Flags().StringVarP(&gitMessage, "git-last-commit-message", "c", "", "git last commit message")
	rootCmd.Flags().StringVarP(&gitBranch, "git-branch", "b", "", "git branch") 
	rootCmd.Flags().StringVarP(&imagePolicyJob, "image-policy-job", "i", "", "image policy job")
	rootCmd.Flags().StringVarP(&repoUrl, "repo-url", "", "", "repoUrl")
	rootCmd.Flags().StringVarP(&gitLastCommitId, "git-last-commitId", "", "", "git last commitId")
	rootCmd.Flags().StringVarP(&targetEnvironment, "target-environment", "", "", "target environment")
	rootCmd.Flags().StringVarP(&custom, "custom","","", "custom variable for debugging")
	rootCmd.Flags().StringVarP(&argocdAppName, "argocd-app-name","","", "argocd application on which the plugin is applied")
}

func main() {
	Execute()
}

func preparePayloads(images []string, imageMetaData []ImageMetaData) ([]string, error) {
	payloads := make([]string, 0)
	for i, image := range images {
		payload, err := preparePayload(image, imageMetaData[i])
		if err != nil {
			return nil, err
		}
		payloads = append(payloads, payload)
	}
	return payloads, nil
}

func preparePayload(image string, imageMetaData ImageMetaData) (string, error) {
	imageName, imageTag := splitImageName(image)
	jobPayload := JobPayload{
		OrganizationName:				strings.TrimSpace(organization),
		ArtifactName:					imageName,
		ArtifactTag:					imageTag,
		ArtifactId					: imageMetaData.ArtifactId,
		ArtifactCreateDate			: imageMetaData.ArtifactCreateDate,
		JetId						: imageMetaData.JetId,
		SealId						: imageMetaData.SealId,
		DeploymentId				: deploymentId,
		ProjectName					: imageMetaData.ProjectName,
		artifactLocation			: imageMetaData.artifactLocation,
	}
	payloadBytea, err := json.Marshal(jobPayload)
	if err != nil {
		return "", err
	}
	return string(payloadBytea), nil
}