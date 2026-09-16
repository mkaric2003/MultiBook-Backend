package application

import (
	"context"
	"errors"
	"testing"

	"github.com/mkaric2003/multibook-backend/internal/modules/users/domain"
)

type repositoryStub struct {
	provisioned, roleSet, profileUpdated bool
}

func (r *repositoryStub) ProvisionFromAuthentication(context.Context, AuthenticatedUser) error {
	r.provisioned = true
	return nil
}

func (r *repositoryStub) SetRole(context.Context, string, domain.UserRole) (domain.User, error) {
	r.roleSet = true
	return domain.User{}, nil
}

func (r *repositoryStub) UpdateProfile(context.Context, string, domain.UpdateProfileInput) (domain.User, error) {
	r.profileUpdated = true
	return domain.User{}, nil
}

type queriesStub struct{ fetched bool }

func (q *queriesStub) GetByID(context.Context, string) (domain.User, error) {
	q.fetched = true
	return domain.User{}, nil
}

func TestSetRoleRejectsRolesThatUsersCannotSelfAssign(t *testing.T) {
	service := NewService(&repositoryStub{}, &queriesStub{})
	if _, err := service.SetRole(context.Background(), "firebase-uid", domain.UserRoleAdmin); err == nil {
		t.Fatal("SetRole() error = nil, want rejection for admin self-assignment")
	}
}

func TestUpdateProfileRejectsUnsupportedCurrency(t *testing.T) {
	service := NewService(&repositoryStub{}, &queriesStub{})
	currency := "XYZ"
	if _, err := service.UpdateProfile(context.Background(), "firebase-uid", domain.UpdateProfileInput{BusinessCurrency: &currency}); err == nil {
		t.Fatal("UpdateProfile() error = nil, want unsupported currency rejection")
	}
}

func TestUpdateProfileRejectsNonDateValue(t *testing.T) {
	service := NewService(&repositoryStub{}, &queriesStub{})
	dateOfBirth := "20-07-2003"

	if _, err := service.UpdateProfile(context.Background(), "firebase-uid", domain.UpdateProfileInput{DateOfBirth: &dateOfBirth}); !errors.Is(err, ErrValidation) {
		t.Fatalf("UpdateProfile() error = %v, want validation error", err)
	}
}

func TestCurrentUserRequiresAuthenticatedIdentity(t *testing.T) {
	service := NewService(&repositoryStub{}, &queriesStub{})

	if _, err := service.CurrentUser(context.Background(), AuthenticatedUser{}); err == nil {
		t.Fatal("CurrentUser() error = nil, want missing authenticated user rejection")
	}
}

func TestCurrentUserUsesCommandAndQueryPorts(t *testing.T) {
	t.Parallel()
	repository, queries := &repositoryStub{}, &queriesStub{}
	if _, err := NewService(repository, queries).CurrentUser(context.Background(), AuthenticatedUser{ID: "firebase-uid"}); err != nil {
		t.Fatal(err)
	}
	if !repository.provisioned || !queries.fetched {
		t.Fatalf("current user should provision then query: repository=%+v queries=%+v", repository, queries)
	}
}

func TestUserMutationsUseCommandRepository(t *testing.T) {
	t.Parallel()
	repository := &repositoryStub{}
	service := NewService(repository, &queriesStub{})
	if _, err := service.SetRole(context.Background(), "firebase-uid", domain.UserRoleProvider); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateProfile(context.Background(), "firebase-uid", domain.UpdateProfileInput{}); err != nil {
		t.Fatal(err)
	}
	if !repository.roleSet || !repository.profileUpdated {
		t.Fatalf("mutations should use command repository: %+v", repository)
	}
}
