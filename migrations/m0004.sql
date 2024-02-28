create table if not exists jobs (
    id uuid primary key,
    type text not null,
    status text not null,
    entity_id uuid not null,
    attempt integer not null,
    last_error text,
    next_attempt_at timestamp,
    created_at timestamp not null,
    updated_at timestamp not null
)