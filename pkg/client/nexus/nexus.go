package nexus

import (
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/resty.v1"
	"io/ioutil"

	"nexus-operator/pkg/apis/edp/v1alpha1"
	nexusClientHelper "nexus-operator/pkg/client/helper"
	"nexus-operator/pkg/helper"
	"strings"
	"time"
)

type NexusClient struct {
	instance *v1alpha1.Nexus
	resty    resty.Client
	ApiUrl   string
}

// InitNewRestClient performs initialization of Nexus connection
func (nc *NexusClient) InitNewRestClient(instance *v1alpha1.Nexus, url string, user string, password string) error {
	nc.resty = *resty.SetHostURL(url).SetBasicAuth(user, password)
	nc.instance = instance
	nc.ApiUrl = url
	return nil
}

// WaitForStatusIsUp waits for Nexus to be up
func (nc NexusClient) WaitForStatusIsUp(retryCount int, timeout time.Duration) error {
	resp, err := nc.resty.
		SetRetryCount(retryCount).
		SetRetryWaitTime(timeout * time.Second).
		AddRetryCondition(
			func(response *resty.Response) (bool, error) {
				if response.IsError() || !response.IsSuccess() {
					return response.IsError(), nil
				}
				return false, nil
			},
		).
		R().
		Get("/status")
	if err != nil || resp.IsError() {
		return helper.LogErrorAndReturn(errors.New(fmt.Sprintf("Checking Nexus component %v/%v status failed. Err - %v. Response - %s", nc.instance.Namespace, nc.instance.Name, err, resp.Status())))
	}
	return nil
}

// CheckScriptExist checks if script is already uploaded
func (nc NexusClient) CheckScriptExist(scriptName string) (bool, error) {
	resp, err := nc.resty.R().
		SetHeader("accept", "application/json").
		Get("/script")
	if err != nil || resp.IsError() {
		return false, helper.LogErrorAndReturn(errors.New(fmt.Sprintf("Checking if script %v exist failed. Err - %v. Response - %s", scriptName, err, resp.Status())))
	}

	var scriptsList []map[string]string
	err = json.Unmarshal(resp.Body(), &scriptsList)
	for _, script := range scriptsList {
		if script["name"] == scriptName {
			return true, nil
		}
	}
	return false, nil
}

// UploadScript uploads script to Nexus
func (nc NexusClient) UploadScript(scriptName string, scriptType string, scriptPath string) error {
	scriptContent, err := ioutil.ReadFile(scriptPath)
	formattedContent := nexusClientHelper.FormateNexusScript(string(scriptContent))
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
func (nc NexusClient) AreDefaultScriptsDeclared(scriptsPath string) (bool, error) {
	defaultScriptsAreDeclared := true
	files, err := ioutil.ReadDir(scriptsPath)
	if err != nil {
		return false, helper.LogErrorAndReturn(errors.New(fmt.Sprintf("[ERROR] Couldn't read directory with scripts for %v/%v. Err - %v.", nc.instance.Namespace, nc.instance.Name, err)))
	}

	for _, f := range files {
		scriptName := strings.Split(f.Name(), ".")[0]
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
func (nc NexusClient) DeclareDefaultScripts(scriptsPath string) error {
	files, err := ioutil.ReadDir(scriptsPath)
	if err != nil {
		return helper.LogErrorAndReturn(errors.New(fmt.Sprintf("[ERROR] Couldn't read directory with scripts for %v/%v. Err - %v.", nc.instance.Namespace, nc.instance.Name, err)))
	}

	for _, f := range files {
		scriptName := strings.Split(f.Name(), ".")[0]
		scriptExtension := strings.Split(f.Name(), ".")[1]
		scriptExist, err := nc.CheckScriptExist(scriptName)
		if err != nil {
			return err
		}
		if !scriptExist {
			nc.UploadScript(scriptName, scriptExtension, fmt.Sprintf("%v/%v", scriptsPath, f.Name()))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// CheckScriptExist checks if task is already uploaded
func (nc NexusClient) CheckTaskExist(taskName string) (bool, error) {
	resp, err := nc.resty.R().
		SetHeader("accept", "application/json").
		Get("/tasks")
	if err != nil || resp.IsError() {
		return false, helper.LogErrorAndReturn(errors.New(fmt.Sprintf("Checking if task %v exist failed. Err - %v. Response - %s", taskName, err, resp.Status())))
	}

	var tasksList map[string][]map[string]interface{}
	err = json.Unmarshal(resp.Body(), &tasksList)
	for _, task := range tasksList["items"] {
		if task["name"] == taskName {
			return true, nil
		}
	}
	return false, nil
}

// RunScript runs script in Nexus
func (nc NexusClient) RunScript(scriptName string, parameters map[string]interface{}) error {
	body, err := json.Marshal(parameters)
	if err != nil {
		return helper.LogErrorAndReturn(errors.New(fmt.Sprintf("Creating task %v failed. Err - %v. Response - %s", parameters["name"], err)))
	}
	resp, err := nc.resty.R().
		SetBody(body).
		SetHeader("Content-type", "text/plain").
		Post(fmt.Sprintf("/script/%v/run", scriptName))
	if err != nil || resp.IsError() {
		return helper.LogErrorAndReturn(errors.New(fmt.Sprintf("Creating task %v failed. Err - %v. Response - %s", parameters["name"], err, resp.Status())))
	}
	return nil
}

// CreateTask creates task in Nexus
func (nc NexusClient) CreateTask(parameters map[string]interface{}) error {
	err := nc.RunScript("create-task", parameters)
	if err != nil {
		return err
	}
	return nil
}
