package gosn

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ### server not required for following tests
func TestGenerateEncryptedPasswordWithValidInput(t *testing.T) {
	var testInput generateEncryptedPasswordInput
	testInput.userPassword = "oWB7c&77Zahw8XK$AUy#"
	testInput.Identifier = "soba@lessknown.co.uk"
	testInput.PasswordNonce = "9e88fc67fb8b1efe92deeb98b5b6a801c78bdfae08eecb315f843f6badf60aef"
	testInput.PasswordCost = 110000
	testInput.Version = "003"
	testInput.PasswordSalt = ""
	result, _, _, _ := generateEncryptedPasswordAndKeys(testInput)
	if result != "1312fe421aa49a6444684b58cbd5a43a55638cd5bf77514c78d50c7f3ae9c4e7" {
		t.Errorf("failed password generation")
	}
}

func TestGenerateEncryptedPasswordWithInvalidPasswordCostForVersion003(t *testing.T) {
	var testInput generateEncryptedPasswordInput
	testInput.userPassword = "oWB7c&77Zahw8XK$AUy#"
	testInput.Identifier = "soba@lessknown.co.uk"
	testInput.PasswordNonce = "9e88fc67fb8b1efe92deeb98b5b6a801c78bdfae08eecb315f843f6badf60aef"
	testInput.PasswordCost = 99999
	testInput.Version = "003"
	testInput.PasswordSalt = ""
	_, _, _, err := generateEncryptedPasswordAndKeys(testInput)
	if err == nil {
		t.Errorf("version 003 requires password cost of at least 100000")
	}
}

// ### server required for following tests
func TestSignIn(t *testing.T) {

	sOutput, err := SignIn(sInput)
	assert.NoError(t, err, "sign-in failed", err)

	if sOutput.Session.Token == "" || sOutput.Session.Mk == "" || sOutput.Session.Ak == "" {
		t.Errorf("SignIn Failed - token: %s mk: %s ak: %s",
			sOutput.Session.Token, sOutput.Session.Mk, sOutput.Session.Ak)
	}
}

func TestRegistration(t *testing.T) {
	time.Now().Format("20060102150405")
	emailAddr := fmt.Sprintf("testuser-%s@example.com", time.Now().Format("20060102150405"))
	password := "secret"
	rInput := RegisterInput{
		Email:     emailAddr,
		Password:  password,
		APIServer: os.Getenv("SN_SERVER"),
	}
	_, err := rInput.Register()
	if err != nil {
		t.Errorf("registration failed: %+v\n", err)
	}

	sInput := SignInInput{
		APIServer: os.Getenv("SN_SERVER"),
		Email:     emailAddr,
		Password:  password,
	}
	_, err = SignIn(sInput)
	if err != nil {
		t.Errorf("SignIn Failed - err returned: %v", err)
	}

}
