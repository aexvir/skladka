-- name: CreateUser :one
insert into users
(username, uuid, credentials)
values ($1, $2, $3)
returning *;

-- name: GetUserByUsername :one
select *
from users
where username = $1
    and deleted_at is null;

-- name: UpdateUserCredentials :exec
update users
set credentials = $2
where username = $1;

-- name: UpdateUserAvatar :exec
update users
set avatar = $2
where username = $1;

-- name: UpdateUserEmail :exec
update users
set email =$2
where username = $1;

-- name: DeleteUser :exec
update users
set deleted_at = now()
where id = $1;
