package nexus

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/resty.v1"

	nexusClientHelper "github.com/epam/edp-nexus-operator/v2/pkg/client/helper"
	"github.com/epam/edp-nexus-operator/v2/pkg/helper"
)

type Client struct {
	resty *resty.Client
}

// Init performs initialization of Nexus connection
func Init(url string, user string, password string) *Client {
	return &Client{
		resty: resty.SetHostURL(url).SetBasicAuth(user, password).SetHeaders(map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
		}).SetDisableWarn(true),
	}
}

// WaitForStatusIsUp waits for Nexus to be up
func (nc Client) IsNexusRestApiReady() (bool, int, error) {
	nexusIsReady := true
	resp, err := nc.resty.
		SetRedirectPolicy(resty.FlexibleRedirectPolicy(10)).
		R().
		Get("/status")
	if err != nil {
		return nexusIsReady, resp.StatusCode(), helper.LogErrorAndReturn(err)
	} else if resp.IsError() {
		nexusIsReady = false
	}
	return nexusIsReady, resp.StatusCode(), nil
}

// CheckScriptExist checks if script is already uploaded
func (nc Client) CheckScriptExist(scriptName string) (bool, error) {
	resp, err := nc.resty.R().
		SetHeader("accept", "application/json").
		Get("/script")
	if err != nil || resp.IsError() {
		return false, helper.LogErrorAndReturn(errors.New(fmt.Sprintf("Checking if script %v exist failed. Err - %v. Response - %s", scriptName, err, resp.Status())))
	}

	var scriptsList []map[string]string
	err = json.Unmarshal(resp.Body(), &scriptsList)
	if err != nil {
		return false, errors.Wrapf(err, "cant umarshal %v", resp.Body())
	}
	for _, script := range scriptsList {
		if script["name"] == scriptName {
			return true, nil
		}
	}
	return false, nil
}

// UploadScript uploads script to Nexus
func (nc Client) UploadScript(scriptName string, scriptType string, scriptContent string) error {
	formattedContent := nexusClientHelper.FormateNexusScript(scriptContent)
	resp, err := nc.resty.R().
		SetBody(`{"name":"` + scriptName + `", "type":"` + scriptType + `", "content": "` + formattedContent + `"}`).
		SetHeaders(map[string]string{"accept": "application/json", "Content-type": "application/json"}).
		Post("/script")
	if err != nil || resp.IsError() {
		return helper.LogErrorAndReturn(errors.New(fmt.Sprintf("Uploading script %v failed. Err - %v. Response - %s", scriptName, err, resp.Status())))
	}
	return nil
}

// AreDefaultScriptsDeclared checks if default scripts are already declared in Nexus
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

// DeclareDefaultScripts declares default scripts in Nexus
func (nc Client) DeclareDefaultScripts(listOfScripts map[string]string) error {
	for scriptFullName, scriptContent := range listOfScripts {
		scriptName := strings.Split(scriptFullName, ".")[0]
		scriptExtension := strings.Split(scriptFullName, ".")[1]
		scriptExist, err := nc.CheckScriptExist(scriptName)
		if err != nil {
			return err
		}
		if !scriptExist {
			err := nc.UploadScript(scriptName, scriptExtension, scriptContent)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// CheckTaskExist checks if task is already uploaded
func (nc Client) CheckTaskExist(taskName string) (bool, error) {
	resp, err := nc.resty.R().
		SetHeader("accept", "application/json").
		Get("/tasks")
	if err != nil || resp.IsError() {
		return false, helper.LogErrorAndReturn(errors.New(fmt.Sprintf("Checking if task %v exist failed. Err - %v. Response - %s", taskName, err, resp.Status())))
	}

	var tasksList map[string][]map[string]interface{}
	err = json.Unmarshal(resp.Body(), &tasksList)
	if err != nil {
		return false, errors.Wrapf(err, "cant unmarshal %v", resp.Body())
	}
	for _, task := range tasksList["items"] {
		if task["name"] == taskName {
			return true, nil
		}
	}
	return false, nil
}

// RunScript runs script in Nexus
func (nc Client) RunScript(scriptName string, parameters map[string]interface{}) ([]byte, error) {
	body, err := json.Marshal(parameters)
	if err != nil {
		return nil, helper.LogErrorAndReturn(errors.New(fmt.Sprintf("Couldn't marshmal parameters %v from script %v. Err - %v", parameters, scriptName, err)))
	}
	resp, err := nc.resty.R().
		SetBody(body).
		SetHeader("Content-type", "text/plain").
		Post(fmt.Sprintf("/script/%v/run", scriptName))
	if err != nil || resp.IsError() {
		return nil, errors.Wrapf(err, fmt.Sprintf("Running script %v failed. Response - %s", scriptName, resp.Status()))
	}
	return resp.Body(), nil
}

// CheckRoleExist checks if role is already exist
func (nc Client) CheckRoleExist(roleName interface{}) (bool, error) {
	resp, err := nc.RunScript("get-role", map[string]interface{}{"id": roleName})
	if err != nil {
		return false, err
	}
	var parsedResponse map[string]string
	err = json.Unmarshal(resp, &parsedResponse)
	if err != nil {
		return false, errors.Wrapf(err, "Unable to unmarshal %v. Err - %v.", string(resp), err)
	}
	var parsedResult map[string]interface{}
	err = json.Unmarshal([]byte(parsedResponse["result"]), &parsedResult)
	if err != nil {
		return false, errors.Wrapf(err, "Unable to unmarshal %v. Err - %v.", parsedResponse["result"], err)
	}
	if parsedResult["roleId"] == roleName {
		return true, nil
	}
	return false, nil
}

// CheckRepositoryExist checks if repository name is present in Nexus repository list
func (nc Client) CheckRepositoryExist(repositoryName string) (bool, error) {
	serverRepoList, err := nc.GetRepositoryList()
	if err != nil {
		return false, errors.Wrap(err, "Failed to get repository list from Nexus!")
	}

	for _, repository := range serverRepoList {
		if repository["name"] == repositoryName {
			return true, nil
		}
	}
	return false, nil
}

// GetRepositoryList takes list of repositories from Nexus server
func (nc Client) GetRepositoryList() ([]map[string]interface{}, error) {
	var out []map[string]interface{}

	resp, err := nc.resty.R().Get("/repositories")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get repository list from Nexus!")
	}

	err = json.Unmarshal(resp.Body(), &out)
	if err != nil {
		return nil, errors.Wrapf(err, "Cant unmarshal %v", resp.Body())
	}
	return out, nil
}
