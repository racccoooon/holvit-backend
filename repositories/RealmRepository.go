package repositories

import (
	"context"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"holvit/httpErrors"
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

	RequireUsername  bool
	RequireEmail     bool
	RequireTotp      bool
	EnableRememberMe bool
}

type RealmFilter struct {
	BaseFilter

	Name *string
}

type RealmUpdate struct {
	DisplayName *string
	Name        *string

	RequireUsername  *bool
	RequireEmail     *bool
	RequireTotp      *bool
	EnableRememberMe *bool
}

type RealmRepository interface {
	FindRealmById(ctx context.Context, id uuid.UUID) (*Realm, error)
	FindRealms(ctx context.Context, filter RealmFilter) ([]*Realm, int, error)
	CreateRealm(ctx context.Context, realm *Realm) (uuid.UUID, error)
	UpdateRealm(ctx context.Context, id uuid.UUID, upd RealmUpdate) error
}

type RealmRepositoryImpl struct {
}

func NewRealmRepository() RealmRepository {
	return &RealmRepositoryImpl{}
}

func (r *RealmRepositoryImpl) FindRealmById(ctx context.Context, id uuid.UUID) (*Realm, error) {
	result, resultCount, err := r.FindRealms(ctx, RealmFilter{
		BaseFilter: BaseFilter{
			Id: id,
			PagingInfo: PagingInfo{
				PageSize:   1,
				PageNumber: 0,
			},
		},
	})

	if err != nil {
		return nil, err
	}
	if resultCount == 0 {
		return nil, httpErrors.NotFound()
	}
	return result[0], nil
}

func (r *RealmRepositoryImpl) FindRealms(ctx context.Context, filter RealmFilter) ([]*Realm, int, error) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		return nil, 0, err
	}

	sb := sqlbuilder.Select("count(*) over()",
		"id", "name", "display_name", "encrypted_private_key", "require_username", "require_email", "require_totp", "enable_remember_me").
		From("realms")

	if filter.Name != nil {
		sb.Where(sb.Equal("name", *filter.Name))
	}

	if filter.PagingInfo.PageSize > 0 {
		sb.Limit(filter.PagingInfo.PageSize).
			Offset(filter.PagingInfo.PageSize * (filter.PagingInfo.PageNumber - 1))
	}

	sqlString, args := sb.Build()
	logging.Logger.Debugf("executing sql: %s", sqlString)
	rows, err := tx.Query(sqlString, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var totalCount int
	var result []*Realm
	for rows.Next() {
		var row Realm
		err := rows.Scan(&totalCount,
			&row.Id,
			&row.Name,
			&row.DisplayName,
			&row.EncryptedPrivateKey,
			&row.RequireUsername,
			&row.RequireEmail,
			&row.RequireTotp,
			&row.EnableRememberMe)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, &row)
	}

	return result, totalCount, nil
}

func (r *RealmRepositoryImpl) CreateRealm(ctx context.Context, realm *Realm) (uuid.UUID, error) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		return resultingId, err
	}

	err = tx.QueryRow(`insert into "realms"
    			("name", "display_name", "encrypted_private_key", "require_username", "require_email", "require_totp", "enable_remember_me")
    			values ($1, $2, $3, $4, $5, $6, $7)
    			returning "id"`,
		realm.Name,
		realm.DisplayName,
		realm.EncryptedPrivateKey,
		realm.RequireUsername,
		realm.RequireEmail,
		realm.RequireTotp,
		realm.EnableRememberMe).Scan(&resultingId)

	return resultingId, err
}

func (r *RealmRepositoryImpl) UpdateRealm(ctx context.Context, id uuid.UUID, upd RealmUpdate) error {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		return err
	}

	sb := sqlbuilder.Update("realms")

	if upd.Name != nil {
		sb.Set(sb.Assign("name", *upd.Name))
	}

	if upd.DisplayName != nil {
		sb.Set(sb.Assign("display_name", *upd.DisplayName))
	}

	if upd.RequireUsername != nil {
		sb.Set(sb.Assign("require_username", *upd.RequireUsername))
	}

	if upd.RequireEmail != nil {
		sb.Set(sb.Assign("require_email", *upd.RequireEmail))
	}

	if upd.RequireTotp != nil {
		sb.Set(sb.Assign("require_totp", *upd.RequireTotp))
	}

	if upd.EnableRememberMe != nil {
		sb.Set(sb.Assign("enable_remember_me", *upd.EnableRememberMe))
	}

	sb.Where(sb.Equal("id", id))

	sqlString, args := sb.Build()
	logging.Logger.Debugf("executing sql: %s", sqlString)
	_, err = tx.Exec(sqlString, args...)
	if err != nil {
		return err
	}

	return nil
}
