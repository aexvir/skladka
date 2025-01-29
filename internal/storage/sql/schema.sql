create table users (
    id bigserial primary key,

    username varchar(255) not null unique,
    uuid uuid not null unique,
    credentials bytea not null,
    email varchar(255) null,

    created_at timestamp not null default now(),
    updated_at timestamp null,
    deleted_at timestamp null
);

create table pastes (
    id bigserial primary key,

    reference varchar(8) not null,
    title varchar(255) not null,
    owner bigint references users(id),

    content text not null,

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

    -- Session token used in cookies
    token uuid not null unique,

    -- User that owns this session, nullable for registration sessions
    username text not null,

    -- Session data stored as JSON
    data jsonb not null,

    -- Sessions expire after a certain time
    created_at timestamp not null default now(),
    expires_at timestamp not null
);
