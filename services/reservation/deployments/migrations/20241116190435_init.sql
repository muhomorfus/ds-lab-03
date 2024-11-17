-- +goose Up
-- +goose StatementBegin
create table reservation
(
    id              serial primary key,
    reservation_uid uuid unique not null,
    username        varchar(80) not null,
    book_uid        uuid        not null,
    library_uid     uuid        not null,
    status          varchar(20) not null
        check (status in ('RENTED', 'RETURNED', 'EXPIRED')),
    start_date      timestamp   not null,
    till_date       timestamp   not null
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table reservation;
-- +goose StatementEnd
