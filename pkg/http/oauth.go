package http

import (
	"io"
	"net/http"

	"github.com/hashicorp/go-retryablehttp"
	"k8s.io/apimachinery/pkg/util/json"
)

type tokenJSON struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int32  `json:"expires_in"`
}

func GetAccessToken(retryClient *retryablehttp.Client, tokenUrl, clientId, clientSecret, audience string) (string, error) {

	requestBody := map[string]string{
		"client_id":     clientId,
		"client_secret": clientSecret,
		"grant_type":    "client_credentials",
		"audience":      audience,
	}
	requestString, _ := json.Marshal(requestBody)

	response, err := retryClient.Post(tokenUrl, "application/json", requestString)
	if err != nil {
		return "", err
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	var token tokenJSON
	err = json.Unmarshal(responseBody, &token)
	if err != nil {
		return "", err
	}
	return token.AccessToken, nil
}

func SendRequestWithToken(retryClient *retryablehttp.Client, url, token string, body []byte) (*http.Response, error) {
	request, err := retryablehttp.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	return retryClient.Do(request)
}
