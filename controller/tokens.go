package controller

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

// SetupTokens configures a simple bearer token authorization layer for the API.
//
// The primary goals of this config are:
//  1. Fault tolerance - the server can be restarted with the same
//     parameters without needing to re-issue existing tokens.
//  2. Supports minimal access logging - we pack the requestor id into the token,
//     so that it can be logged during subsequent requests.
//  3. Reasonably secure against attacks - GCM is generally secure, although if
//     a nonce is reused then it is straightforward to reverse engineer the
//     encryption key. If your threat model cannot handle this, you should
//     implement a real access layer on top of this API.
func (c *Controller) SetupTokens(passphrase, tokenName string) error {
	if passphrase == "" {
		return errors.New("invalid token passphrase")
	}
	if tokenName == "" {
		tokenName = "bdog_token"
	}

	c.tokenKey = argon2.IDKey([]byte(passphrase), []byte(tokenName), 1, 64*1024, 2, 32)

	block, err := aes.NewCipher(c.tokenKey)
	if err != nil {
		return err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	c.newToken = func(ident string) string {
		s := fmt.Sprintf("%s=%s", tokenName, ident)
		nonce := make([]byte, 12)
		if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
			panic(err.Error())
		}
		dst := aesgcm.Seal(nil, nonce, []byte(s), nil)
		return base64.URLEncoding.EncodeToString(append(nonce, dst...))
	}
	c.checkToken = func(tok string) (res bool, ident string) {
		defer func() {
			if recover() != nil {
				// decrypting panic'ed, invalid token
				res = false
				ident = ""
			}
		}()

		data, err := base64.URLEncoding.DecodeString(tok)
		if err != nil {
			return false, ""
		}
		if len(data) < len(tokenName)+13 {
			return false, ""
		}

		dst, err := aesgcm.Open(nil, data[:12], data[12:], nil)
		if err != nil {
			panic(err)
		}

		res = bytes.Equal([]byte(tokenName+"="), dst[:len(tokenName)+1])
		ident = string(dst[len(tokenName)+1:])
		return
	}

	return nil
}
