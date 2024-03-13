create table if not exists orders (
    id uuid primary key,
    user_id uuid not null references users,
    number text not null,
    type text not null,
    income numeric(12,2) not null,
    outcome numeric(12,2) not null,
    status text not null,
    created_at timestamp not null,
    updated_at timestamp not null
);
alter table if exists orders drop constraint if exists orders_number_key;
alter table if exists orders add constraint orders_number_key unique (number);