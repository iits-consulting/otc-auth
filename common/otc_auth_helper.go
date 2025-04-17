package common

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"otc-auth/common/xheaders"
)

const PrintTimeFormat = time.RFC1123

func GetCloudCredentialsFromResponseOrThrow(response *http.Response) TokenResponse {
	var tokenResponse TokenResponse
	unscopedToken := response.Header.Get(xheaders.XSubjectToken)
	if unscopedToken == "" {
		bodyBytes := GetBodyBytesFromResponse(response)
		responseString := string(bodyBytes)
		if strings.Contains(responseString, "mfa totp code verify fail") {
			ThrowError(
				errors.New(
					"fatal: invalid otp unscopedToken.\n" +
						"\nPlease try it again with a new otp unscopedToken"))
		}
		formattedError := ByteSliceToIndentedJSONFormat(bodyBytes)
		ThrowError(
			fmt.Errorf(
				"fatal: response failed with status %s. Body:\n%s",
				response.Status, formattedError))
	}

	bodyBytes := GetBodyBytesFromResponse(response)
	tokenResponse = *DeserializeJSONForType[TokenResponse](bodyBytes)
	tokenResponse.Token.Secret = unscopedToken

	return tokenResponse
}

func ParseTime(timeString string) (*time.Time, error) {
	if timeString == "" {
		return &time.Time{}, nil
	}
	parsedTime, err := time.Parse(time.RFC3339, timeString)
	if err != nil {
		return nil, fmt.Errorf("fatal: error parsing time from token %w", err)
	}

	return &parsedTime, nil
}
