package hydra

import (
	"context"

	client "github.com/ory/hydra-client-go/v2"
)

// Service wraps Hydra client for login/consent/logout flows
type Service struct {
	admin *client.APIClient
}

// NewService creates a new Hydra service with the given admin URL
func NewService(adminURL string) *Service {
	config := client.NewConfiguration()
	config.Servers = []client.ServerConfiguration{
		{URL: adminURL},
	}
	return &Service{
		admin: client.NewAPIClient(config),
	}
}

// GetLoginRequest fetches information about a login request
func (s *Service) GetLoginRequest(ctx context.Context, challenge string) (*client.OAuth2LoginRequest, error) {
	req, _, err := s.admin.OAuth2API.GetOAuth2LoginRequest(ctx).
		LoginChallenge(challenge).
		Execute()
	return req, err
}

// AcceptLogin accepts a login request
func (s *Service) AcceptLogin(ctx context.Context, challenge string, subject string, remember bool, rememberFor int64) (*client.OAuth2RedirectTo, error) {
	body := client.NewAcceptOAuth2LoginRequest(subject)
	body.SetRemember(remember)
	body.SetRememberFor(rememberFor)

	resp, _, err := s.admin.OAuth2API.AcceptOAuth2LoginRequest(ctx).
		LoginChallenge(challenge).
		AcceptOAuth2LoginRequest(*body).
		Execute()
	return resp, err
}

// RejectLogin rejects a login request
func (s *Service) RejectLogin(ctx context.Context, challenge string, errorID string, errorDescription string) (*client.OAuth2RedirectTo, error) {
	body := client.NewRejectOAuth2Request()
	body.SetError(errorID)
	body.SetErrorDescription(errorDescription)

	resp, _, err := s.admin.OAuth2API.RejectOAuth2LoginRequest(ctx).
		LoginChallenge(challenge).
		RejectOAuth2Request(*body).
		Execute()
	return resp, err
}

// GetConsentRequest fetches information about a consent request
func (s *Service) GetConsentRequest(ctx context.Context, challenge string) (*client.OAuth2ConsentRequest, error) {
	req, _, err := s.admin.OAuth2API.GetOAuth2ConsentRequest(ctx).
		ConsentChallenge(challenge).
		Execute()
	return req, err
}

// AcceptConsent accepts a consent request
func (s *Service) AcceptConsent(ctx context.Context, challenge string, grantScope []string, remember bool, rememberFor int64, session *client.AcceptOAuth2ConsentRequestSession) (*client.OAuth2RedirectTo, error) {
	body := client.NewAcceptOAuth2ConsentRequest()
	body.SetGrantScope(grantScope)
	body.SetRemember(remember)
	body.SetRememberFor(rememberFor)
	if session != nil {
		body.SetSession(*session)
	}

	resp, _, err := s.admin.OAuth2API.AcceptOAuth2ConsentRequest(ctx).
		ConsentChallenge(challenge).
		AcceptOAuth2ConsentRequest(*body).
		Execute()
	return resp, err
}

// RejectConsent rejects a consent request
func (s *Service) RejectConsent(ctx context.Context, challenge string, errorID string, errorDescription string) (*client.OAuth2RedirectTo, error) {
	body := client.NewRejectOAuth2Request()
	body.SetError(errorID)
	body.SetErrorDescription(errorDescription)

	resp, _, err := s.admin.OAuth2API.RejectOAuth2ConsentRequest(ctx).
		ConsentChallenge(challenge).
		RejectOAuth2Request(*body).
		Execute()
	return resp, err
}

// GetLogoutRequest fetches information about a logout request
func (s *Service) GetLogoutRequest(ctx context.Context, challenge string) (*client.OAuth2LogoutRequest, error) {
	req, _, err := s.admin.OAuth2API.GetOAuth2LogoutRequest(ctx).
		LogoutChallenge(challenge).
		Execute()
	return req, err
}

// AcceptLogout accepts a logout request
func (s *Service) AcceptLogout(ctx context.Context, challenge string) (*client.OAuth2RedirectTo, error) {
	resp, _, err := s.admin.OAuth2API.AcceptOAuth2LogoutRequest(ctx).
		LogoutChallenge(challenge).
		Execute()
	return resp, err
}

// RejectLogout rejects a logout request
func (s *Service) RejectLogout(ctx context.Context, challenge string) error {
	_, err := s.admin.OAuth2API.RejectOAuth2LogoutRequest(ctx).
		LogoutChallenge(challenge).
		Execute()
	return err
}

// IntrospectToken validates a token and returns its metadata
func (s *Service) IntrospectToken(ctx context.Context, token string, scope string) (*client.IntrospectedOAuth2Token, error) {
	req := s.admin.OAuth2API.IntrospectOAuth2Token(ctx).Token(token)
	if scope != "" {
		req = req.Scope(scope)
	}
	resp, _, err := req.Execute()
	return resp, err
}
