create table pastes (
    id bigserial primary key,

    reference varchar(8) not null,
    title varchar(255) not null,

    content text not null,

    syntax varchar(50) null,
    tags text[],
    expiration timestamp null,
    public boolean not null default true,
    views integer default 0,

    created_at timestamp not null default now(),
    updated_at timestamp null,
    deleted_at timestamp null
);
