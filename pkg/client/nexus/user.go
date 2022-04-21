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

func (nc *Client) GetUsers(ctx context.Context) ([]User, error) {
	var ret []User
	rsp, err := nc.requestWithContext(ctx).SetResult(&ret).Get("/security/users")
	if err := checkRestyResponse(rsp, err); err != nil {
		return nil, errors.Wrap(err, "unable to get users")
	}

	return ret, nil
}

func (nc *Client) GetUser(ctx context.Context, email string) (*User, error) {
	users, err := nc.GetUsers(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get users")
	}

	for i, usr := range users {
		if usr.Email == email || usr.ID == email {
			return &users[i], nil
		}
	}

	return nil, ErrNotFound("user not found")
}
