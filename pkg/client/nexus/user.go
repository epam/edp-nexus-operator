package nexus

import (
	"context"

	"github.com/pkg/errors"
)

type User struct {
	ID        string   `json:"userId"`
	FirstName string   `json:"firstName"`
	LastName  string   `json:"lastName"`
	Email     string   `json:"emailAddress"`
	Password  string   `json:"password"`
	Status    string   `json:"status"`
	Roles     []string `json:"roles"`
	Source    string   `json:"source,omitempty"`
}

func (nc *Client) CreateUser(ctx context.Context, u *User) error {
	rsp, err := nc.requestWithContext(ctx).
		SetBody(u).
		SetResult(u).
		Post("/security/users")

	if err := checkRestyResponse(rsp, err); err != nil {
		return errors.Wrap(err, "unable to create user")
	}

	return nil
}

func (nc *Client) UpdateUser(ctx context.Context, u *User) error {
	rsp, err := nc.requestWithContext(ctx).
		SetBody(u).
		SetPathParams(map[string]string{
			"userID": u.ID,
		}).
		Put("/security/users/{userID}")

	if err := checkRestyResponse(rsp, err); err != nil {
		return errors.Wrap(err, "unable to update user")
	}

	return nil
}

func (nc *Client) DeleteUser(ctx context.Context, ID string) error {
	rsp, err := nc.requestWithContext(ctx).SetPathParams(map[string]string{
		"userID": ID,
	}).Delete("/security/users/{userID}")

	if err := checkRestyResponse(rsp, err); err != nil {
		return errors.Wrap(err, "unable to delete user")
	}

	return nil
}
