CREATE TABLE serials (
    serial BIGINT NOT NULL AUTO_INCREMENT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (serial)
);

CREATE TABLE certificates (
    serial BIGINT NOT NULL,

    cn               VARCHAR(1024) NULL,
    not_valid_before DATETIME NOT NULL,
    not_valid_after  DATETIME NOT NULL,
    certificate_pem  TEXT NOT NULL,
    revoked          BOOLEAN NOT NULL DEFAULT 0,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (serial),

    FOREIGN KEY (serial)
        REFERENCES serials (serial),

    CHECK (SUBSTRING(certificate_pem FROM 1 FOR 27) = '-----BEGIN CERTIFICATE-----')
);

