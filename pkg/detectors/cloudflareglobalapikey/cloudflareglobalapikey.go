package cloudflareglobalapikey

import (
	"context"
	"net/http"
	"strings"

	regexp "github.com/wasilibs/go-re2"

	"github.com/trufflesecurity/trufflehog/v3/pkg/common"
	"github.com/trufflesecurity/trufflehog/v3/pkg/detectors"
	"github.com/trufflesecurity/trufflehog/v3/pkg/pb/detectorspb"
)

type Scanner struct{
	detectors.DefaultMultiPartCredentialProvider
}

// Ensure the Scanner satisfies the interface at compile time.
var _ detectors.Detector = (*Scanner)(nil)

var (
	client = common.SaneHttpClient()

	apiKeyPat = regexp.MustCompile(detectors.PrefixRegex([]string{"cloudflare"}) + `\b([A-Za-z0-9_-]{37})\b`)

	emailPat = regexp.MustCompile(`\b([A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}(\.[A-Za-z]{2})?)\b`)
)

// Keywords are used for efficiently pre-filtering chunks.
// Use identifiers in the secret preferably, or the provider name.
func (s Scanner) Keywords() []string {
	return []string{"cloudflare"}
}

// FromData will find and optionally verify CloudflareGlobalApiKey secrets in a given set of bytes.
func (s Scanner) FromData(ctx context.Context, verify bool, data []byte) (results []detectors.Result, err error) {
	dataStr := string(data)

	apiKeyMatches := apiKeyPat.FindAllStringSubmatch(dataStr, -1)
	emailMatches := emailPat.FindAllStringSubmatch(dataStr, -1)

	for _, apiKeyMatch := range apiKeyMatches {
		if len(apiKeyMatch) != 2 {
			continue
		}
		apiKeyRes := strings.TrimSpace(apiKeyMatch[1])

		for _, emailMatch := range emailMatches {
			if len(emailMatch) != 2 {
				continue
			}
			emailRes := strings.TrimSpace(emailMatch[1])

			s1 := detectors.Result{
				DetectorType: detectorspb.DetectorType_CloudflareGlobalApiKey,
				Redacted:     emailRes,
				Raw:          []byte(apiKeyRes),
				RawV2:        []byte(apiKeyRes + emailRes),
			}

			if verify {
				req, err := http.NewRequestWithContext(ctx, "GET", "https://api.cloudflare.com/client/v4/user", nil)
				if err != nil {
					continue
				}
				req.Header.Add("X-Auth-Email", emailRes)
				req.Header.Add("X-Auth-Key", apiKeyRes)
				req.Header.Add("Content-Type", "application/json")

				res, err := client.Do(req)
				if err == nil {
					defer res.Body.Close()
					if res.StatusCode >= 200 && res.StatusCode < 300 {
						s1.Verified = true
					}
				}
			}

			results = append(results, s1)
		}
	}

	return results, nil
}

func (s Scanner) Type() detectorspb.DetectorType {
	return detectorspb.DetectorType_CloudflareGlobalApiKey
}

func (s Scanner) Description() string {
	return "Cloudflare is a web infrastructure and website security company. Its services include content delivery network (CDN), DDoS mitigation, Internet security, and distributed domain name server (DNS) services. Cloudflare API keys can be used to access and modify these services."
}
