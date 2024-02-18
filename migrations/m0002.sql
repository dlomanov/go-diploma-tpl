create table if not exists balances (
    user_id uuid primary key references users,
    current numeric(12,2) not null,
    withdrawn numeric(12,2) not null,
    created_at timestamp not null,
    updated_at timestamp not null
);