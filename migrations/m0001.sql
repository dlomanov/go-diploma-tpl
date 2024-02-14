create table if not exists users (
    id uuid primary key,
    login text not null,
    pass_hash text not null,
    created_at timestamp not null
);