package nexus

import (
	"context"
	"fmt"
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

	if err = checkRestyResponse(rsp, err); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
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

	if err = checkRestyResponse(rsp, err); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

func (nc *Client) DeleteUser(ctx context.Context, id string) error {
	rsp, err := nc.requestWithContext(ctx).SetPathParams(map[string]string{
		"userID": id,
	}).Delete("/security/users/{userID}")

	if err = checkRestyResponse(rsp, err); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

func (nc *Client) GetUsers(ctx context.Context) ([]User, error) {
	var ret []User

	rsp, err := nc.requestWithContext(ctx).SetResult(&ret).Get("/security/users")
	if err = checkRestyResponse(rsp, err); err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	return ret, nil
}

func (nc *Client) GetUser(ctx context.Context, email string) (*User, error) {
	users, err := nc.GetUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	for i := range users {
		if users[i].Email == email || users[i].ID == email {
			return &users[i], nil
		}
	}

	return nil, ErrNotFound
}
