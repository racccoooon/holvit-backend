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
}

type RealmFilter struct {
	BaseFilter

	Name h.Optional[string]
}

type RealmUpdate struct {
	DisplayName h.Optional[string]
	Name        h.Optional[string]

	RequireUsername           h.Optional[bool]
	RequireEmail              h.Optional[bool]
	RequireDeviceVerification h.Optional[bool]
	RequireTotp               h.Optional[bool]
	EnableRememberMe          h.Optional[bool]
}

type RealmRepository interface {
	FindRealmById(ctx context.Context, id uuid.UUID) h.Optional[Realm]
	FindRealms(ctx context.Context, filter RealmFilter) FilterResult[Realm]
	CreateRealm(ctx context.Context, realm *Realm) h.Result[uuid.UUID]
	UpdateRealm(ctx context.Context, id uuid.UUID, upd RealmUpdate) h.Result[h.Unit]
}

type RealmRepositoryImpl struct {
}

func NewRealmRepository() RealmRepository {
	return &RealmRepositoryImpl{}
}

func (r *RealmRepositoryImpl) FindRealmById(ctx context.Context, id uuid.UUID) h.Optional[Realm] {
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

	sb := sqlbuilder.Select(filter.CountCol(),
		"id", "name", "display_name", "encrypted_private_key", "require_username", "require_email", "require_device_verification", "require_totp", "enable_remember_me").
		From("realms")

	filter.Id.IfSome(func(x uuid.UUID) {
		sb.Where(sb.Equal("id", x))
	})

	filter.Name.IfSome(func(x string) {
		sb.Where(sb.Equal("name", x))
	})

	filter.PagingInfo.IfSome(func(x PagingInfo) {
		x.Apply(sb)
	})

	sqlString, args := sb.Build()
	logging.Logger.Debugf("executing sql: %s", sqlString)
	rows, err := tx.Query(sqlString, args...)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

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
			&row.EnableRememberMe)
		if err != nil {
			panic(err)
		}
		result = append(result, row)
	}

	return NewPagedResult(result, totalCount)
}

func (r *RealmRepositoryImpl) CreateRealm(ctx context.Context, realm *Realm) h.Result[uuid.UUID] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	sqlString := `insert into "realms"
    			("name", "display_name", "encrypted_private_key", "require_username", "require_email", "require_device_verification", "require_totp", "enable_remember_me")
    			values ($1, $2, $3, $4, $5, $6, $7, $8)
    			returning "id"`
	logging.Logger.Debugf("executing sql: %s", sqlString)

	err = tx.QueryRow(sqlString,
		realm.Name,
		realm.DisplayName,
		realm.EncryptedPrivateKey,
		realm.RequireUsername,
		realm.RequireEmail,
		realm.RequireDeviceVerification,
		realm.RequireTotp,
		realm.EnableRememberMe).Scan(&resultingId)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				if pqErr.Constraint == "idx_unique_realm_name" {
					return h.Err[uuid.UUID](DuplicateUsernameError{})
				}
				break
			}
		} else {
			panic(err)
		}
	}

	return h.Ok(resultingId)
}

func (r *RealmRepositoryImpl) UpdateRealm(ctx context.Context, id uuid.UUID, upd RealmUpdate) h.Result[h.Unit] {
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
				break
			}
		} else {
			panic(err)
		}
	}

	return h.UOk()
}
