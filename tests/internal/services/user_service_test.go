package services_test

import (
	"context"
	"errors"
	"testing"

	"accesspath/internal/models"
	"accesspath/internal/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"accesspath/tests/internal/fakes"
)

func newUserSvc(t *testing.T, repo *fakes.RepoUser) services.UserService {
	t.Helper()
	return services.NewUserService(repo)
}

// --- GetByID ---------------------------------------------------------------

func TestUserService_GetByID_DelegatesToRepo(t *testing.T) {
	repo := &fakes.RepoUser{}
	repo.FindByIDFn = func(_ context.Context, id int64) (*models.User, error) {
		return &models.User{ID: id, Username: "juan"}, nil
	}
	svc := newUserSvc(t, repo)

	u, err := svc.GetByID(context.Background(), 7)

	require.NoError(t, err)
	assert.Equal(t, int64(7), u.ID)
	assert.Equal(t, "juan", u.Username)
}

func TestUserService_GetByID_RepoErrorPropagates(t *testing.T) {
	repo := &fakes.RepoUser{}
	repo.FindByIDFn = func(_ context.Context, _ int64) (*models.User, error) {
		return nil, errors.New("db down")
	}
	svc := newUserSvc(t, repo)

	u, err := svc.GetByID(context.Background(), 7)

	require.Error(t, err)
	assert.Nil(t, u)
}

// --- Register --------------------------------------------------------------

func TestUserService_Register_EmailTakenReturnsSentinel(t *testing.T) {
	repo := &fakes.RepoUser{}
	repo.FindByEmailFn = func(_ context.Context, _ string) (*models.UserWithPassword, error) {
		return &models.UserWithPassword{}, nil // ya existe
	}
	svc := newUserSvc(t, repo)

	u, err := svc.Register(context.Background(), models.CreateUserRequest{
		Email: "a@b.com", Password: "abcdef",
	})

	require.Error(t, err)
	assert.Nil(t, u)
	assert.ErrorIs(t, err, services.ErrEmailAlreadyUsed)
}

func TestUserService_Register_AutoGeneratesUsernameWhenEmpty(t *testing.T) {
	repo := &fakes.RepoUser{}
	repo.FindByEmailFn = func(_ context.Context, _ string) (*models.UserWithPassword, error) {
		return nil, errors.New("no rows")
	}
	var capturedReq models.CreateUserRequest
	repo.CreateFn = func(_ context.Context, req models.CreateUserRequest, _ string) (*models.User, error) {
		capturedReq = req
		return &models.User{ID: 1, Username: req.Username, Email: req.Email}, nil
	}
	svc := newUserSvc(t, repo)

	u, err := svc.Register(context.Background(), models.CreateUserRequest{
		Email: "juan@example.com", Password: "abcdef",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, u.Username)
	assert.NotEmpty(t, capturedReq.Username)
	assert.Contains(t, capturedReq.Username, "juan.")
}

func TestUserService_Register_RespectsClientUsername(t *testing.T) {
	repo := &fakes.RepoUser{}
	repo.FindByEmailFn = func(_ context.Context, _ string) (*models.UserWithPassword, error) {
		return nil, errors.New("no rows")
	}
	var capturedReq models.CreateUserRequest
	repo.CreateFn = func(_ context.Context, req models.CreateUserRequest, _ string) (*models.User, error) {
		capturedReq = req
		return &models.User{Username: req.Username}, nil
	}
	svc := newUserSvc(t, repo)

	_, err := svc.Register(context.Background(), models.CreateUserRequest{
		Username: "custom_name",
		Email:    "a@b.com",
		Password: "abcdef",
	})

	require.NoError(t, err)
	assert.Equal(t, "custom_name", capturedReq.Username,
		"si el cliente manda username NO debe regenerarse")
}

func TestUserService_Register_HashesPassword(t *testing.T) {
	repo := &fakes.RepoUser{}
	repo.FindByEmailFn = func(_ context.Context, _ string) (*models.UserWithPassword, error) {
		return nil, errors.New("no rows")
	}
	var capturedHash string
	repo.CreateFn = func(_ context.Context, _ models.CreateUserRequest, h string) (*models.User, error) {
		capturedHash = h
		return &models.User{}, nil
	}
	svc := newUserSvc(t, repo)

	_, err := svc.Register(context.Background(), models.CreateUserRequest{
		Email: "a@b.com", Password: "abcdef",
	})

	require.NoError(t, err)
	assert.NotEqual(t, "abcdef", capturedHash, "password debe estar hasheada, no en plano")
	assert.GreaterOrEqual(t, len(capturedHash), 50, "bcrypt genera hashes >= 50 chars")
}

func TestUserService_Register_HappyPath(t *testing.T) {
	repo := &fakes.RepoUser{}
	repo.FindByEmailFn = func(_ context.Context, _ string) (*models.UserWithPassword, error) {
		return nil, errors.New("no rows")
	}
	repo.CreateFn = func(_ context.Context, req models.CreateUserRequest, h string) (*models.User, error) {
		return &models.User{ID: 99, Username: req.Username, Email: req.Email}, nil
	}
	svc := newUserSvc(t, repo)

	u, err := svc.Register(context.Background(), models.CreateUserRequest{
		Email: "a@b.com", Password: "abcdef",
	})

	require.NoError(t, err)
	assert.Equal(t, int64(99), u.ID)
}

// --- Login -----------------------------------------------------------------

func TestUserService_Login_EmailNotFoundReturnsInvalidCredentials(t *testing.T) {
	repo := &fakes.RepoUser{}
	repo.FindByEmailFn = func(_ context.Context, _ string) (*models.UserWithPassword, error) {
		return nil, errors.New("no rows")
	}
	svc := newUserSvc(t, repo)

	u, err := svc.Login(context.Background(), models.LoginRequest{
		Email: "a@b.com", Password: "abcdef",
	})

	require.Error(t, err)
	assert.Nil(t, u)
	assert.ErrorIs(t, err, services.ErrInvalidCredentials)
}

func TestUserService_Login_WrongPasswordReturnsInvalidCredentials(t *testing.T) {
	repo := &fakes.RepoUser{}
	repo.FindByEmailFn = func(_ context.Context, _ string) (*models.UserWithPassword, error) {
		// bcrypt de "right" -> el test manda "wrong"
		return &models.UserWithPassword{
			User:         models.User{ID: 1},
			PasswordHash: "$2a$10$abcdefghijklmnopqrstuv.WXyZ01234567890123456789012345",
		}, nil
	}
	svc := newUserSvc(t, repo)

	u, err := svc.Login(context.Background(), models.LoginRequest{
		Email: "a@b.com", Password: "wrong",
	})

	require.Error(t, err)
	assert.Nil(t, u)
	assert.ErrorIs(t, err, services.ErrInvalidCredentials)
}

func TestUserService_Login_HappyPath(t *testing.T) {
	// Generamos un hash bcrypt real del password para que el match funcione.
	const password = "abcdef"
	hash := bcryptHash(t, password)

	repo := &fakes.RepoUser{}
	repo.FindByEmailFn = func(_ context.Context, email string) (*models.UserWithPassword, error) {
		return &models.UserWithPassword{
			User:         models.User{ID: 42, Email: email},
			PasswordHash: hash,
		}, nil
	}
	svc := newUserSvc(t, repo)

	u, err := svc.Login(context.Background(), models.LoginRequest{
		Email: "a@b.com", Password: password,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(42), u.ID)
	assert.Equal(t, "a@b.com", u.Email)
}
