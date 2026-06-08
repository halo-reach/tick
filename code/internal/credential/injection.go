package credential

import "github.com/tickplatform/tick/internal/domain"

type injectDefault struct {
	Location string
	Key      string
	Prefix   string
}

var typeDefaults = map[domain.CredentialType]injectDefault{
	domain.CredTypeBearer:   {"header", "Authorization", "Bearer "},
	domain.CredTypeBasic:    {"header", "Authorization", "Basic "},
	domain.CredTypeOAuth2CC: {"header", "Authorization", "Bearer "},
	domain.CredTypeDynamic:  {"header", "Authorization", "Bearer "},
	domain.CredTypeHMAC:     {"header", "X-Signature", ""},
}

func BuildInjections(resolved *ResolvedCredential) []domain.CredentialInjection {
	if resolved.Type == domain.CredTypeCustomHeader {
		var injs []domain.CredentialInjection
		for k, v := range resolved.Headers {
			injs = append(injs, domain.CredentialInjection{Location: "header", Key: k, Value: v})
		}
		return injs
	}

	def := typeDefaults[resolved.Type]
	loc := ptrOr(resolved.InjectLocation, def.Location)
	key := ptrOr(resolved.InjectKey, def.Key)
	prefix := ptrOr(resolved.InjectPrefix, def.Prefix)

	return []domain.CredentialInjection{{Location: loc, Key: key, Value: prefix + resolved.Token}}
}

func ptrOr(p *string, def string) string {
	if p != nil {
		return *p
	}
	return def
}
