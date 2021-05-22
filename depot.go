package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"errors"
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

var ErrNotImplemented = errors.New("not implemented")

func (d *MySQLDepot) CreateOrLoadCA(pass []byte, years int, cn, org, country string) (*x509.Certificate, *rsa.PrivateKey, error) {
	_, err := d.db.ExecContext(d.ctx, `INSERT IGNORE INTO serials (serial) VALUES (1)`)
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
    (?, ?)`,
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

func (d *MySQLDepot) CA(pass []byte) ([]*x509.Certificate, *rsa.PrivateKey, error) {
	return []*x509.Certificate{d.crt}, d.key, nil
}

func (d *MySQLDepot) Put(name string, crt *x509.Certificate) error {
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
    (?, ?, ?, ?, ?)`,
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
	row := d.db.QueryRowContext(d.ctx, `SELECT COUNT(*) FROM certificates WHERE name = ?`, cn)
	if err := row.Scan(&ct); err != nil {
		return false, err
	}
	return ct >= 1, nil
}
