-- +migrate Up
create extension if not exists "citext";

-- +migrate StatementBegin
create or replace function update_audit_timestamp()
    returns trigger as
$$
begin
    new.audit_updated_at = now();
    -- TODO: new.audit_updated_by_user_id = current_setting('app.current_user_id')::uuid;
    return new;
end;
$$ language plpgsql;
-- +migrate StatementEnd

create table "realms"
(
    "id"                          uuid      not null default gen_random_uuid(),
    "audit_created_at"            timestamp not null default now(),
    "audit_updated_at"            timestamp not null default now(),

    "name"                        text      not null,
    "display_name"                text      not null,
    "encrypted_private_key"       bytea     not null,

    "require_username"            bool      not null,
    "require_email"               bool      not null,
    "require_device_verification" bool      not null,
    "require_totp"                bool      not null,
    "enable_remember_me"          bool      not null,

    "password_history_length"     int       not null,

    primary key ("id")
);

create trigger "set_audit_updated_at"
    before update
    on "realms"
    for each row
execute function update_audit_timestamp();

create unique index "idx_unique_realm_name" on "realms" ("name");

create table "clients"
(
    "id"                   uuid      not null default gen_random_uuid(),
    "audit_created_at"     timestamp not null default now(),
    "audit_updated_at"     timestamp not null default now(),
    "realm_id"             uuid      not null,
    "display_name"         text      not null,
    "client_id"            text      not null,
    "hashed_client_secret" text      null,
    "redirect_uris"        text[]    not null,
    primary key ("id")
);

create trigger "set_audit_updated_at"
    before update
    on "clients"
    for each row
execute function update_audit_timestamp();

create unique index "idx_unique_client_id_per_realm" on "clients" ("client_id", "realm_id");

alter table "clients"
    add constraint "fk_clients_realms" foreign key ("realm_id") references "realms";

create table "users"
(
    "id"               uuid      not null default gen_random_uuid(),
    "audit_created_at" timestamp not null default now(),
    "audit_updated_at" timestamp not null default now(),
    "realm_id"         uuid      not null,
    "username"         text      not null,
    "email"            citext    null,
    "email_verified"   bool      not null default false,
    primary key ("id")
);

create trigger "set_audit_updated_at"
    before update
    on "users"
    for each row
execute function update_audit_timestamp();

alter table "users"
    add constraint "fk_users_realms" foreign key ("realm_id") references "realms";

create unique index "idx_unique_username_per_realm" on "users" ("realm_id", lower("username"));

create table "credentials"
(
    "id"               uuid      not null default gen_random_uuid(),
    "audit_created_at" timestamp not null default now(),
    "audit_updated_at" timestamp not null default now(),
    "user_id"          uuid      not null,
    "type"             text      not null,
    "details"          jsonb     not null,
    primary key ("id")
);

create trigger "set_audit_updated_at"
    before update
    on "credentials"
    for each row
execute function update_audit_timestamp();

create unique index "idx_only_one_password_per_user" on "credentials" ("user_id", "type")
    where type = 'password';

alter table "credentials"
    add constraint "fk_credentials_users" foreign key ("user_id") references "users";

create table "password_history"
(
    "id"               uuid      not null default gen_random_uuid(),
    "audit_created_at" timestamp not null default now(),
    "audit_updated_at" timestamp not null default now(),
    "user_id"          uuid      not null,
    "hashed_password"  text      not null,
    "created_at"       timestamp not null,
    primary key ("id")
);

create trigger "set_audit_updated_at"
    before update
    on "password_history"
    for each row
execute function update_audit_timestamp();

alter table "password_history"
    add constraint "fk_password_history_users" foreign key ("user_id") references "users";

create table "scopes"
(
    "id"               uuid      not null default gen_random_uuid(),
    "audit_created_at" timestamp not null default now(),
    "audit_updated_at" timestamp not null default now(),
    "realm_id"         uuid      not null,
    "name"             text      not null,
    "display_name"     text      not null,
    "description"      text      not null,
    "sort_index"       int       not null,
    primary key ("id")
);

create trigger "set_audit_updated_at"
    before update
    on "scopes"
    for each row
execute function update_audit_timestamp();

alter table "scopes"
    add constraint "fk_scopes_realms" foreign key ("realm_id") references "realms";

create unique index "idx_unique_scope_name_in_realm" on "scopes" ("name", "realm_id");

create table "grants"
(
    "id"               uuid      not null default gen_random_uuid(),
    "audit_created_at" timestamp not null default now(),
    "audit_updated_at" timestamp not null default now(),
    "scope_id"         uuid      not null,
    "user_id"          uuid      not null,
    "client_id"        uuid      not null,
    primary key ("id")
);

create trigger "set_audit_updated_at"
    before update
    on "grants"
    for each row
execute function update_audit_timestamp();

alter table "grants"
    add constraint "fk_grants_scopes" foreign key ("scope_id") references "scopes";
alter table "grants"
    add constraint "fk_grants_users" foreign key ("user_id") references "users";
alter table "grants"
    add constraint "fk_grants_clients" foreign key ("client_id") references "clients";

create unique index "idx_unique_grants" on "grants" ("scope_id", "user_id", "client_id");

create table "user_devices"
(
    "id"               uuid      not null default gen_random_uuid(),
    "audit_created_at" timestamp not null default now(),
    "audit_updated_at" timestamp not null default now(),
    "user_id"          uuid      not null,
    "device_id"        text      not null,
    "display_name"     text      not null,
    "user_agent"       text      not null,
    "last_ip"          inet      not null,
    "last_login_at"    timestamp not null,
    primary key ("id")
);

create trigger "set_audit_updated_at"
    before update
    on "user_devices"
    for each row
execute function update_audit_timestamp();

alter table "user_devices"
    add constraint "fk_user_devices_users" foreign key ("user_id") references "users";

create table "sessions"
(
    "id"               uuid      not null default gen_random_uuid(),
    "audit_created_at" timestamp not null default now(),
    "audit_updated_at" timestamp not null default now(),
    "user_id"          uuid      not null,
    "user_device_id"   uuid      not null,
    "realm_id"         uuid      not null,
    "hashed_token"     text      not null,
    "valid_until"      timestamp not null,
    primary key ("id")
);

create trigger "set_audit_updated_at"
    before update
    on "sessions"
    for each row
execute function update_audit_timestamp();

alter table "sessions"
    add constraint "fk_sessions_users" foreign key ("user_id") references "users";
alter table "sessions"
    add constraint "fk_sessions_realms" foreign key ("realm_id") references "realms";
alter table "sessions"
    add constraint "fk_sessions_user_devices" foreign key ("user_device_id") references "user_devices";

create table "refresh_tokens"
(
    "id"               uuid      not null default gen_random_uuid(),
    "audit_created_at" timestamp not null default now(),
    "audit_updated_at" timestamp not null default now(),
    "user_id"          uuid      not null,
    "client_id"        uuid      not null,
    "realm_id"         uuid      not null,
    "hashed_token"     text      not null,
    "valid_until"      timestamp not null,
    "issuer"           text      not null,
    "subject"          text      not null,
    "audience"         text      not null,
    "scopes"           text[]    not null,
    primary key ("id")
);

create trigger "set_audit_updated_at"
    before update
    on "refresh_tokens"
    for each row
execute function update_audit_timestamp();

create unique index "idx_unique_refresh_token" on "refresh_tokens" ("hashed_token");

alter table "refresh_tokens"
    add constraint "fk_refresh_tokens_users" foreign key ("user_id") references "users";
alter table "refresh_tokens"
    add constraint "fk_refresh_tokens_clients" foreign key ("client_id") references "clients";
alter table "refresh_tokens"
    add constraint "fk_refresh_tokens_realms" foreign key ("realm_id") references "realms";

create table "claim_mappers"
(
    "id"               uuid      not null default gen_random_uuid(),
    "audit_created_at" timestamp not null default now(),
    "audit_updated_at" timestamp not null default now(),
    "realm_id"         uuid      not null,
    "display_name"     text      not null,
    "description"      text      not null,
    "type"             text      not null,
    "details"          jsonb     not null,
    primary key ("id")
);

create trigger "set_audit_updated_at"
    before update
    on "claim_mappers"
    for each row
execute function update_audit_timestamp();

create unique index "idx_unique_realm_claim" on "claim_mappers" ("display_name", "realm_id");

alter table "claim_mappers"
    add constraint "fk_claim_mappers_realms" foreign key ("realm_id") references "realms";

create table "scope_claims"
(
    "id"               uuid      not null default gen_random_uuid(),
    "audit_created_at" timestamp not null default now(),
    "audit_updated_at" timestamp not null default now(),
    "scope_id"         uuid      not null,
    "claim_mapper_id"  uuid      not null,
    primary key ("id")
);

create trigger "set_audit_updated_at"
    before update
    on "scope_claims"
    for each row
execute function update_audit_timestamp();

alter table "scope_claims"
    add constraint "fk_scope_claims_scopes" foreign key ("scope_id") references "scopes";

-- +migrate StatementBegin
do
$$
    begin
        create type job_status as enum ('pending', 'completed', 'failed');
    exception
        when duplicate_object then null;
    end
$$;
-- +migrate StatementEnd

create table "queued_jobs"
(
    "id"               uuid       not null default gen_random_uuid(),
    "audit_created_at" timestamp  not null default now(),
    "audit_updated_at" timestamp  not null default now(),
    "status"           job_status not null default 'pending',
    "type"             text       not null,
    "details"          jsonb      not null,
    "failure_count"    int        not null,
    "error"            text       null,
    primary key ("id")
);

create trigger "set_audit_updated_at"
    before update
    on "queued_jobs"
    for each row
execute function update_audit_timestamp();

create table "roles"
(
    "id"               uuid      not null default gen_random_uuid(),
    "audit_created_at" timestamp not null default now(),
    "audit_updated_at" timestamp not null default now(),
    "realm_id"         uuid      not null,
    "client_id"        uuid      null,
    "display_name"     text      not null,
    "name"             text      not null,
    "description"      text      not null,
    "implies_cache"    uuid[]    not null default [],
    primary key ("id")
);

create trigger "set_audit_updated_at"
    before update
    on "roles"
    for each row
execute function update_audit_timestamp();

create unique index "idx_unique_role_per_realm" on "roles" ("name", "realm_id");

alter table "roles"
    add constraint "fk_roles_realms" foreign key ("realm_id") references "realms";

alter table "roles"
    add constraint "fk_roles_clients" foreign key ("client_id") references "clients";

create table "role_implications"
(
    "id"               uuid      not null default gen_random_uuid(),
    "audit_created_at" timestamp not null default now(),
    "audit_updated_at" timestamp not null default now(),
    "role_id"          uuid      not null,
    "implied_role_id"  uuid      not null,
    primary key ("id")
);

create trigger "set_audit_updated_at"
    before update
    on "role_implications"
    for each row
execute function update_audit_timestamp();

alter table "role_implications"
    add constraint "fk_implied_roles_role" foreign key ("role_id") references "roles" on delete cascade;

alter table "role_implications"
    add constraint "fk_implied_roles_implied_role" foreign key ("implied_role_id") references "roles" on delete cascade;

create table "user_roles"
(
    "id"               uuid      not null default gen_random_uuid(),
    "audit_created_at" timestamp not null default now(),
    "audit_updated_at" timestamp not null default now(),
    "user_id"          uuid      not null,
    "role_id"          uuid      not null,
    primary key ("id")
);

create trigger "set_audit_updated_at"
    before update
    on "user_roles"
    for each row
execute function update_audit_timestamp();

alter table "user_roles"
    add constraint "fk_user_roles_users" foreign key ("user_id") references "users";
alter table "user_roles"
    add constraint "fk_user_roles_roles" foreign key ("role_id") references "roles";

-- +migrate Down
drop table "password_history" cascade;
drop table "user_roles" cascade;
drop table "role_implications" cascade;
drop table "roles" cascade;
drop table "queued_jobs" cascade;
drop table "user_devices" cascade;
drop table "sessions" cascade;
drop table "grants" cascade;
drop table "scopes" cascade;
drop table "credentials" cascade;
drop table "clients" cascade;
drop table "users" cascade;
drop table "realms" cascade;
drop table "claim_mappers" cascade;
drop table "scope_claims" cascade;
