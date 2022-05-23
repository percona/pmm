package rsa_encryptor

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	_ "embed"
	"encoding/base32"
	"encoding/pem"
	"io/fs"
	"net/url"
	"os"
	"strings"

	"github.com/pkg/errors"
)

const (
	EncryptedTextBlockStart           = "##!!"
	EncryptedTextBlockParamsDelimiter = ":"
	EncryptedTextBlockCipherStart     = "["
	EncryptedTextBlockCipherEnd       = "]"
)

// Formatter formats given string.
type Formatter func(string) string

// Service provides RSA encryption interface for sensitive data.
type Service struct {
	key        string
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

type CipherBloc struct {
	key        string
	ciphertext string
}

// NewFromPrivateKey create new Service from private key encoded in PEM.
func NewFromPrivateKey(key string, privateKeyPEM []byte) (*Service, error) {
	pemBlock, _ := pem.Decode(privateKeyPEM)
	if pemBlock == nil {
		return nil, errors.New("Cannot parse PEM")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(pemBlock.Bytes)
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot parse PEM")
	}
	privateKeyTyped := privateKey.(*rsa.PrivateKey)
	publicKeyTyped := privateKeyTyped.Public().(*rsa.PublicKey)

	return &Service{
		key:        key,
		privateKey: privateKeyTyped,
		publicKey:  publicKeyTyped,
	}, nil
}

// NewFromPublicKey create new Service from public key encoded in PEM.
// You will be able to encrypt, calling decrypt will result in error.
func NewFromPublicKey(key string, publicKeyPEM []byte) (*Service, error) {
	pemBlock, _ := pem.Decode(publicKeyPEM)
	if pemBlock == nil {
		return nil, errors.New("Cannot parse PEM")
	}

	publicKey, err := x509.ParsePKIXPublicKey(pemBlock.Bytes)
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot parse PEM")
	}
	publicKeyTyped := publicKey.(*rsa.PublicKey)

	return &Service{
		key:        key,
		privateKey: nil,
		publicKey:  publicKeyTyped,
	}, nil
}

// GenerateKeys creates asymmetric keys.
func (s *Service) GenerateKeys(privateKeyPath, publicKeyPath string, bits int, perm fs.FileMode) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return errors.Wrap(err, "Cannot generate ed25519 key")
	}
	publicKey := privateKey.Public()

	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return errors.Wrap(err, "Cannot marshal to PKCS8 key")
	}
	privateKeyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}
	privateKeyPem, err := os.Create(privateKeyPath)
	if err != nil {
		return errors.Wrapf(err, "Error when create [%s]: %s \n", privateKeyPath, err)
	}
	err = privateKeyPem.Chmod(perm)
	if err != nil {
		return errors.Wrapf(err, "Error when changing file permissions [%s]: %s \n", privateKeyPath, err)
	}
	err = pem.Encode(privateKeyPem, privateKeyBlock)
	if err != nil {
		return errors.Wrapf(err, "Error when encode private pem: %s \n", err)
	}

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return errors.Wrapf(err, "Error when dumping publickey: %s \n", err)
	}
	publicKeyBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	publicPem, err := os.Create(publicKeyPath)
	if err != nil {
		return errors.Wrapf(err, "error when create [%s]: %s \n", publicKeyPath, err)
	}
	err = publicPem.Chmod(perm)
	if err != nil {
		return errors.Wrapf(err, "error when changing file permissions [%s]: %s \n", publicKeyPath, err)
	}
	err = pem.Encode(publicPem, publicKeyBlock)
	if err != nil {
		return errors.Wrapf(err, "Error when encode public pem: %s \n", err)
	}

	return nil
}

// Encrypt encrypts sequence of bytes with private RSA key.
func (s *Service) Encrypt(msg []byte) ([]byte, error) {
	if s.publicKey == nil {
		return nil, errors.New("PublicKey must be set to encrypt")
	}
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, s.publicKey, msg, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to encypt with OAEP: %s \n", err)
	}
	return ciphertext, nil
}

// EncryptAsBlock encrypts sequence of bytes with private RSA key and encodes it with Base32.
//				  After, wraps into the CipherBlock.
func (s *Service) EncryptAsBlock(msg string) (string, error) {
	encrypted, err := s.EncryptBase32([]byte(msg))
	if err != nil {
		return "", err
	}
	return s.EncodeCipherBlock(encrypted)
}

// EncryptBase32 encrypts sequence of bytes with private RSA key and encodes it with Base32.
func (s *Service) EncryptBase32(msg []byte) ([]byte, error) {
	ciphertext, err := s.Encrypt(msg)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to encypt: %s \n", err)
	}

	buf := make([]byte, base32.StdEncoding.EncodedLen(len(ciphertext)))
	base32.StdEncoding.Encode(buf, ciphertext)
	return buf, nil
}

// Decrypt decrypts ciphertext to msg.
func (s *Service) Decrypt(ciphertext []byte) ([]byte, error) {
	if s.privateKey == nil {
		return nil, errors.New("PrivateKey must be set to decrypt")
	}

	plaintext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, s.privateKey, ciphertext, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to decrypt with OAEP: %s \n", err)
	}
	return plaintext, nil
}

// DecryptBase32 decrypts ciphertext encoded in Base32 to msg.
func (s *Service) DecryptBase32(ciphertextBase32 string) ([]byte, error) {
	ciphertext, err := base32.StdEncoding.DecodeString(ciphertextBase32)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to decode from Base32: %s \n", err)
	}
	return s.Decrypt(ciphertext)
}

// DecryptEmbedded decrypts embedded ciphertext in text.
func (s *Service) DecryptEmbedded(text string) (string, error) {
	return s.DecryptEmbeddedWithFormat(text, func(str string) string {
		return str
	})
}

// DecryptEmbeddedWithFormat decrypts embedded ciphertext in text.
func (s *Service) DecryptEmbeddedWithFormat(text string, formatter Formatter) (string, error) {
	result := strings.Builder{}
	for splitIndex := strings.Index(text, EncryptedTextBlockStart); splitIndex >= 0; splitIndex = strings.Index(text, EncryptedTextBlockStart) {
		before := text[:splitIndex]
		result.WriteString(before)

		text = text[splitIndex:]
		cipherBlockStart := strings.Index(text, EncryptedTextBlockStart)
		cipherBlockEnd := strings.Index(text, EncryptedTextBlockCipherEnd)
		block, err := s.DecodeCipherBlock(text[cipherBlockStart : cipherBlockEnd+len(EncryptedTextBlockCipherEnd)])
		if err != nil {
			return "", errors.Wrapf(err, "Failed to decode Cipher block: %s \n", err)
		}
		if s.key != block.key {
			return "", errors.Wrapf(err, "Block is encrypted with [%s] key, expected [%s]\n", block.key, s.key)
		}

		decryptBase32, err := s.DecryptBase32(block.ciphertext)
		if err != nil {
			return "", errors.Wrapf(err, "Failed to decrypt: %s \n", err)
		}
		text = text[cipherBlockEnd+len(EncryptedTextBlockCipherEnd):]

		result.WriteString(formatter(string(decryptBase32)))
	}
	result.WriteString(text)

	return result.String(), nil
}

// EncodeCipherBlock encodes ciphertext into a Base32 block with metadata.
func (s *Service) EncodeCipherBlock(text []byte) (string, error) {
	if bytes.Index(text, []byte(EncryptedTextBlockStart)) != -1 {
		return "", errors.Errorf("Text cannot include %s", EncryptedTextBlockStart)
	}

	result := strings.Builder{}
	result.WriteString(EncryptedTextBlockStart)
	result.WriteString(EncryptedTextBlockParamsDelimiter)
	result.WriteString(s.key)
	result.WriteString(EncryptedTextBlockCipherStart)
	result.Write(text)
	result.WriteString(EncryptedTextBlockCipherEnd)
	return result.String(), nil
}

// DecodeCipherBlock decodes ciphertext into a Base32 block with metadata.
func (s *Service) DecodeCipherBlock(text string) (CipherBloc, error) {
	if strings.Index(text, EncryptedTextBlockStart) != 0 {
		return CipherBloc{}, errors.Errorf("Cipher block must start with %s", EncryptedTextBlockStart)
	}

	firstMetaIndex := strings.Index(text, EncryptedTextBlockParamsDelimiter)

	// expect only one meta
	text = text[firstMetaIndex+len(EncryptedTextBlockParamsDelimiter):]
	nextMetaIndexStart := strings.Index(text, EncryptedTextBlockParamsDelimiter)
	if nextMetaIndexStart != -1 {
		return CipherBloc{}, errors.New("Not supported encoding format: expected one meta param")
	}

	// extract key
	bodyStart := strings.Index(text, EncryptedTextBlockCipherStart)
	key := text[:bodyStart]

	// extract ciphertext
	bodyEnd := strings.Index(text, EncryptedTextBlockCipherEnd)
	ciphertext := text[bodyStart+len(EncryptedTextBlockParamsDelimiter) : bodyEnd]

	return CipherBloc{
		key:        key,
		ciphertext: ciphertext,
	}, nil
}

func (s *Service) DecryptDsn(dsn string) (string, error) {
	parsedUrl, err := url.Parse(dsn)
	if err != nil {
		return "", errors.Wrap(err, "cannot parse DSN")
	}
	password, hasPassword := parsedUrl.User.Password()
	if hasPassword {
		decrypted, err := s.DecryptEmbedded(password)
		if err != nil {
			return "", errors.Wrap(err, "cannot decrypt password")
		}

		parsedUrl.User = url.UserPassword(parsedUrl.User.Username(), decrypted)
		return parsedUrl.String(), nil
	}

	return dsn, nil
}
