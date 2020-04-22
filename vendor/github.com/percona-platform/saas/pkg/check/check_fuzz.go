// +build gofuzz

package check

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
)

const dataG = `random data`

const publicKeyG = `RWRQmBOLeYzAeuR2L6L1GJN9qTR8ceQrawtijPTQkVbf3LJsrLeUjQcL`

const signG = `untrusted comment: signature from minisign secret key
RWRQmBOLeYzAetS6fGVWAvzwCgDuo/zNlvdOrClAvjCUSMLnUimp6NQd1L+x77HZa0kEB7ei+K9lW+W4hIf1D8gRNm+cdQr7dgk=
trusted comment: timestamp:1586854934	file:data
WXAxVyC6G82QuXtGlJZzLWoVmw8QNWks2T6RfXo8F9oKjI+sPbBf0ZOBWD2hXKFBCo5pKPSJiaVeI4G36OlEAw==
`

func fuzz(data []byte, publicKey, sign string) int {
	minisignErr := VerifyWithMinisignBin(data, publicKey, sign)

	err := Verify(data, publicKey, sign)

	if minisignErr == nil && err == nil {
		return 1
	}

	if minisignErr != nil && err != nil {
		return 0
	}

	log.Printf("verifyErr = %v", err)
	log.Printf("minisignErr = %v", minisignErr)
	panic("implementations differ")
}

func FuzzPublicKey(publicKey []byte) int {
	return fuzz(publicKey, dataG, signG)
}

func FuzzData(data []byte) int {
	return fuzz(data, publicKeyG, signG)
}

func FuzzSign(sign []byte) int {
	return fuzz([]byte(dataG), publicKeyG, string(sign))
}

func VerifyWithMinisignBin(data []byte, key, sign string) error {
	dataFile, err := writeTempFile("minisign-data-*", data)
	if err != nil {
		return err
	}
	defer os.Remove(dataFile)

	signFile, err := writeTempFile("minisign-sign-*", []byte(sign))
	if err != nil {
		return err
	}
	defer os.Remove(signFile)

	cmd := exec.Command("minisign", "-V", "-P", key, "-m", dataFile, "-x", signFile)
	return cmd.Run()
}

func writeTempFile(pattern string, b []byte) (string, error) {
	f, err := ioutil.TempFile("", pattern)
	if err != nil {
		return "", err
	}

	if _, err = f.Write(b); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}

	return f.Name(), f.Close()
}
