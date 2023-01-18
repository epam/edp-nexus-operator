package nexus

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/resty.v1"

	nexusClientHelper "github.com/epam/edp-nexus-operator/v2/pkg/client/helper"
	"github.com/epam/edp-nexus-operator/v2/pkg/helper"
)

const (
	crAppJson     = "application/json"
	magicNumber10 = 10
)

type Client struct {
	resty *resty.Client
}

// Init performs initialization of Nexus connection.
func Init(url, user, password string) *Client {
	return &Client{
		resty: resty.SetHostURL(url).SetBasicAuth(user, password).SetHeaders(map[string]string{
			"Content-Type": crAppJson,
			"Accept":       crAppJson,
		}).SetDisableWarn(true),
	}
}

// IsNexusRestApiReady check if nexus rest api is ready.
func (nc Client) IsNexusRestApiReady() (nexusIsReady bool, statusCode int, err error) {
	nexusIsReady = true

	resp, err := nc.resty.
		SetRedirectPolicy(resty.FlexibleRedirectPolicy(magicNumber10)).
		R().
		Get("/status")
	if err != nil {
		helper.LogIfError(err)

		return nexusIsReady, resp.StatusCode(), fmt.Errorf("failed to check if nexus rest api is ready: %w", err)
	}

	if resp.IsError() {
		nexusIsReady = false
	}

	return nexusIsReady, resp.StatusCode(), nil
}

// CheckScriptExist checks if script is already uploaded.
func (nc Client) CheckScriptExist(scriptName string) (bool, error) {
	resp, err := nc.resty.R().
		SetHeader("accept", crAppJson).
		Get("/script")
	if err != nil || resp.IsError() {
		err = fmt.Errorf("failed to check if script - %s, status - %s: %w", scriptName, resp.Status(), err)
		helper.LogIfError(err)

		return false, err
	}

	var scriptsList []map[string]string

	if err = json.Unmarshal(resp.Body(), &scriptsList); err != nil {
		return false, fmt.Errorf("failed to unmarshal - %v: %w", resp.Body(), err)
	}

	for _, script := range scriptsList {
		if script["name"] == scriptName {
			return true, nil
		}
	}

	return false, nil
}

// UploadScript uploads script to Nexus.
func (nc Client) UploadScript(scriptName, scriptType, scriptContent string) error {
	formattedContent := nexusClientHelper.FormateNexusScript(scriptContent)

	resp, err := nc.resty.R().
		SetBody(`{"name":"` + scriptName + `", "type":"` + scriptType + `", "content": "` + formattedContent + `"}`).
		SetHeaders(map[string]string{"accept": crAppJson, "Content-type": crAppJson}).
		Post("/script")
	if err != nil || resp.IsError() {
		err = fmt.Errorf("failed to upload script - %s, status - %s: %w", scriptName, resp.Status(), err)
		helper.LogIfError(err)

		return err
	}

	return nil
}

// AreDefaultScriptsDeclared checks if default scripts are already declared in Nexus.
func (nc Client) AreDefaultScriptsDeclared(listOfScripts map[string]string) (bool, error) {
	defaultScriptsAreDeclared := true

	for scriptFullName := range listOfScripts {
		scriptName := strings.Split(scriptFullName, ".")[0]

		scriptExist, err := nc.CheckScriptExist(scriptName)
		if err != nil {
			return false, err
		}

		if !scriptExist {
			defaultScriptsAreDeclared = false
		}
	}

	return defaultScriptsAreDeclared, nil
}

// DeclareDefaultScripts declares default scripts in Nexus.
func (nc Client) DeclareDefaultScripts(listOfScripts map[string]string) error {
	for scriptFullName, scriptContent := range listOfScripts {
		scriptName := strings.Split(scriptFullName, ".")[0]
		scriptExtension := strings.Split(scriptFullName, ".")[1]

		scriptExist, err := nc.CheckScriptExist(scriptName)
		if err != nil {
			return err
		}

		if !scriptExist {
			if uploadErr := nc.UploadScript(scriptName, scriptExtension, scriptContent); uploadErr != nil {
				return uploadErr
			}
		}
	}

	return nil
}

// CheckTaskExist checks if task is already uploaded.
func (nc Client) CheckTaskExist(taskName string) (bool, error) {
	resp, err := nc.resty.R().
		SetHeader("accept", crAppJson).
		Get("/tasks")
	if err != nil || resp.IsError() {
		err = fmt.Errorf("failed to check if task - %s exists, status - %s: %w", taskName, resp.Status(), err)
		helper.LogIfError(err)

		return false, err
	}

	var tasksList map[string][]map[string]interface{}

	if err = json.Unmarshal(resp.Body(), &tasksList); err != nil {
		return false, fmt.Errorf("failed to unmarshal - %v: %w", resp.Body(), err)
	}

	for _, task := range tasksList["items"] {
		if task["name"] == taskName {
			return true, nil
		}
	}

	return false, nil
}

// RunScript runs script in Nexus.
func (nc Client) RunScript(scriptName string, parameters map[string]interface{}) ([]byte, error) {
	body, err := json.Marshal(parameters)
	if err != nil {
		err = fmt.Errorf("failed to marshal parameters - %s from script - %s: %w", parameters, scriptName, err)
		helper.LogIfError(err)

		return nil, err
	}

	resp, err := nc.resty.R().
		SetBody(body).
		SetHeader("Content-type", "text/plain").
		Post(fmt.Sprintf("/script/%v/run", scriptName))
	if err != nil || resp.IsError() {
		return nil, fmt.Errorf("failed to run script - %s, status - %s: %w", scriptName, resp.Status(), err)
	}

	return resp.Body(), nil
}

// CheckRoleExist checks if role is already exist.
func (nc Client) CheckRoleExist(roleName interface{}) (bool, error) {
	resp, err := nc.RunScript("get-role", map[string]interface{}{"id": roleName})
	if err != nil {
		return false, err
	}

	var parsedResponse map[string]string

	if err = json.Unmarshal(resp, &parsedResponse); err != nil {
		return false, fmt.Errorf("failed to unmarshal - %s: %w", string(resp), err)
	}

	var parsedResult map[string]interface{}

	if err = json.Unmarshal([]byte(parsedResponse["result"]), &parsedResult); err != nil {
		return false, fmt.Errorf("failed to unmarshal - %s: %w", parsedResponse["result"], err)
	}

	if parsedResult["roleId"] == roleName {
		return true, nil
	}

	return false, nil
}

// CheckRepositoryExist checks if repository name is present in Nexus repository list.
func (nc Client) CheckRepositoryExist(repositoryName string) (bool, error) {
	serverRepoList, err := nc.GetRepositoryList()
	if err != nil {
		return false, fmt.Errorf("failed to get repository list from nexus: %w", err)
	}

	for _, repository := range serverRepoList {
		if repository["name"] == repositoryName {
			return true, nil
		}
	}

	return false, nil
}

// GetRepositoryList takes list of repositories from Nexus server.
func (nc Client) GetRepositoryList() ([]map[string]interface{}, error) {
	var out []map[string]interface{}

	resp, err := nc.resty.R().Get("/repositories")
	if err != nil {
		return nil, fmt.Errorf("failed to get repository list from nexus: %w", err)
	}

	err = json.Unmarshal(resp.Body(), &out)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal body - %v: %w", resp.Body(), err)
	}

	return out, nil
}
