package repos

import (
	"context"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"github.com/lib/pq"
	"holvit/h"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"holvit/requestContext"
	"holvit/sqlb"
	"holvit/utils"
)

type Realm struct {
	BaseModel

	Name        string
	DisplayName string

	EncryptedPrivateKey []byte

	RequireUsername           bool
	RequireEmail              bool
	RequireDeviceVerification bool
	RequireTotp               bool
	EnableRememberMe          bool
	PasswordHistoryLength     int
}

type RealmFilter struct {
	BaseFilter

	Name h.Opt[string]
}

type RealmUpdate struct {
	DisplayName h.Opt[string]
	Name        h.Opt[string]

	RequireUsername           h.Opt[bool]
	RequireEmail              h.Opt[bool]
	RequireDeviceVerification h.Opt[bool]
	RequireTotp               h.Opt[bool]
	EnableRememberMe          h.Opt[bool]
}

type RealmRepository interface {
	FindRealmById(ctx context.Context, id uuid.UUID) h.Opt[Realm]
	FindRealms(ctx context.Context, filter RealmFilter) FilterResult[Realm]
	CreateRealm(ctx context.Context, realm Realm) h.Result[uuid.UUID]
	UpdateRealm(ctx context.Context, id uuid.UUID, upd RealmUpdate) h.UResult
}

type RealmRepositoryImpl struct {
}

func NewRealmRepository() RealmRepository {
	return &RealmRepositoryImpl{}
}

func (r *RealmRepositoryImpl) FindRealmById(ctx context.Context, id uuid.UUID) h.Opt[Realm] {
	return r.FindRealms(ctx, RealmFilter{
		BaseFilter: BaseFilter{
			Id:         h.Some(id),
			PagingInfo: h.Some(NewPagingInfo(1, 0)),
		},
	}).FirstOrNone()
}

func (r *RealmRepositoryImpl) FindRealms(ctx context.Context, filter RealmFilter) FilterResult[Realm] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	q := sqlb.Select(filter.CountCol(),
		"id", "name", "display_name", "encrypted_private_key", "require_username", "require_email",
		"require_device_verification", "require_totp", "enable_remember_me", "password_history_length").
		From("realms")

	filter.Id.IfSome(func(x uuid.UUID) {
		q.Where("id = ?", x)
	})

	filter.Name.IfSome(func(x string) {
		q.Where("name = ?", x)
	})

	filter.PagingInfo.IfSome(func(x PagingInfo) {
		x.Apply(q)
	})

	filter.SortInfo.IfSome(func(x SortInfo) {
		x.Apply(q)
	})

	query := q.Build()
	logging.Logger.Debugf("executing sql: %s", query.Query)
	rows, err := tx.Query(query.Query, query.Parameters...)
	if err != nil {
		panic(mapCustomErrorCodes(err))
	}
	defer utils.PanicOnErr(rows.Close)

	var totalCount int
	var result []Realm
	for rows.Next() {
		var row Realm
		err := rows.Scan(&totalCount,
			&row.Id,
			&row.Name,
			&row.DisplayName,
			&row.EncryptedPrivateKey,
			&row.RequireUsername,
			&row.RequireEmail,
			&row.RequireDeviceVerification,
			&row.RequireTotp,
			&row.EnableRememberMe,
			&row.PasswordHistoryLength)
		if err != nil {
			panic(err)
		}
		result = append(result, row)
	}

	return NewPagedResult(result, totalCount)
}

func (r *RealmRepositoryImpl) CreateRealm(ctx context.Context, realm Realm) h.Result[uuid.UUID] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	q := sqlb.InsertInto("realms", "name", "display_name", "encrypted_private_key", "require_username", "require_email", "require_device_verification", "require_totp", "enable_remember_me", "password_history_length").
		Values(realm.Name,
			realm.DisplayName,
			realm.EncryptedPrivateKey,
			realm.RequireUsername,
			realm.RequireEmail,
			realm.RequireDeviceVerification,
			realm.RequireTotp,
			realm.EnableRememberMe,
			realm.PasswordHistoryLength).
		Returning("id")

	query := q.Build()
	logging.Logger.Debugf("executing sql: %s", query.Query)
	err = tx.QueryRow(query.Query, query.Parameters...).Scan(&resultingId)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				if pqErr.Constraint == "idx_unique_realm_name" {
					return h.Err[uuid.UUID](DuplicateUsernameError{}) //TODO: correct error
				}
			}
		}

		panic(mapCustomErrorCodes(err))
	}

	return h.Ok(resultingId)
}

func (r *RealmRepositoryImpl) UpdateRealm(ctx context.Context, id uuid.UUID, upd RealmUpdate) h.UResult {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	sb := sqlbuilder.Update("realms")

	upd.Name.IfSome(func(x string) {
		sb.Set(sb.Assign("name", x))
	})

	upd.DisplayName.IfSome(func(x string) {
		sb.Set(sb.Assign("display_name", x))
	})

	upd.RequireUsername.IfSome(func(x bool) {
		sb.Set(sb.Assign("require_username", x))
	})

	upd.RequireEmail.IfSome(func(x bool) {
		sb.Set(sb.Assign("require_email", x))
	})

	upd.RequireTotp.IfSome(func(x bool) {
		sb.Set(sb.Assign("require_totp", x))
	})

	upd.EnableRememberMe.IfSome(func(x bool) {
		sb.Set(sb.Assign("enable_remember_me", x))
	})

	upd.RequireDeviceVerification.IfSome(func(x bool) {
		sb.Set(sb.Assign("require_device_verification", x))
	})

	sb.Where(sb.Equal("id", id))

	sqlString, args := sb.Build()
	logging.Logger.Debugf("executing sql: %s", sqlString)
	_, err = tx.Exec(sqlString, args...)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				if pqErr.Constraint == "idx_unique_realm_name" {
					return h.Err[h.Unit](DuplicateUsernameError{})
				}
			}
		} else {
			panic(err)
		}
	}

	return h.UOk()
}
