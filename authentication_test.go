package gosn

import (
	"testing"
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
	if err != nil {
		t.Errorf("SignIn Failed - err returned: %v", err)
	}
	if sOutput.Session.Token == "" || sOutput.Session.Mk == "" || sOutput.Session.Ak == "" {
		t.Errorf("SignIn Failed - token: %s mk: %s ak: %s",
			sOutput.Session.Token, sOutput.Session.Mk, sOutput.Session.Ak)
	}
}
