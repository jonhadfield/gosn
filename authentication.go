package gosn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	mathrand "math/rand"
	"net/http"
	"net/url"
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
	debug         bool
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
	if err != nil {
		return
	}

	signInURLReq.Header.Set("Content-Type", "application/json")
	signInURLReq.Header.Set("Connection", "keep-alive")

	var signInResp *http.Response

	start := time.Now()
	signInResp, err = httpClient.Do(signInURLReq)
	elapsed := time.Since(start)

	debugPrint(input.debug, fmt.Sprintf("requestToken | request took: %+v", elapsed))

	if err != nil {
		return signInSuccess, signInFailure, err
	}

	defer func() {
		if err := signInResp.Body.Close(); err != nil {
			fmt.Println("failed to close response:", err)
		}
	}()

	var signInRespBody []byte

	readStart := time.Now()
	signInRespBody, err = ioutil.ReadAll(signInResp.Body)
	debugPrint(input.debug, fmt.Sprintf("requestToken | response read took %+v", time.Since(readStart)))


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

func processDoAuthRequestResponse(response *http.Response, debug bool) (output doAuthRequestOutput, errResp errorResponse, err error) {
	var body []byte
	body, err = ioutil.ReadAll(response.Body)

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
	var reqURL string

	if input.tokenName == "" {
		// initial request
		reqURL = input.authParamsURL + "?email=" + input.email
	} else {
		// request with mfa
		reqURL = input.authParamsURL + "?email=" + input.email + "&" + input.tokenName + "=" + input.tokenValue
	}

	var req *http.Request

	req, err = http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return
	}

	var response *http.Response

	response, err = httpClient.Do(req)
	if err != nil {
		return
	}

	defer func() {
		if err := response.Body.Close(); err != nil {
			fmt.Println("failed to close response:", err)
		}
	}()

	var requestOutput doAuthRequestOutput

	var errResp errorResponse

	requestOutput, errResp, err = processDoAuthRequestResponse(response, input.debug)
	if err != nil {
		return
	}

	output.Identifier = requestOutput.Identifier
	output.Version = requestOutput.Version
	output.PasswordCost = requestOutput.PasswordCost
	output.PasswordNonce = requestOutput.PasswordNonce
	output.PasswordSalt = requestOutput.PasswordSalt
	output.mfaKEY = errResp.Error.Payload.MFAKey

	return output, err
}

func getAuthParams(input authParamsInput) (output authParamsOutput, err error) {
	var authRequestOutput doAuthRequestOutput
	// if token name not provided, then make request without
	authRequestOutput, err = doAuthParamsRequest(input)
	if err != nil {
		return
	}

	output.Identifier = authRequestOutput.Identifier
	output.PasswordCost = authRequestOutput.PasswordCost
	output.PasswordNonce = authRequestOutput.PasswordNonce
	output.Version = authRequestOutput.Version
	output.PasswordSalt = authRequestOutput.PasswordSalt
	output.TokenName = authRequestOutput.mfaKEY

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
	debug       bool
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
	Debug     bool
}

type SignInOutput struct {
	Session   Session
	TokenName string
}

func processConnectionFailure(i error, reqURL string) error {
	switch {
	case strings.Contains(i.Error(), "no such host"):
		urlBits, pErr := url.Parse(reqURL)
		if pErr != nil {
			break
		}

		return fmt.Errorf("failed to connect to %s as %s cannot be resolved", reqURL, urlBits.Hostname())
	case strings.Contains(i.Error(), "unsupported protocol scheme"):
		if len(reqURL) > 0 {
			return fmt.Errorf("protocol is missing from API server URL: %s", reqURL)
		}

		return fmt.Errorf("API Server URL is undefined")
	case strings.Contains(i.Error(), "i/o timeout"):
		return fmt.Errorf("failed to connect to %s within %d seconds", reqURL, connectionTimeout)
	case strings.Contains(i.Error(), "permission denied"):
		return fmt.Errorf("failed to connect to %s", reqURL)
	}

	return i
}

// SignIn authenticates with the server using credentials and optional MFA
// in order to obtain the data required to interact with Standard Notes
func SignIn(input SignInInput) (output SignInOutput, err error) {
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
	if err != nil {
		err = processConnectionFailure(err, getAuthParamsInput.authParamsURL)
		return
	}
	// if we received a token name then we need to request token value
	if getAuthParamsOutput.TokenName != "" {
		output.TokenName = getAuthParamsOutput.TokenName
		return
	}

	// generate encrypted password
	var encPassword string

	var genEncPasswordInput generateEncryptedPasswordInput

	genEncPasswordInput.userPassword = input.Password
	genEncPasswordInput.Identifier = input.Email
	genEncPasswordInput.TokenName = input.TokenName
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
		tokenName:   input.TokenName,
		tokenValue:  input.TokenVal,
		signInURL:   input.APIServer + signInPath,
	})

	if err != nil {
		return
	}

	if requestTokenFailure.Error.Message != "" {
		err = fmt.Errorf(strings.ToLower(requestTokenFailure.Error.Message))
		return
	}

	output.Session.Mk = mk
	output.Session.Ak = ak
	output.Session.Token = tokenResp.Token
	output.Session.Server = input.APIServer

	return output, err
}

type RegisterInput struct {
	Email     string
	Password  string
	APIServer string
	Debug     bool
}

func processDoRegisterRequestResponse(response *http.Response, debug bool) (token string, err error) {
	var body []byte

	body, err = ioutil.ReadAll(response.Body)
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
			err = fmt.Errorf("email is already registered")
			return
		}
	default:
		err = fmt.Errorf("unhandled: %+v", response)
		return
	}

	return token, err
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
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "keep-alive")

	req.Host = input.APIServer

	var response *http.Response

	response, err = httpClient.Do(req)
	if err != nil {
		return
	}

	defer func() {
		if err := response.Body.Close(); err != nil {
			fmt.Println("failed to close response:", err)
		}
	}()

	token, err = processDoRegisterRequestResponse(response, input.Debug)
	if err != nil {
		return
	}

	return token, err
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

// CLiSignIn takes the server URL and credentials and sends them to the API to get a response including
// an authentication token plus the keys required to encrypt and decrypt SN items
func CliSignIn(email, password, apiServer string) (session Session, err error) {
	sInput := SignInInput{
		Email:     email,
		Password:  password,
		APIServer: apiServer,
	}

	// attempt sign-in without MFA
	var sioNoMFA SignInOutput

	sioNoMFA, err = SignIn(sInput)
	if err != nil {
		return
	}
	// return session if auth and master key returned
	if sioNoMFA.Session.Ak != "" && sioNoMFA.Session.Mk != "" {
		return sioNoMFA.Session, err
	}

	if sioNoMFA.TokenName != "" {
		// MFA token value required, so request
		var tokenValue string

		fmt.Print("token: ")

		_, err = fmt.Scanln(&tokenValue)
		if err != nil {
			return
		}
		// TODO: handle missing TokenName and Session
		// add token name and value to sign-in input
		sInput.TokenName = sioNoMFA.TokenName
		sInput.TokenVal = strings.TrimSpace(tokenValue)

		sOutTwo, sErrTwo := SignIn(sInput)
		if sErrTwo != nil {
			return session, sErrTwo
		}

		session = sOutTwo.Session
	}

	return session, err
}

func (s *Session) Valid() bool {
	switch {
	case s.Ak == "":
		return false
	case s.Mk == "":
		return false
	case s.Token == "":
		return false
	case s.Server == "":
		return false
	}

	return true
}
