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

func GetCloudCredentialsFromResponse(response *http.Response) (*TokenResponse, error) {
	unscopedToken := response.Header.Get(xheaders.XSubjectToken)
	if unscopedToken == "" {
		bodyBytes, err := GetBodyBytesFromResponse(response)
		if err != nil {
			return nil, err
		}
		responseString := string(bodyBytes)
		if strings.Contains(responseString, "mfa totp code verify fail") {
			return nil, errors.New(
				"fatal: invalid otp unscopedToken.\n" +
					"\nPlease try it again with a new otp unscopedToken")
		}
		formattedError, fmtErr := ByteSliceToIndentedJSONFormat(bodyBytes)
		if fmtErr != nil {
			return nil, fmt.Errorf("couldn't format error message: %w", fmtErr) // the irony
		}
		return nil, fmt.Errorf(
			"fatal: response failed with status %s. Body:\n%s",
			response.Status, formattedError)
	}

	bodyBytes, err := GetBodyBytesFromResponse(response)
	if err != nil {
		return nil, err
	}
	tokenResponse, err := DeserializeJSONForType[TokenResponse](bodyBytes)
	if err != nil {
		return nil, err
	}
	tokenResponse.Token.Secret = unscopedToken

	return tokenResponse, nil
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
