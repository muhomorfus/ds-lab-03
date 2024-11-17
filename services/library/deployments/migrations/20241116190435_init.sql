-- +goose Up
-- +goose StatementBegin
create table library
(
    id          serial primary key,
    library_uid uuid unique  not null,
    name        varchar(80)  not null,
    city        varchar(255) not null,
    address     varchar(255) not null
);

insert into library (id, library_uid, name, city, address) values
    (1, '83575e12-7ce0-48ee-9931-51919ff3c9ee', 'Библиотека имени 7 Непьющих', 'Москва', '2-я Бауманская ул., д.5, стр.1');

CREATE TABLE books
(
    id        serial primary key,
    book_uid  uuid unique  not null,
    name      varchar(255) not null,
    author    varchar(255),
    genre     varchar(255),
    condition varchar(20) default 'EXCELLENT'
        check (condition in ('EXCELLENT', 'GOOD', 'BAD'))
);

insert into books (id, book_uid, name, author, genre, condition) values
    (1, 'f7cdc58f-2caf-4b15-9727-f89dcc629b27', 'Краткий курс C++ в 7 томах', 'Бьерн Страуструп', 'Научная фантастика', 'EXCELLENT');

create table library_books
(
    book_id         int references books (id),
    library_id      int references library (id),
    available_count int not null
);

insert into library_books (book_id, library_id, available_count) values
    (1, 1, 1);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table library_books;
drop table books;
drop table library;
-- +goose StatementEnd
