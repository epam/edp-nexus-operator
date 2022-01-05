package nexus

import (
	"context"
	"testing"

	"github.com/jarcoal/httpmock"
	"gopkg.in/resty.v1"
)

func initClient() *Client {
	cl := Client{
		resty: *resty.New(),
	}

	httpmock.ActivateNonDefault(cl.resty.GetClient())

	return &cl
}

func TestClient_CreateUser(t *testing.T) {
	cl := initClient()

	httpmock.RegisterResponder("POST", "/security/users", httpmock.NewStringResponder(200, ""))
	if err := cl.CreateUser(context.Background(), &User{
		Email: "mktest@example.com",
	}); err != nil {
		t.Fatal(err)
	}

	httpmock.RegisterResponder("POST", "/security/users",
		httpmock.NewStringResponder(400, "bad request"))

	err := cl.CreateUser(context.Background(), &User{})
	if err == nil || !IsHTTPErrorCode(err, 400) {
		t.Fatalf("wrong error returned: %+v", err)
	}
}

func TestClient_UpdateUser(t *testing.T) {
	cl := initClient()

	httpmock.RegisterResponder("PUT", "/security/users/id1", httpmock.NewStringResponder(200, ""))
	if err := cl.UpdateUser(context.Background(), &User{Email: "mktest1@example.com", ID: "id1"}); err != nil {
		t.Fatal(err)
	}

	httpmock.RegisterResponder("PUT", "/security/users/id1",
		httpmock.NewStringResponder(400, "bad request"))
	err := cl.UpdateUser(context.Background(), &User{Email: "mktest1@example.com", ID: "id1"})
	if err == nil || !IsHTTPErrorCode(err, 400) {
		t.Fatalf("wrong error returned: %+v", err)
	}
}

func TestClient_DeleteUser(t *testing.T) {
	cl := initClient()

	httpmock.RegisterResponder("DELETE", "/security/users/id1",
		httpmock.NewStringResponder(200, ""))
	if err := cl.DeleteUser(context.Background(), "id1"); err != nil {
		t.Fatal(err)
	}

	httpmock.RegisterResponder("DELETE", "/security/users/id2",
		httpmock.NewStringResponder(500, ""))
	err := cl.DeleteUser(context.Background(), "id2")

	if err == nil || !IsHTTPErrorCode(err, 500) {
		t.Fatalf("wrong error returned: %+v", err)
	}
}
