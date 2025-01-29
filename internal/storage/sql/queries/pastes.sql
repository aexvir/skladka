-- name: GetPasteByID :one
select *
from pastes
where id = $1
    and deleted_at is null;

-- name: GetPasteByReference :one
update pastes
set views = views + 1
where reference = $1
    and deleted_at is null
returning *;

-- name: CreatePaste :one
insert into pastes
(reference, owner, title, content, syntax, tags, expiration, public, password)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
returning id;

-- name: ListPublicPastes :many
select *
from pastes
where public = true
    and deleted_at is null
order by created_at desc;

-- name: DeletePaste :exec
update pastes
set deleted_at = now()
where reference = $1
    and deleted_at is null;

-- name: DeleteExpiredPastes :many
update pastes
set deleted_at = now()
where expiration < now() and deleted_at is null
returning reference;
