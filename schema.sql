/* Certificate serials must be generated before certificate issuance.
 * While it may seem somehwat wasteful to have a table just for this
 * purpose it offers the opportunity to LEFT JOIN against the
 * certificate table to find any serials that were generated but which
 * did not result in an accompanying certificate. I.e. some problem with
 * signing that certificate. The timestamp here could be used to look at
 * the logs for that case. */
CREATE TABLE serials (
    serial BIGINT NOT NULL AUTO_INCREMENT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (serial)
);

CREATE TABLE certificates (
    serial BIGINT NOT NULL,

    -- the name field should contain either the common name of the
    -- certificate or, if the CN is empty, the SHA-256 of the entire
    -- certificate DER bytes.
    name             VARCHAR(1024) NULL,
    not_valid_before DATETIME NOT NULL,
    not_valid_after  DATETIME NOT NULL,
    certificate_pem  TEXT NOT NULL,
    revoked          BOOLEAN NOT NULL DEFAULT 0,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (serial),

    FOREIGN KEY (serial)
        REFERENCES serials (serial),

    CHECK (SUBSTRING(certificate_pem FROM 1 FOR 27) = '-----BEGIN CERTIFICATE-----'),
    CHECK (name IS NULL OR name != '')
);

CREATE TABLE ca_keys (
    serial BIGINT NOT NULL,

    -- mysqlscepserver encrypts the the private key and encodes it in
    -- PEM armour. this uses the "CA pass" provided to the SCEP server.
    key_pem  TEXT NOT NULL,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (serial),

    FOREIGN KEY (serial)
        REFERENCES certificates (serial),

    CHECK (SUBSTRING(key_pem  FROM 1 FOR  5) = '-----')
);

/* The challenges table contains generated challenges created using the
 * API. If the challenge is successfully validated it is deleted from
 * this table (i.e. no longer valid). Ideally no outstanding unverified
 * challenges would exist in the table. */
CREATE TABLE challenges (
    challenge CHAR(32),

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (challenge)
);
