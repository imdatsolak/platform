package auth

import (
	"cydb"
	"data"
	"fmt"
	"time"
	"token"
)

type AuthenticationRequest struct {
	ApplicationLogin       string `json:"application_login"`
	ApplicationSecret      string `json:"application_secret"`
	ApplicationInstanceUID string `json:"application_instance_uid"`
}

type AuthenticationResponse struct {
	Token      string    `json:"auth_token"`
	Expires    time.Time `json:"expires"`
	ServerTime time.Time `json:"server_time"`
}

var AuthTokenTTL time.Duration = time.Hour
var keyIn string = "0a15d45db2dc75f4ed7f2738fe662988eaa58e678e796328b72cc5115877b3ce"
var nonceIn string = "f2b4f2ed9eea1e93f9686334"

func decodeAuthToken(authToken string) (bool, int, int, time.Time) {
	var err error = nil
	var authTokenPlain string
	key, nonce, err := token.ValidateKeyAndNonce(keyIn, nonceIn)
	if err == nil {
		authTokenPlain, err = token.Decrypt(key, nonce, authToken)
	}
	if err == nil {
		var applicationId, applicationInstanceId int
		var expirationTimeStr string
		numConverted, err := fmt.Sscanf(authTokenPlain, "%d|%d|%s", &applicationId, &applicationInstanceId, &expirationTimeStr)
		data.Logger.Printf("authToken = %s, applicationId = %d, applicationInstanceId = %d, expirationTimeStr = %s", authToken, applicationId, applicationInstanceId, expirationTimeStr)
		if err == nil && numConverted == 3 {
			expirationTime, err := time.Parse(time.RFC3339, expirationTimeStr)
			data.Logger.Printf("expirationTime = %s", expirationTime)
			if err == nil {
				return true, applicationId, applicationInstanceId, expirationTime
			}
		}
	}
	return false, -1, -1, time.Now()
}

func encodeAuthToken(applicationId int, applicationInstanceId int) AuthenticationResponse {
	var authTokenCipher string
	expirationTime := time.Now().Add(AuthTokenTTL)
	expirationTimeStr := expirationTime.Format(time.RFC3339)
	authToken := fmt.Sprintf("%d|%d|%s", applicationId, applicationInstanceId, expirationTimeStr)
	key, nonce, err := token.ValidateKeyAndNonce(keyIn, nonceIn)
	if err == nil {
		authTokenCipher, err = token.Encrypt(key, nonce, authToken)
		if err == nil {
			return AuthenticationResponse{Token: authTokenCipher, Expires: expirationTime, ServerTime: time.Now()}
		}
	}
	return AuthenticationResponse{Token: "", Expires: time.Now(), ServerTime: time.Now()}
}

func DecodeAndCheckAuthToken(authToken string) (bool, int, int) {
	success, applicationId, applicationInstanceId, expirationTime := decodeAuthToken(authToken)
	if success && expirationTime.After(time.Now()) {
		return true, applicationId, applicationInstanceId
	}
	return false, -1, -1
}

func RenewAuthToken(authToken string) (bool, *AuthenticationResponse) {
	success, applicationId, applicationInstanceId, _ := decodeAuthToken(authToken)
	if success {
		authResponse := encodeAuthToken(applicationId, applicationInstanceId)
		return true, &authResponse
	}
	return false, nil
}

func Authenticate(authReq AuthenticationRequest) (bool, *AuthenticationResponse) {
	loginOk, applicationId := cydb.LoginApplication(authReq.ApplicationLogin, authReq.ApplicationSecret)
	if loginOk {
		registrationOk, applicationInstanceId := cydb.RegisterApplicationInstanceIfNeeded(applicationId, authReq.ApplicationInstanceUID)
		if registrationOk {
			authResponse := encodeAuthToken(applicationId, applicationInstanceId)
			return true, &authResponse

		}
	}
	return false, nil
}

func IsAuthenticated(authToken string) bool {
	success, _, _, expirationTime := decodeAuthToken(authToken)
	data.Logger.Printf("Time Now = %s", time.Now())
	if success && expirationTime.After(time.Now()) {
		return true
	}
	return false
}
