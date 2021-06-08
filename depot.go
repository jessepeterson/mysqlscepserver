package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"

	_ "github.com/go-sql-driver/mysql"
	"github.com/micromdm/scep/v2/depot"
)

type MySQLDepot struct {
	db  *sql.DB
	ctx context.Context
	crt *x509.Certificate
	key *rsa.PrivateKey
}

func NewMySQLDepot(conn string) (*MySQLDepot, error) {
	db, err := sql.Open("mysql", conn)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return &MySQLDepot{
		db:  db,
		ctx: context.Background(),
	}, nil
}

func (d *MySQLDepot) loadCA(pass []byte) (*x509.Certificate, *rsa.PrivateKey, error) {
	var pemCert, pemKey []byte
	err := d.db.QueryRowContext(
		d.ctx, `
SELECT
    certificate_pem, key_pem
FROM
    certificates INNER JOIN ca_keys
        ON certificates.serial = ca_keys.serial
WHERE
    certificates.serial = 1;`,
	).Scan(&pemCert, &pemKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	block, _ := pem.Decode(pemCert)
	if block.Type != "CERTIFICATE" {
		return nil, nil, errors.New("PEM block not a certificate")
	}
	crt, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, err
	}
	block, _ = pem.Decode(pemKey)
	if !x509.IsEncryptedPEMBlock(block) {
		return nil, nil, errors.New("PEM block not encrypted")
	}
	keyBytes, err := x509.DecryptPEMBlock(block, pass)
	if err != nil {
		return nil, nil, err
	}
	key, err := x509.ParsePKCS1PrivateKey(keyBytes)
	if err != nil {
		return nil, nil, err
	}
	return crt, key, nil
}

func (d *MySQLDepot) createCA(pass []byte, years int, cn, org, country string) (*x509.Certificate, *rsa.PrivateKey, error) {
	_, err := d.db.ExecContext(d.ctx, `INSERT IGNORE INTO serials (serial) VALUES (1);`)
	if err != nil {
		return nil, nil, err
	}
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	caCert := depot.NewCACert(
		depot.WithYears(years),
		depot.WithOrganization(org),
		depot.WithCommonName(cn),
		depot.WithCountry(country),
	)
	crtBytes, err := caCert.SelfSign(rand.Reader, &privKey.PublicKey, privKey)
	if err != nil {
		return nil, nil, err
	}
	crt, err := x509.ParseCertificate(crtBytes)
	if err != nil {
		return nil, nil, err
	}
	err = d.Put(crt.Subject.CommonName, crt)
	if err != nil {
		return nil, nil, err
	}
	encPemBlock, err := x509.EncryptPEMBlock(
		rand.Reader,
		"RSA PRIVATE KEY",
		x509.MarshalPKCS1PrivateKey(privKey),
		pass,
		x509.PEMCipher3DES,
	)
	if err != nil {
		return nil, nil, err
	}
	_, err = d.db.ExecContext(
		d.ctx,
		`
INSERT INTO ca_keys
    (serial, key_pem)
VALUES
    (?, ?);`,
		crt.SerialNumber.Int64(),
		pem.EncodeToMemory(encPemBlock),
	)
	if err != nil {
		return nil, nil, err
	}
	d.crt = crt
	d.key = privKey
	return d.crt, d.key, nil
}

func (d *MySQLDepot) CreateOrLoadCA(pass []byte, years int, cn, org, country string) (*x509.Certificate, *rsa.PrivateKey, error) {
	var err error
	d.crt, d.key, err = d.loadCA(pass)
	if err != nil {
		return nil, nil, err
	}
	if d.crt != nil && d.key != nil {
		return d.crt, d.key, nil
	}
	return d.createCA(pass, years, cn, org, country)
}

func (d *MySQLDepot) CA(pass []byte) ([]*x509.Certificate, *rsa.PrivateKey, error) {
	if d.crt == nil || d.key == nil {
		return nil, nil, errors.New("CA crt or key is empty")
	}
	return []*x509.Certificate{d.crt}, d.key, nil
}

func (d *MySQLDepot) Put(name string, crt *x509.Certificate) error {
	if crt.Subject.CommonName == "" {
		// this means our cn was replaced by the certificate Signature
		// which is inappropriate for a filename
		name = fmt.Sprintf("%x", sha256.Sum256(crt.Raw))
	}
	if !crt.SerialNumber.IsInt64() {
		return errors.New("cannot represent serial number as int64")
	}
	block := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: crt.Raw,
	}
	_, err := d.db.ExecContext(
		d.ctx, `
INSERT INTO certificates
    (serial, name, not_valid_before, not_valid_after, certificate_pem)
VALUES
    (?, ?, ?, ?, ?);`,
		crt.SerialNumber.Int64(),
		name,
		crt.NotBefore,
		crt.NotAfter,
		pem.EncodeToMemory(block),
	)
	return err
}

func (d *MySQLDepot) Serial() (*big.Int, error) {
	result, err := d.db.ExecContext(d.ctx, `INSERT INTO serials () VALUES ();`)
	if err != nil {
		return nil, err
	}
	lid, err := result.LastInsertId()
	return big.NewInt(lid), err
}

func (d *MySQLDepot) HasCN(cn string, allowTime int, cert *x509.Certificate, revokeOldCertificate bool) (bool, error) {
	var ct int
	row := d.db.QueryRowContext(d.ctx, `SELECT COUNT(*) FROM certificates WHERE name = ?;`, cn)
	if err := row.Scan(&ct); err != nil {
		return false, err
	}
	return ct >= 1, nil
}

func (d *MySQLDepot) SCEPChallenge() (string, error) {
	key := make([]byte, 24)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}
	challenge := base64.StdEncoding.EncodeToString(key)
	_, err = d.db.ExecContext(d.ctx, `INSERT INTO challenges (challenge) VALUES (?);`, challenge)
	if err != nil {
		return "", err
	}
	return challenge, nil
}

func (d *MySQLDepot) HasChallenge(pw string) (bool, error) {
	result, err := d.db.ExecContext(d.ctx, `DELETE FROM challenges WHERE challenge = ?;`, pw)
	if err != nil {
		return false, err
	}
	rowCt, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if rowCt < 1 {
		return false, errors.New("challenge not found")
	}
	return true, nil
}
