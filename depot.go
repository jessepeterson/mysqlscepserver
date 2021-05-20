package main

import (
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"errors"
	"math/big"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLDepot struct {
	db *sql.DB
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
	return &MySQLDepot{db: db}, nil
}

var ErrNotImplemented = errors.New("not implemented")

func (d *MySQLDepot) CreateOrLoadCA(pass []byte) (*x509.Certificate, *rsa.PrivateKey, error) {
	return nil, nil, ErrNotImplemented
}

func (d *MySQLDepot) CA(pass []byte) ([]*x509.Certificate, *rsa.PrivateKey, error) {
	return nil, nil, ErrNotImplemented
}
func (d *MySQLDepot) Put(name string, crt *x509.Certificate) error {
	return ErrNotImplemented
}
func (d *MySQLDepot) Serial() (*big.Int, error) {
	return nil, ErrNotImplemented
}
func (d *MySQLDepot) HasCN(cn string, allowTime int, cert *x509.Certificate, revokeOldCertificate bool) (bool, error) {
	return false, ErrNotImplemented
}
