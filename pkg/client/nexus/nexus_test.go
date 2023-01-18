package nexus

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"gopkg.in/resty.v1"
)

const (
	name          = "name"
	scriptType    = "type"
	scriptContent = "content"
	url           = "https://domain"
	user          = "user"
	password      = "pwd"
)

func CreateMockResty() *resty.Client {
	restyClient := resty.SetHostURL(url).SetBasicAuth(user, password).SetHeaders(map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	})

	httpmock.DeactivateAndReset()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	return restyClient
}

func TestInit(t *testing.T) {
	client := Init(url, user, password)
	assert.NotNil(t, client)
}

func TestClient_IsNexusRestApiReady_GetErr(t *testing.T) {
	restClient := CreateMockResty()
	client := Client{resty: restClient}
	_, _, err := client.IsNexusRestApiReady()
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "no responder found"))
}

func TestClient_IsNexusRestApiReady_StatusNotFound(t *testing.T) {
	restClient := CreateMockResty()

	httpmock.RegisterResponder(http.MethodGet, "https://domain/status", httpmock.NewStringResponder(http.StatusNotFound, ""))

	client := Client{resty: restClient}

	ok, statusCode, err := client.IsNexusRestApiReady()
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, statusCode)
	assert.False(t, ok)
}

func TestClient_CheckScriptExist_GetErr(t *testing.T) {
	restClient := CreateMockResty()
	client := Client{resty: restClient}
	exist, err := client.CheckScriptExist(name)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check if script")
	assert.False(t, exist)
}

func TestClient_CheckScriptExist_UnmarshallErr(t *testing.T) {
	restClient := CreateMockResty()

	httpmock.RegisterResponder(http.MethodGet, "https://domain/script", httpmock.NewStringResponder(http.StatusOK, ""))

	client := Client{resty: restClient}
	exist, err := client.CheckScriptExist(name)

	assert.Contains(t, err.Error(), "failed to unmarshal")
	assert.False(t, exist)
}

func TestClient_CheckScriptExist_True(t *testing.T) {
	respData := []map[string]string{{name: name}}

	raw, err := json.Marshal(respData)
	if err != nil {
		t.Fatal(err)
	}

	restClient := CreateMockResty()

	httpmock.RegisterResponder(http.MethodGet, "https://domain/script", httpmock.NewBytesResponder(http.StatusOK, raw))

	client := Client{resty: restClient}
	exist, err := client.CheckScriptExist(name)
	assert.NoError(t, err)
	assert.True(t, exist)
}

func TestClient_CheckScriptExist_False(t *testing.T) {
	var respData []map[string]string

	raw, err := json.Marshal(respData)
	if err != nil {
		t.Fatal(err)
	}

	restClient := CreateMockResty()

	httpmock.RegisterResponder(http.MethodGet, "https://domain/script", httpmock.NewBytesResponder(http.StatusOK, raw))

	client := Client{resty: restClient}
	exist, err := client.CheckScriptExist(name)
	assert.NoError(t, err)
	assert.False(t, exist)
}

func TestClient_UploadScript_Err(t *testing.T) {
	restClient := CreateMockResty()
	client := Client{resty: restClient}

	err := client.UploadScript(name, scriptType, scriptContent)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to upload script")
}

func TestClient_UploadScript(t *testing.T) {
	restClient := CreateMockResty()

	httpmock.RegisterResponder(http.MethodPost, "https://domain/script", httpmock.NewStringResponder(http.StatusOK, ""))

	client := Client{resty: restClient}

	err := client.UploadScript(name, scriptType, scriptContent)
	assert.NoError(t, err)
}

func TestClient_AreDefaultScriptsDeclared_Err(t *testing.T) {
	scriptList := map[string]string{name: name}
	restClient := CreateMockResty()
	client := Client{resty: restClient}

	declared, err := client.AreDefaultScriptsDeclared(scriptList)
	assert.False(t, declared)
	assert.Error(t, err)
}

func TestClient_AreDefaultScriptsDeclared_False(t *testing.T) {
	scriptList := map[string]string{name: name}
	respData := []map[string]string{{}}

	raw, err := json.Marshal(respData)
	if err != nil {
		t.Fatal(err)
	}

	restClient := CreateMockResty()

	httpmock.RegisterResponder(http.MethodGet, "https://domain/script", httpmock.NewBytesResponder(http.StatusOK, raw))

	client := Client{resty: restClient}

	declared, err := client.AreDefaultScriptsDeclared(scriptList)
	assert.False(t, declared)
	assert.NoError(t, err)
}

func TestClient_AreDefaultScriptsDeclared_True(t *testing.T) {
	scriptList := map[string]string{name: name}
	respData := []map[string]string{{name: name}}

	raw, err := json.Marshal(respData)
	if err != nil {
		t.Fatal(err)
	}

	restClient := CreateMockResty()

	httpmock.RegisterResponder(http.MethodGet, "https://domain/script", httpmock.NewBytesResponder(http.StatusOK, raw))

	client := Client{resty: restClient}
	declared, err := client.AreDefaultScriptsDeclared(scriptList)
	assert.True(t, declared)
	assert.NoError(t, err)
}

func TestClient_DeclareDefaultScripts_CheckScriptExistErr(t *testing.T) {
	scriptList := map[string]string{name + "." + scriptType: name}
	restClient := CreateMockResty()
	client := Client{resty: restClient}
	err := client.DeclareDefaultScripts(scriptList)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check if script")
}

func TestClient_DeclareDefaultScripts_UploadScriptErr(t *testing.T) {
	scriptList := map[string]string{name + "." + scriptType: name}
	restClient := CreateMockResty()
	respData := []map[string]string{{}}

	raw, err := json.Marshal(respData)
	if err != nil {
		t.Fatal(err)
	}

	httpmock.RegisterResponder(http.MethodGet, "https://domain/script", httpmock.NewBytesResponder(http.StatusOK, raw))

	client := Client{resty: restClient}
	err = client.DeclareDefaultScripts(scriptList)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to upload script")
}

func TestClient_DeclareDefaultScripts(t *testing.T) {
	scriptList := map[string]string{name + "." + scriptType: name}
	restClient := CreateMockResty()
	respData := []map[string]string{{name: name}}

	raw, err := json.Marshal(respData)
	if err != nil {
		t.Fatal(err)
	}

	httpmock.RegisterResponder(http.MethodGet, "https://domain/script", httpmock.NewBytesResponder(http.StatusOK, raw))

	client := Client{resty: restClient}
	err = client.DeclareDefaultScripts(scriptList)
	assert.NoError(t, err)
}

func TestClient_CheckTaskExist_GetErr(t *testing.T) {
	restClient := CreateMockResty()
	client := Client{resty: restClient}
	exist, err := client.CheckTaskExist(name)
	assert.False(t, exist)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check if task")
}

func TestClient_CheckTaskExist_UnmarshalErr(t *testing.T) {
	restClient := CreateMockResty()

	httpmock.RegisterResponder(http.MethodGet, "https://domain/tasks", httpmock.NewStringResponder(http.StatusOK, ""))

	client := Client{resty: restClient}
	exist, err := client.CheckTaskExist(name)

	errJSON := &json.SyntaxError{}
	assert.ErrorAs(t, err, &errJSON)
	assert.False(t, exist)
}

func TestClient_CheckTaskExist_False(t *testing.T) {
	restClient := CreateMockResty()

	var respData map[string][]map[string]interface{}

	raw, err := json.Marshal(respData)
	if err != nil {
		t.Fatal(err)
	}

	httpmock.RegisterResponder(http.MethodGet, "https://domain/tasks", httpmock.NewBytesResponder(http.StatusOK, raw))

	client := Client{resty: restClient}
	exist, err := client.CheckTaskExist(name)

	assert.NoError(t, err)
	assert.False(t, exist)
}

func TestClient_RunScript_PostErr(t *testing.T) {
	params := map[string]interface{}{}
	restClient := CreateMockResty()
	client := Client{resty: restClient}
	_, err := client.RunScript(name, params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to run script")
}

func TestClient_RunScript(t *testing.T) {
	params := map[string]interface{}{}
	restClient := CreateMockResty()
	bytes := []byte(name)

	httpmock.RegisterResponder(http.MethodPost, "https://domain/script/name/run", httpmock.NewBytesResponder(http.StatusOK, bytes))

	client := Client{resty: restClient}
	body, err := client.RunScript(name, params)
	assert.NoError(t, err)
	assert.Equal(t, bytes, body)
}

func TestClient_CheckRoleExist_PostErr(t *testing.T) {
	restClient := CreateMockResty()
	client := Client{resty: restClient}
	exist, err := client.CheckRoleExist(name)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to run script")
	assert.False(t, exist)
}

func TestClient_CheckRoleExist_UnmarshalErr1(t *testing.T) {
	restClient := CreateMockResty()
	client := Client{resty: restClient}

	var raw []byte

	httpmock.RegisterResponder(http.MethodPost, "https://domain/script/get-role/run", httpmock.NewBytesResponder(http.StatusOK, raw))

	exist, err := client.CheckRoleExist(name)
	errJSON := &json.SyntaxError{}
	assert.ErrorAs(t, err, &errJSON)
	assert.False(t, exist)
}

func TestClient_CheckRoleExist_UnmarshalErr2(t *testing.T) {
	restClient := CreateMockResty()
	client := Client{resty: restClient}
	respData := map[string]string{}

	raw, err := json.Marshal(respData)
	if err != nil {
		t.Fatal(err)
	}

	httpmock.RegisterResponder(http.MethodPost, "https://domain/script/get-role/run", httpmock.NewBytesResponder(http.StatusOK, raw))

	exist, err := client.CheckRoleExist(name)
	errJSON := &json.SyntaxError{}
	assert.ErrorAs(t, err, &errJSON)
	assert.False(t, exist)
}

func TestClient_GetRepositoryList_GetErr(t *testing.T) {
	restClient := CreateMockResty()
	client := Client{resty: restClient}
	list, err := client.GetRepositoryList()
	assert.Error(t, err)
	assert.Empty(t, list)
	assert.Contains(t, err.Error(), "failed to get repository list from nexus")
}

func TestClient_GetRepositoryList_UnmarshalErr(t *testing.T) {
	restClient := CreateMockResty()

	var raw []byte

	httpmock.RegisterResponder(http.MethodGet, "https://domain/repositories", httpmock.NewBytesResponder(http.StatusOK, raw))

	client := Client{resty: restClient}
	list, err := client.GetRepositoryList()
	errJSON := &json.SyntaxError{}

	assert.ErrorAs(t, err, &errJSON)
	assert.Empty(t, list)
}

func TestClient_GetRepositoryList(t *testing.T) {
	restClient := CreateMockResty()
	out := []map[string]interface{}{{name: name}}

	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}

	httpmock.RegisterResponder(http.MethodGet, "https://domain/repositories", httpmock.NewBytesResponder(http.StatusOK, raw))

	client := Client{resty: restClient}
	list, err := client.GetRepositoryList()
	assert.NoError(t, err)
	assert.Equal(t, out, list)
}

func TestClient_CheckRepositoryExist_GetRepositoryListErr(t *testing.T) {
	restClient := CreateMockResty()
	client := Client{resty: restClient}
	exist, err := client.CheckRepositoryExist(name)
	assert.False(t, exist)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get repository list from nexus")
}

func TestClient_CheckRepositoryExist_True(t *testing.T) {
	restClient := CreateMockResty()
	client := Client{resty: restClient}

	out := []map[string]interface{}{{name: name}}

	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}

	httpmock.RegisterResponder(http.MethodGet, "https://domain/repositories", httpmock.NewBytesResponder(http.StatusOK, raw))

	exist, err := client.CheckRepositoryExist(name)
	assert.True(t, exist)
	assert.NoError(t, err)
}

func TestClient_CheckRepositoryExist_False(t *testing.T) {
	restClient := CreateMockResty()
	client := Client{resty: restClient}

	var out []map[string]interface{}

	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}

	httpmock.RegisterResponder(http.MethodGet, "https://domain/repositories", httpmock.NewBytesResponder(http.StatusOK, raw))

	exist, err := client.CheckRepositoryExist(name)
	assert.False(t, exist)
	assert.NoError(t, err)
}
