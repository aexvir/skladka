-- name: CreateSession :one
insert into sessions
(token, username, data, expires_at)
values ($1, $2, $3, $4)
returning *;

-- name: GetSessionByToken :one
select *
from sessions
where token = $1
    and expires_at > now();

-- name: DeleteExpiredSessions :exec
delete from sessions
where expires_at <= now();

-- name: DeleteSession :exec
delete from sessions
where token = $1;
