-- +goose Up
-- +goose StatementBegin
CREATE TABLE rating
(
    id       serial primary key,
    username varchar(80) not null,
    stars    int         not null
        check (stars between 0 and 100)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table rating;
-- +goose StatementEnd
