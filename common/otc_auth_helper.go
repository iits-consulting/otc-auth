package common

import (
	"log"
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
			log.Fatalf(
				"fatal: invalid otp unscopedToken.\n" +
					"\nPlease try it again with a new otp unscopedToken")
		}
		formattedError := ByteSliceToIndentedJSONFormat(bodyBytes)
		log.Fatalf(
			"fatal: response failed with status %s. Body:\n%s",
			response.Status, formattedError)
	}

	bodyBytes := GetBodyBytesFromResponse(response)
	tokenResponse = *DeserializeJSONForType[TokenResponse](bodyBytes)
	tokenResponse.Token.Secret = unscopedToken

	return tokenResponse
}

func ParseTimeOrThrow(timeString string) time.Time {
	if timeString == "" {
		return time.Time{}
	}
	parsedTime, err := time.Parse(time.RFC3339, timeString)
	if err != nil {
		log.Fatalf("fatal: error parsing time from token %s", err)
	}

	return parsedTime
}
