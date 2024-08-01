-- +migrate Up
create table "realms"
(
    "id"                    uuid  not null default gen_random_uuid(),
    "name"                  text  not null,
    "display_name"          text  not null,
    "encrypted_private_key" bytea not null,
    primary key ("id")
);

create unique index "idx_unique_realm_name" on "realms" ("name");

create table "clients"
(
    "id"                   uuid   not null default gen_random_uuid(),
    "realm_id"             uuid   not null,
    "display_name"         text   not null,
    "client_id"            text   not null,
    "hashed_client_secret" text   not null,
    "redirect_uris"        text[] not null,
    primary key ("id")
);

alter table "clients"
    add constraint "fk_clients_realms" foreign key ("realm_id") references "realms";

create table "users"
(
    "id"             uuid not null default gen_random_uuid(),
    "realm_id"       uuid not null,
    "username"       text,
    "email"          text,
    "email_verified" bool not null default false,
    primary key ("id")
);

alter table "users"
    add constraint "fk_users_realms" foreign key ("realm_id") references "realms";

create table "credentials"
(
    "id"      uuid  not null default gen_random_uuid(),
    "user_id" uuid  not null,
    "type"    text  not null,
    "details" jsonb not null,
    primary key ("id")
);

alter table "credentials"
    add constraint "fk_credentials_users" foreign key ("user_id") references "users";

create table "scopes"
(
    "id"           uuid not null default gen_random_uuid(),
    "realm_id"     uuid not null,
    "name"         text not null,
    "display_name" text not null,
    "description"  text not null,
    primary key ("id")
);

alter table "scopes"
    add constraint "fk_scopes_realms" foreign key ("realm_id") references "realms";

create unique index "idx_unique_scope_name_in_realm" on "scopes" ("name", "realm_id");

create table "grants"
(
    "id"        uuid not null default gen_random_uuid(),
    "scope_id"  uuid not null,
    "user_id"   uuid not null,
    "client_id" uuid not null,
    primary key ("id")
);

alter table "grants"
    add constraint "fk_grants_scopes" foreign key ("scope_id") references "scopes";
alter table "grants"
    add constraint "fk_grants_users" foreign key ("user_id") references "users";
alter table "grants"
    add constraint "fk_grants_clients" foreign key ("client_id") references "clients";

create unique index "idx_unique_grants" on "grants" ("scope_id", "user_id", "client_id");

create table "sessions"
(
    "id"           uuid not null,
    "user_id"      uuid not null,
    "realm_id"     uuid not null,
    "hashed_token" text not null,
    primary key ("id")
);

alter table "sessions"
    add constraint "fk_sessions_users" foreign key ("user_id") references "users";
alter table "sessions"
    add constraint "fk_sessions_realms" foreign key ("realm_id") references "realms";

create table "refresh_tokens"
(
    "id"           uuid      not null default gen_random_uuid(),
    "user_id"      uuid      not null,
    "client_id"    uuid      not null,
    "realm_id"     uuid      not null,
    "hashed_token" text      not null,
    "valid_until"  timestamp not null,
    "issuer"       text      not null,
    "subject"      text      not null,
    "audience"     text      not null,
    "scopes"       text[]    not null,
    primary key ("id")
);

alter table "refresh_tokens"
    add constraint "fk_refresh_tokens_users" foreign key ("user_id") references "users";
alter table "refresh_tokens"
    add constraint "fk_refresh_tokens_clients" foreign key ("client_id") references "clients";
alter table "refresh_tokens"
    add constraint "fk_refresh_tokens_realms" foreign key ("realm_id") references "realms";

create table "claim_mappers"
(
    "id"           uuid  not null default gen_random_uuid(),
    "realm_id"     uuid  not null,
    "display_name" text  not null,
    "description"  text  not null,
    "type"         text  not null,
    "details"      jsonb not null,
    primary key ("id")
);

create unique index "idx_unique_realm_claim" on "claim_mappers" ("display_name", "realm_id");

alter table "claim_mappers"
    add constraint "fk_claim_mappers_realms" foreign key ("realm_id") references "realms";

create table "scope_claims"
(
    "id"              uuid not null default gen_random_uuid(),
    "scope_id"        uuid not null,
    "claim_mapper_id" uuid not null,
    primary key ("id")
);

alter table "scope_claims"
    add constraint "fk_scope_claims_scopes" foreign key ("scope_id") references "scopes";

-- +migrate Down
drop table "sessions" cascade;
drop table "grants" cascade;
drop table "scopes" cascade;
drop table "credentials" cascade;
drop table "clients" cascade;
drop table "users" cascade;
drop table "realms" cascade;
drop table "claim_mappers" cascade;
drop table "scope_claims" cascade;
