CREATE TABLE IF NOT EXISTS landlord.account
(
    id
    INT
    NOT
    NULL
    AUTO_INCREMENT
    PRIMARY
    KEY,
    email
    VARCHAR
(
    20
) NOT NULL UNIQUE,
    username VARCHAR
(
    10
) NOT NULL,
    password VARCHAR
(
    100
) NOT NULL,
    coin INT default 4000,
    created_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
    );