create table users (
    id bigserial primary key,

    username varchar(255) not null unique,
    uuid uuid not null unique,
    credentials bytea not null,
    email varchar(255) null,
    avatar text null,

    created_at timestamp not null default now(),
    updated_at timestamp null,
    deleted_at timestamp null
);

create table pastes (
    id bigserial primary key,

    reference varchar(8) not null,
    title varchar(255) not null,
    owner text references users(username),

    content text not null,
    mimetype varchar(255) null,

    syntax varchar(50) null,
    tags text[],
    expiration timestamp null,
    public boolean not null default true,
    views integer default 0,
    password text null,

    created_at timestamp not null default now(),
    updated_at timestamp null,
    deleted_at timestamp null
);

create table sessions (
    id bigserial primary key,

    token uuid not null unique,
    username text not null,
    data jsonb not null,
    throwaway boolean not null default false,

    created_at timestamp not null default now(),
    expires_at timestamp not null
);
