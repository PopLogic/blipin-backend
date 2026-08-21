package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/PopLogic/blipin-backend/token"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store provides all functions to execute db queries and transactions.
type Store struct {
	*Queries
	db *pgxpool.Pool
}

var ErrInvalidBirthdate = errors.New("invalid birthdate")
var ErrInvalidGender = errors.New("invalid gender")

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{
		db:      db,
		Queries: New(db),
	}
}

func (store *Store) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}

	q := New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit(ctx)
}

type CreateUserWithPasswordCredentialParams struct {
	CreateUserParams
	CreatePasswordCredentialParams
}

type CreateUserWithPasswordCredentialResult struct {
	User               User
	PasswordCredential PasswordCredential
}

func (store *Store) CreateUserWithPasswordCredential(ctx context.Context, arg CreateUserWithPasswordCredentialParams) (CreateUserWithPasswordCredentialResult, error) {
	var result CreateUserWithPasswordCredentialResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error
		result.User, err = q.CreateUser(ctx, arg.CreateUserParams)
		if err != nil {
			return err
		}

		arg.CreatePasswordCredentialParams.UserID = result.User.ID
		result.PasswordCredential, err = q.CreatePasswordCredential(ctx, arg.CreatePasswordCredentialParams)
		if err != nil {
			return err
		}

		return nil
	})

	return result, err
}

type CreateUserWithUserIdentityParams struct {
	DisplayName   string       `json:"display_name"`
	AvatarUrl     string       `json:"avatar_url"`
	Provider      AuthProvider `json:"provider"`
	ProviderSub   string       `json:"provider_sub"`
	ProviderEmail string       `json:"provider_email"`
}

type CreateUserWithUserIdentityResult struct {
	User         User
	UserIdentity UserIdentity
}

func (store *Store) CreateUserWithUserIdentity(ctx context.Context, tokenMaker token.Maker, accessTokenDuration time.Duration, refreshTokenDuration time.Duration, arg CreateUserWithUserIdentityParams) (CreateUserWithUserIdentityResult, error) {
	var result CreateUserWithUserIdentityResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error
		result.User, err = q.CreateUser(ctx, CreateUserParams{
			DisplayName: pgtype.Text{
				String: arg.DisplayName,
				Valid:  true,
			},
			AvatarUrl: pgtype.Text{
				String: arg.AvatarUrl,
				Valid:  true,
			},
			Status: UserStatusActive,
		})
		if err != nil {
			return err
		}

		result.UserIdentity, err = q.CreateUserIdentity(ctx, CreateUserIdentityParams{
			UserID:      result.User.ID,
			Provider:    arg.Provider,
			ProviderSub: arg.ProviderSub,
			ProviderEmail: pgtype.Text{
				String: arg.ProviderEmail,
				Valid:  true,
			},
		})
		if err != nil {
			return err
		}
		return nil
	})

	return result, err
}

type CheckUserExistsByUserIdentityParams struct {
	Provider    string
	ProviderSub string
}

type CheckUserExistsByUserIdentityResult struct {
	UserIdentity UserIdentity
	User         User
}

func (store *Store) CheckUserExistsByUserIdentity(ctx context.Context, checkUserExistsByUserIdentityParams CheckUserExistsByUserIdentityParams) (*CheckUserExistsByUserIdentityResult, error) {
	authProvider := AuthProvider(checkUserExistsByUserIdentityParams.Provider)
	userIdentity, err := store.GetUserIdentityByProviderAndSub(ctx, GetUserIdentityByProviderAndSubParams{
		Provider:    authProvider,
		ProviderSub: checkUserExistsByUserIdentityParams.ProviderSub,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			log.Printf("User identity not found for provider: %s, provider_sub: %s", checkUserExistsByUserIdentityParams.Provider, checkUserExistsByUserIdentityParams.ProviderSub)
			return nil, nil
		}
		log.Printf("Error checking user identity for provider: %s, provider_sub: %s, error: %v", checkUserExistsByUserIdentityParams.Provider, checkUserExistsByUserIdentityParams.ProviderSub, err)
		return nil, err
	}

	if userIdentity.ID > 0 {
		user, err := store.GetUser(ctx, userIdentity.UserID)
		if err != nil {
			log.Printf("Error retrieving user for user_id: %d, error: %v", userIdentity.UserID, err)
			return nil, err
		}
		return &CheckUserExistsByUserIdentityResult{
			UserIdentity: userIdentity,
			User:         user,
		}, nil
	}

	return nil, nil
}

func (store *Store) CheckUserExistsByPasswordCredential(ctx context.Context, email string) (*User, error) {
	passwordCredential, err := store.GetPasswordCredentialByEmail(ctx, email)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	user, err := store.GetUser(ctx, passwordCredential.UserID)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func parseDate(dateStr string) (pgtype.Date, error) {
	trimmed := strings.TrimSpace(dateStr)
	if trimmed == "" {
		return pgtype.Date{Valid: false}, nil
	}

	if t, err := time.Parse("2006-01-02", trimmed); err == nil {
		return pgtype.Date{Time: t, Valid: true}, nil
	}

	if t, err := time.Parse(time.RFC3339, trimmed); err == nil {
		year, month, day := t.Date()
		dateOnly := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		return pgtype.Date{Time: dateOnly, Valid: true}, nil
	}

	return pgtype.Date{Valid: false}, fmt.Errorf("%w: expected YYYY-MM-DD or RFC3339", ErrInvalidBirthdate)
}

func normalizeGender(gender string) (NullGender, error) {
	trimmed := strings.TrimSpace(gender)
	if trimmed == "" {
		return NullGender{Valid: false}, nil
	}

	normalized := strings.ToLower(trimmed)
	switch Gender(normalized) {
	case GenderMale, GenderFemale, GenderOther:
		return NullGender{Gender: Gender(normalized), Valid: true}, nil
	default:
		return NullGender{}, fmt.Errorf("%w: must be one of male, female, other", ErrInvalidGender)
	}
}

func (store *Store) UpdateUserProfile(ctx context.Context, userID int64, displayName, birthdate, gender *string) (*User, error) {
	// Fetch the existing user
	user, err := store.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	// Update fields only if they are provided (non-nil)
	if displayName != nil {
		user.DisplayName = pgtype.Text{String: *displayName, Valid: true}
	}
	if birthdate != nil {
		parsedBirthdate, err := parseDate(*birthdate)
		if err != nil {
			return nil, err
		}
		user.Birthdate = parsedBirthdate
	}
	if gender != nil {
		normalizedGender, err := normalizeGender(*gender)
		if err != nil {
			return nil, err
		}
		user.Gender = normalizedGender
	}

	// Update the user in the database
	user, err = store.UpdateUser(ctx, UpdateUserParams{
		ID:          user.ID,
		DisplayName: user.DisplayName,
		Birthdate:   user.Birthdate,
		Gender:      user.Gender,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update user profile: %w", err)
	}

	return &user, nil
}
