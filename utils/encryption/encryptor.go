package encryption

import (
	"bytes"
	"context"
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
	encryptedTextBlockStart           = "##!!"
	encryptedTextBlockParamsDelimiter = ":"
	encryptedTextBlockCipherStart     = "["
	encryptedTextBlockCipherEnd       = "]"
)

// Formatter formats given string.
type Formatter func(string) string

// Encryptor provides RSA encryption interface for sensitive data.
type Encryptor struct {
	key        string
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

type CipherBloc struct {
	key        string
	ciphertext string
}

// NewFromPrivateKey create new Encryptor from private key encoded in PEM.
func NewFromPrivateKey(key string, privateKeyPEM []byte) (*Encryptor, error) {
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

	return &Encryptor{
		key:        key,
		privateKey: privateKeyTyped,
		publicKey:  publicKeyTyped,
	}, nil
}

// NewFromPublicKey create new Encryptor from public key encoded in PEM.
// You will be able to encrypt, calling decrypt will result in error.
func NewFromPublicKey(key string, publicKeyPEM []byte) (*Encryptor, error) {
	pemBlock, _ := pem.Decode(publicKeyPEM)
	if pemBlock == nil {
		return nil, errors.New("Cannot parse PEM")
	}

	publicKey, err := x509.ParsePKIXPublicKey(pemBlock.Bytes)
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot parse PEM")
	}
	publicKeyTyped := publicKey.(*rsa.PublicKey)

	return &Encryptor{
		key:        key,
		privateKey: nil,
		publicKey:  publicKeyTyped,
	}, nil
}

// GenerateKeys creates asymmetric keys.
func (s *Encryptor) GenerateKeys(privateKeyPath, publicKeyPath string, bits int, perm fs.FileMode) error {
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
func (s *Encryptor) Encrypt(msg []byte) ([]byte, error) {
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
func (s *Encryptor) EncryptAsBlock(msg string) (string, error) {
	encrypted, err := s.EncryptBase32([]byte(msg))
	if err != nil {
		return "", err
	}
	return s.EncodeCipherBlock(encrypted)
}

// EncryptBase32 encrypts sequence of bytes with private RSA key and encodes it with Base32.
func (s *Encryptor) EncryptBase32(msg []byte) ([]byte, error) {
	ciphertext, err := s.Encrypt(msg)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to encypt: %s \n", err)
	}

	buf := make([]byte, base32.StdEncoding.EncodedLen(len(ciphertext)))
	base32.StdEncoding.Encode(buf, ciphertext)
	return buf, nil
}

// Decrypt decrypts ciphertext to msg.
func (s *Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
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
func (s *Encryptor) DecryptBase32(ciphertextBase32 string) ([]byte, error) {
	ciphertext, err := base32.StdEncoding.DecodeString(ciphertextBase32)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to decode from Base32: %s \n", err)
	}
	return s.Decrypt(ciphertext)
}

// DecryptEmbedded decrypts embedded ciphertext in text.
func (s *Encryptor) DecryptEmbedded(text string) (string, error) {
	return s.DecryptEmbeddedWithFormat(text, func(str string) string {
		return str
	})
}

// DecryptEmbeddedWithFormat decrypts embedded ciphertext in text.
func (s *Encryptor) DecryptEmbeddedWithFormat(text string, formatter Formatter) (string, error) {
	result := strings.Builder{}
	for splitIndex := strings.Index(text, encryptedTextBlockStart); splitIndex >= 0; splitIndex = strings.Index(text, encryptedTextBlockStart) {
		before := text[:splitIndex]
		result.WriteString(before)

		text = text[splitIndex:]
		cipherBlockStart := strings.Index(text, encryptedTextBlockStart)
		cipherBlockEnd := strings.Index(text, encryptedTextBlockCipherEnd)
		block, err := s.DecodeCipherBlock(text[cipherBlockStart : cipherBlockEnd+len(encryptedTextBlockCipherEnd)])
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
		text = text[cipherBlockEnd+len(encryptedTextBlockCipherEnd):]

		result.WriteString(formatter(string(decryptBase32)))
	}
	result.WriteString(text)

	return result.String(), nil
}

// EncodeCipherBlock encodes ciphertext into a Base32 block with metadata.
func (s *Encryptor) EncodeCipherBlock(text []byte) (string, error) {
	if bytes.Index(text, []byte(encryptedTextBlockStart)) != -1 {
		return "", errors.Errorf("Text cannot include %s", encryptedTextBlockStart)
	}

	result := strings.Builder{}
	result.WriteString(encryptedTextBlockStart)
	result.WriteString(encryptedTextBlockParamsDelimiter)
	result.WriteString(s.key)
	result.WriteString(encryptedTextBlockCipherStart)
	result.Write(text)
	result.WriteString(encryptedTextBlockCipherEnd)
	return result.String(), nil
}

// DecodeCipherBlock decodes ciphertext into a Base32 block with metadata.
func (s *Encryptor) DecodeCipherBlock(text string) (CipherBloc, error) {
	if strings.Index(text, encryptedTextBlockStart) != 0 {
		return CipherBloc{}, errors.Errorf("Cipher block must start with %s", encryptedTextBlockStart)
	}

	firstMetaIndex := strings.Index(text, encryptedTextBlockParamsDelimiter)

	// expect only one meta
	text = text[firstMetaIndex+len(encryptedTextBlockParamsDelimiter):]
	nextMetaIndexStart := strings.Index(text, encryptedTextBlockParamsDelimiter)
	if nextMetaIndexStart != -1 {
		return CipherBloc{}, errors.New("Not supported encoding format: expected one meta param")
	}

	// extract key
	bodyStart := strings.Index(text, encryptedTextBlockCipherStart)
	key := text[:bodyStart]

	// extract ciphertext
	bodyEnd := strings.Index(text, encryptedTextBlockCipherEnd)
	ciphertext := text[bodyStart+len(encryptedTextBlockParamsDelimiter) : bodyEnd]

	return CipherBloc{
		key:        key,
		ciphertext: ciphertext,
	}, nil
}

func (s *Encryptor) DecryptDSN(dsn string) (string, error) {
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

const EncryptorKey = "encryptor"

func GetEncryptor(ctx context.Context) *Encryptor {
	return ctx.Value(EncryptorKey).(*Encryptor)
}

func InjectEncryptorIfNotPresent(ctx context.Context, key []byte, keyID string) (context.Context, error) {
	encryptor := ctx.Value(EncryptorKey)
	if encryptor == nil {
		encryptor, err := NewFromPublicKey(keyID, key)
		if err != nil {
			return nil, err
		}
		return context.WithValue(ctx, EncryptorKey, encryptor), nil
	}

	return ctx, nil
}
