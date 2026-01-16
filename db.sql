create database "data-open-studio";
create database "test-input";

\c "data-open-studio";
create table test(
    id serial primary key,
    nom varchar(255),
    prenom varchar(255),
    age int
);

INSERT INTO test (nom, prenom, age)
SELECT
    'Nom_' || i,
    'Prenom_' || i,
    (random() * 60 + 18)::int
FROM generate_series(1, 1000) i;

\c "test-input";
CREATE TABLE receiver (
                          age INT
);

create table sender (id serial primary key, hobby varchar, nom varchar);

insert into sender (hobby, nom) SELECT
                                    'hobby_' || i,
                                    'Nom_' || i
FROM generate_series(1, 1000) i;