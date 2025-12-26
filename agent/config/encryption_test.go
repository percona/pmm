// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const rsaKey = `
-----BEGIN PRIVATE KEY-----
MIICdgIBADANBgkqhkiG9w0BAQEFAASCAmAwggJcAgEAAoGBAL7XQZoBKba6RYXp
WsmX8kaeX4OBsNyqUrho3eh85jdEpfAYwzlKWNy3aboDre85ugwMEt5yMFOXL96r
9KnFrp4KpEacfJJdcoAPncAIslb0anwhhXOoc9ZIgz7AyJMKUN7PQSC4VpSP/86w
CShq5cEyRPbs4CwnQ5yYtlqCWQi1AgMBAAECgYA3EuXSrN0953mi0JorrVb0vEWy
LN4+gETJBTJtIoZJkt0UcgD86pDEeYXgcaljbVRcn6teWLPLm8jryNIdoHfoknIB
crf6vemmlP80Lpw2cdg46Q9lcleGTwJGOd+R2QSWJLV7kPrhhR+wIw6m7TDHvhGU
yx9AY4GrAMSrf4wTbQJBAN+3KuNssRBRZXKBDMuCAwgr8hSTNmjivRGmmATv3MGF
cCey/PbvQQ1jPRViSUysmFumOTE59GlcmB6TQQeUXF8CQQDaYZWIuGuAenI4pz+w
BR8JyJs5N52/YmsxTe2XShqksiiyvaHxNJ1mvSVEorPqEwfMwwFLI+cq0rh7YwZd
QLNrAkAV5QlPhL23iR/SmwqziB/f1t00Ykv66+XxKkrKgOcsEXEukXfsevH063d4
9kuSM3odziDezns7LJK+u06r/TslAkEAjowiStNuwLestVRe2ywMnZtHz2qBWvsI
U2+1xgqGJ7lvnXTxL3yTvgt7NzkpTYLMlZk4z+6Ip8hSyZ/S+K4SLwJAPO8sd5U8
mfvGLl15b1BI4o86X4r+HC4Nwb33i4eVBM06YBYbxSKUu9E9lWYiU1wCgPA7+T75
TF1yxL9OKxrMpQ==
-----END PRIVATE KEY-----
`

const rsaPasswordKey = `
-----BEGIN ENCRYPTED PRIVATE KEY-----
MIIC5TBfBgkqhkiG9w0BBQ0wUjAxBgkqhkiG9w0BBQwwJAQQ1vE1Ke5aqPfS+DEA
U0f79QICCAAwDAYIKoZIhvcNAgkFADAdBglghkgBZQMEASoEEKoYunBmlc2MEbT2
aaTRrNcEggKAKAmW5WL80flLmh+llCu4claeq5PT3DfrhUvL0UkdXDVV70aIfOMr
C7d28usIcfbguEtyPA72oWraNpot0u8z6SxVEZ0lFz4tk4eG8II1vLuzSXsswM0q
JKBGwbkwPVFIq21pK1vmfvWpA6gTTd7QU4YKInq1eXuoxHkdDC37ryXEUx/tEd+s
Gd54Zs1QXh2k0jSzTaOLlUaez9ADT1Nk9pS9Fj2aUSns7xXVXKYMYxpEYkUfd7xK
mowC0q0L/av9Xoj+EI5H0f9CdbufMpCe9GPAPShEEkV2feZsuigneAAYiIyAOMUh
T954t9M/rsSN5jAAZ/syaC6bDblx+nL6hDUcQjLyUJah7GRGwt5xexgMtfZccS8n
dp0DNG1gJjhK1QaP0BNBslMGlrbEoIkMn3MbmKVwnmKK/AM9kqalJV45RAogXpTG
lMhJ2bTpgx0CfIzalVEWc3eo7E0tf+Q//OHOptCx/u/sjeO5ZScyYt4SCiRsPYbJ
MjRQd3w3rQj5dW8f7ewDOf1xjueJ6q9sNL0f1OXgmCwq6MUbA2EpOMorETuZaATy
pLhDt8QNTp75NfENGbR+Uk+CoezmxS6KKGEG6P7SAvXnBHY4TGPNQq+pxaWxu6PB
+AS+K3t64XElJT99IB6rGjjpODQRQhk3IiX7hsJpG3KsTi2/sG/ZUkHJ5p7SDiK8
/q7EECYgQoqkQ4CFXbF1+Nz0fVIkX3FfRgJtKeuxNnjsY/CXnb1PERendg+6U0sp
e60uBuuy8TnXdkC3MIZyuc0iuZkw+XXdgZctv3bcAmkMDIcm0wNX5248N6tWcBUb
9j2emDUrENFV+8s8F+Mkj0mpoB+lre7Tww==
-----END ENCRYPTED PRIVATE KEY-----
`

func writeKey(t *testing.T, keyname, key string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), keyname)
	require.NoError(t, os.WriteFile(path, []byte(key), 0o600))
	return path
}

func TestEncryption(t *testing.T) {
	t.Run("Encrypted", func(t *testing.T) {
		key := writeKey(t, "key", rsaKey)
		enc := Encryption{
			KeyFile: key,
		}
		configfilef := writeConfig(t, &Config{ID: "agent-id", Encryption: enc})
		cfg, err := loadFromFile(configfilef, &enc)
		require.NoError(t, err)
		assert.Equal(t, &Config{ID: "agent-id"}, cfg)
	})

	t.Run("EncryptedPassword", func(t *testing.T) {
		key := writeKey(t, "key", rsaPasswordKey)
		enc := Encryption{
			KeyFile:         key,
			KeyFilePassword: "abcdefgh",
		}
		configfilef := writeConfig(t, &Config{ID: "agent-id", Encryption: enc})
		cfg, err := loadFromFile(configfilef, &enc)
		require.NoError(t, err)
		assert.Equal(t, &Config{ID: "agent-id"}, cfg)
	})

	t.Run("EncryptedWrongPassword", func(t *testing.T) {
		key := writeKey(t, "key", rsaPasswordKey)
		configfilef := writeConfig(t, &Config{ID: "agent-id", Encryption: Encryption{
			KeyFile:         key,
			KeyFilePassword: "abcdefgh",
		}})

		cfg, err := loadFromFile(configfilef, &Encryption{
			KeyFile:         key,
			KeyFilePassword: "hgfedcba",
		})
		require.EqualError(t, err, "unable to get RSA key from KeyFile: unable to parse private key: pkcs8: incorrect password")
		assert.Nil(t, cfg)
	})

	t.Run("EncryptedWrongKey", func(t *testing.T) {
		key1 := writeKey(t, "key1", rsaPasswordKey)
		key2 := writeKey(t, "key2", rsaKey)

		configfilef := writeConfig(t, &Config{ID: "agent-id", Encryption: Encryption{
			KeyFile: key2,
		}})
		cfg, err := loadFromFile(configfilef, &Encryption{
			KeyFile:         key1,
			KeyFilePassword: "abcdefgh",
		})
		require.EqualError(t, err, "unable to RSA-unwrap AES key: crypto/rsa: decryption error")
		assert.Nil(t, cfg)
	})
}
