package nexus

import (
	"context"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gopkg.in/resty.v1"
)

type ClientTestSuite struct {
	suite.Suite
	cl *Client
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (s *ClientTestSuite) SetupTest() {
	s.cl = &Client{
		resty: resty.New(),
	}

	httpmock.Reset()
	httpmock.ActivateNonDefault(s.cl.resty.GetClient())
}

func (s *ClientTestSuite) TestClient_CreateUser() {
	t := s.T()

	httpmock.RegisterResponder("POST", "/security/users", httpmock.NewStringResponder(200, ""))
	err := s.cl.CreateUser(context.Background(), &User{
		Email: "mktest@example.com",
	})
	assert.NoError(t, err)

	httpmock.RegisterResponder("POST", "/security/users",
		httpmock.NewStringResponder(400, "bad request"))

	err = s.cl.CreateUser(context.Background(), &User{})
	assert.Error(t, err)
	assert.True(t, IsHTTPErrorCode(err, 400))
}

func (s *ClientTestSuite) TestClient_UpdateUser() {
	t := s.T()

	httpmock.RegisterResponder("PUT", "/security/users/id1", httpmock.NewStringResponder(200, ""))
	err := s.cl.UpdateUser(context.Background(), &User{Email: "mktest1@example.com", ID: "id1"})
	assert.NoError(t, err)

	httpmock.RegisterResponder("PUT", "/security/users/id1",
		httpmock.NewStringResponder(400, "bad request"))
	err = s.cl.UpdateUser(context.Background(), &User{Email: "mktest1@example.com", ID: "id1"})
	assert.Error(t, err)
	assert.True(t, IsHTTPErrorCode(err, 400))
}

func (s *ClientTestSuite) TestClient_DeleteUser() {
	t := s.T()

	httpmock.RegisterResponder("DELETE", "/security/users/id1",
		httpmock.NewStringResponder(200, ""))
	err := s.cl.DeleteUser(context.Background(), "id1")
	assert.NoError(t, err)

	httpmock.RegisterResponder("DELETE", "/security/users/id2",
		httpmock.NewStringResponder(500, ""))
	err = s.cl.DeleteUser(context.Background(), "id2")
	assert.Error(t, err)
	assert.True(t, IsHTTPErrorCode(err, 500))
}

func (s *ClientTestSuite) TestClient_GetUsers() {
	t := s.T()

	httpmock.RegisterResponder("GET", "/security/users", httpmock.NewJsonResponderOrPanic(200, []User{
		{
			Email: "mk@example.com",
		},
	}))
	users, err := s.cl.GetUsers(context.Background())

	assert.NoError(t, err)
	assert.True(t, len(users) > 0)
	assert.Equal(t, users[0].Email, "mk@example.com")

	httpmock.RegisterResponder("GET", "/security/users", httpmock.NewStringResponder(500, ""))
	_, err = s.cl.GetUsers(context.Background())
	assert.Error(t, err)
	assert.True(t, IsHTTPErrorCode(err, 500))
}

func (s *ClientTestSuite) TestClient_GetUser() {
	t := s.T()
	email := "mk@gmail.com"

	httpmock.RegisterResponder("GET", "/security/users", httpmock.NewJsonResponderOrPanic(200, []User{
		{
			Email: email,
		},
	}))

	usr, err := s.cl.GetUser(context.Background(), email)
	assert.NoError(t, err)
	assert.Equal(t, usr.Email, email)

	usr, err = s.cl.GetUser(context.Background(), "wrong_email")
	assert.Error(t, err)
	assert.True(t, IsErrNotFound(err))

	httpmock.RegisterResponder("GET", "/security/users", httpmock.NewStringResponder(500, ""))
	_, err = s.cl.GetUser(context.Background(), "e")
	assert.Error(t, err)
	assert.True(t, IsHTTPErrorCode(err, 500))
}
