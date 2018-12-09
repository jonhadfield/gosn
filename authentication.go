package gosn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	mathrand "math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func init() {
	mathrand.Seed(time.Now().UnixNano())
}

type doAuthRequestOutput struct {
	authParamsOutput
	mfaKEY string
}

type authParamsInput struct {
	email         string
	password      string
	tokenName     string
	tokenValue    string
	authParamsURL string
}

type authParamsOutput struct {
	Identifier    string `json:"identifier"`
	PasswordSalt  string `json:"pw_salt"`
	PasswordCost  int64  `json:"pw_cost"`
	PasswordNonce string `json:"pw_nonce"`
	Version       string `json:"version"`
	TokenName     string
}

func requestToken(client *http.Client, input signInInput) (signInSuccess signInResponse, signInFailure errorResponse, err error) {
	var reqBodyBytes []byte
	var reqBody string
	if input.tokenName != "" {
		reqBody = `{"password":"` + input.encPassword + `","email":"` + input.email + `","` + input.tokenName + `":"` + input.tokenValue + `"}`
	} else {
		reqBody = `{"password":"` + input.encPassword + `","email":"` + input.email + `"}`

	}
	reqBodyBytes = []byte(reqBody)

	var signInURLReq *http.Request
	signInURLReq, err = http.NewRequest(http.MethodPost, input.signInURL, bytes.NewBuffer(reqBodyBytes))
	signInURLReq.Header.Set("Content-Type", "application/json")
	if err != nil {
		return
	}

	var signInResp *http.Response
	signInResp, err = client.Do(signInURLReq)
	if err != nil {
		return signInSuccess, signInFailure, err
	}
	defer signInResp.Body.Close()

	var signInRespBody []byte
	signInRespBody, err = getResponseBody(signInResp)
	if err != nil {
		return
	}
	// unmarshal success
	err = json.Unmarshal(signInRespBody, &signInSuccess)
	if err != nil {
		return
	}
	// unmarshal failure
	err = json.Unmarshal(signInRespBody, &signInFailure)
	if err != nil {
		return
	}
	return signInSuccess, signInFailure, err
}

func processDoAuthRequestResponse(response *http.Response) (output doAuthRequestOutput, errResp errorResponse, err error) {
	body, err := getResponseBody(response)
	switch response.StatusCode {
	case 200:
		err = json.Unmarshal(body, &output)
		if err != nil {
			return
		}
	case 404:
		// email address not recognised
	case 401:
		// need mfa token
		// unmarshal error response
		err = json.Unmarshal(body, &errResp)
		if err != nil {
			return
		}
	default:
		err = fmt.Errorf("unhandled: %+v", response)
		return
	}
	return
}

type errorResponse struct {
	Error struct {
		Tag     string `json:"tag"`
		Message string `json:"message"`
		Payload struct {
			MFAKey string `json:"mfa_key"`
		}
	}
}

// HTTP request bit
func doAuthParamsRequest(input authParamsInput) (output doAuthRequestOutput, err error) {
	// make initial params request without mfa token
	var url string
	var body io.Reader

	if input.tokenName == "" {
		// initial request
		url = input.authParamsURL + "?email=" + input.email
	} else {
		// request with mfa
		url = input.authParamsURL + "?email=" + input.email + "&" + input.tokenName + "=" + input.tokenValue
	}
	var req *http.Request
	req, err = http.NewRequest(http.MethodGet, url, body)
	if err != nil {
		return
	}
	var response *http.Response
	response, err = httpClient.Do(req)
	if err != nil {
		return
	}
	defer response.Body.Close()

	var requestOutput doAuthRequestOutput
	var errResp errorResponse
	requestOutput, errResp, err = processDoAuthRequestResponse(response)
	if err != nil {
		return
	}
	output.Identifier = requestOutput.Identifier
	output.Version = requestOutput.Version
	output.PasswordCost = requestOutput.PasswordCost
	output.PasswordNonce = requestOutput.PasswordNonce
	output.PasswordSalt = requestOutput.PasswordSalt
	output.mfaKEY = errResp.Error.Payload.MFAKey
	return
}

func getAuthParams(input authParamsInput) (output authParamsOutput, err error) {
	var authRequestOutput doAuthRequestOutput
	if input.tokenName == "" {
		authRequestOutput, err = doAuthParamsRequest(input)
		output.Identifier = authRequestOutput.Identifier
		output.PasswordCost = authRequestOutput.PasswordCost
		output.PasswordNonce = authRequestOutput.PasswordNonce
		output.Version = authRequestOutput.Version
		output.PasswordSalt = authRequestOutput.PasswordSalt
		output.TokenName = authRequestOutput.mfaKEY

		if authRequestOutput.mfaKEY != "" {
			err = fmt.Errorf("requestMFA")
			return
		}
		if err != nil {
			return
		}
	} else {
		output, err = getAuthParamsWithMFA(input)
		return
	}

	return
}

func getAuthParamsWithMFA(input authParamsInput) (output authParamsOutput, err error) {
	var authRequestOutput doAuthRequestOutput
	authRequestOutput, err = doAuthParamsRequest(input)

	output.Identifier = authRequestOutput.Identifier
	output.PasswordCost = authRequestOutput.PasswordCost
	output.PasswordNonce = authRequestOutput.PasswordNonce
	output.Version = authRequestOutput.Version
	output.PasswordSalt = authRequestOutput.PasswordSalt
	output.TokenName = input.tokenName
	return
}

type generateEncryptedPasswordInput struct {
	userPassword string
	authParamsOutput
}

type signInInput struct {
	email       string
	encPassword string
	tokenName   string
	tokenValue  string
	signInURL   string
}

type signInResponse struct {
	User struct {
		UUID  string `json:"uuid"`
		Email string `json:"email"`
	}
	Token string `json:"token"`
}

type registerResponse struct {
	User struct {
		UUID  string `json:"uuid"`
		Email string `json:"email"`
	}
	Token string `json:"token"`
}

// Session holds authentication and encryption parameters required
// to communicate with the API and process transferred data
type Session struct {
	Token  string
	Mk     string
	Ak     string
	Server string
}

type SignInInput struct {
	Email     string
	TokenName string
	TokenVal  string
	Password  string
	APIServer string
}

type SignInOutput struct {
	Session   Session
	TokenName string
}

// SignIn authenticates with the server using credentials and optional MFA
// in order to obtain the data required to interact with Standard Notes
func SignIn(input SignInInput) (output SignInOutput, err error) {
	// if we have token name then we already have auth params
	if input.APIServer == "" {
		input.APIServer = apiServer
	}
	getAuthParamsInput := authParamsInput{
		email:         input.Email,
		password:      input.Password,
		tokenValue:    input.TokenVal,
		tokenName:     input.TokenName,
		authParamsURL: input.APIServer + authParamsPath,
	}

	// request authentication parameters
	var getAuthParamsOutput authParamsOutput
	getAuthParamsOutput, err = getAuthParams(getAuthParamsInput)
	output.TokenName = getAuthParamsOutput.TokenName
	if err != nil {
		return
	}
	// generate encrypted password
	var encPassword string
	var genEncPasswordInput generateEncryptedPasswordInput
	genEncPasswordInput.userPassword = input.Password
	genEncPasswordInput.Identifier = input.Email
	genEncPasswordInput.TokenName = getAuthParamsOutput.TokenName
	genEncPasswordInput.PasswordCost = getAuthParamsOutput.PasswordCost
	genEncPasswordInput.PasswordSalt = getAuthParamsOutput.PasswordSalt
	genEncPasswordInput.PasswordNonce = getAuthParamsOutput.PasswordNonce
	genEncPasswordInput.Version = getAuthParamsOutput.Version

	var mk, ak string
	encPassword, mk, ak, err = generateEncryptedPasswordAndKeys(genEncPasswordInput)
	if err != nil {
		return
	}

	// request token
	var tokenResp signInResponse
	var requestTokenFailure errorResponse
	tokenResp, requestTokenFailure, err = requestToken(httpClient, signInInput{
		email:       input.Email,
		encPassword: encPassword,
		tokenName:   getAuthParamsOutput.TokenName,
		tokenValue:  input.TokenVal,
		signInURL:   input.APIServer + signInPath,
	})
	if requestTokenFailure.Error.Message != "" {
		err = fmt.Errorf(strings.ToLower(requestTokenFailure.Error.Message))
		return
	}
	output.Session.Mk = mk
	output.Session.Ak = ak
	output.Session.Token = tokenResp.Token
	output.Session.Server = input.APIServer
	return
}

type RegisterInput struct {
	Email     string
	Password  string
	APIServer string
}

func processDoRegisterRequestResponse(response *http.Response) (token string, err error) {
	var body []byte
	body, err = getResponseBody(response)
	if err != nil {
		return
	}
	switch response.StatusCode {
	case 200:
		var output registerResponse
		err = json.Unmarshal(body, &output)
		if err != nil {
			return
		}
		token = output.Token
	case 404:
		// email address not recognised
		var errResp errorResponse
		err = json.Unmarshal(body, &errResp)
		if err != nil {
			err = fmt.Errorf("email address not recognised")
			return
		}
	case 401:
		// unmarshal error response
		var errResp errorResponse
		err = json.Unmarshal(body, &errResp)
		if errResp.Error.Message != "" {
			err = fmt.Errorf("error: email is already registered")
			return
		}
	default:
		err = fmt.Errorf("unhandled: %+v", response)
		return
	}

	return
}

// Register creates a new user token
// Params: email, password, pw_cost, pw_nonce, version
func (input RegisterInput) Register() (token string, err error) {
	var pw, pwNonce string
	pw, pwNonce, err = generateInitialKeysAndAuthParamsForUser(input.Email, input.Password)
	var req *http.Request

	reqBody := `{"email":"` + input.Email + `","identifier":"` + input.Email + `","password":"` + pw + `","pw_cost":"` + strconv.Itoa(defaultPasswordCost) + `","pw_nonce":"` + pwNonce + `","version":"` + defaultSNVersion + `"}`
	reqBodyBytes := []byte(reqBody)

	req, err = http.NewRequest(http.MethodPost, input.APIServer+authRegisterPath, bytes.NewBuffer(reqBodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Host = input.APIServer

	if err != nil {
		return
	}
	var response *http.Response
	response, err = httpClient.Do(req)
	if err != nil {
		return
	}
	defer response.Body.Close()
	token, err = processDoRegisterRequestResponse(response)
	if err != nil {
		return
	}
	return
}

func generateInitialKeysAndAuthParamsForUser(email, password string) (pw, pwNonce string, err error) {

	var genInput generateEncryptedPasswordInput
	genInput.userPassword = password
	genInput.Version = defaultSNVersion
	genInput.Identifier = email
	genInput.PasswordCost = defaultPasswordCost
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, 65)
	for i := range b {
		b[i] = letterRunes[mathrand.Intn(len(letterRunes))]
	}

	genInput.PasswordNonce = string(b)
	pwNonce = string(b)
	pw, _, _, err = generateEncryptedPasswordAndKeys(genInput)

	return
}
