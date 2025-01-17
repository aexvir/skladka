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
(reference, title, content, syntax, tags, expiration, public, password)
values ($1, $2, $3, $4, $5, $6, $7, $8)
returning id;

-- name: ListPublicPastes :many
select *
from pastes
where public = true
    and deleted_at is null
order by created_at desc;
