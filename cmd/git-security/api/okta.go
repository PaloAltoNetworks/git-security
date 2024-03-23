package api

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/url"
	"path"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/gofiber/fiber/v2"
	verifier "github.com/okta/okta-jwt-verifier-golang"
	"github.com/spf13/cast"
)

var (
	state = generateState()
	nonce = "NonceNotSetYet"
)

func generateState() string {
	// Generate a random byte array for state paramter
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateNonce() string {
	nonceBytes := make([]byte, 32)
	rand.Read(nonceBytes)
	return base64.URLEncoding.EncodeToString(nonceBytes)
}

type Exchange struct {
	Error            string `json:"error,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`
	AccessToken      string `json:"access_token,omitempty"`
	TokenType        string `json:"token_type,omitempty"`
	ExpiresIn        int    `json:"expires_in,omitempty"`
	Scope            string `json:"scope,omitempty"`
	IdToken          string `json:"id_token,omitempty"`
}

// https://developer.okta.com/docs/guides/sign-into-web-app/go/redirect-to-sign-in/
func (api *api) oktaLogin(c *fiber.Ctx) error {
	nonce = generateNonce()

	q := make(url.Values)
	q.Add("client_id", api.oktaOpts.OktaClientID)
	q.Add("response_type", "code")
	q.Add("response_mode", "query")
	q.Add("scope", "openid profile email")
	q.Add("redirect_uri", api.oktaOpts.OktaRedirectURL)
	q.Add("state", state)
	q.Add("nonce", nonce)
	redirectPath := fmt.Sprintf("%s/v1/authorize?%s", api.getHostPath(), q.Encode())

	c.Response().Header.Add("Cache-Control", "no-cache")
	return c.Redirect(redirectPath, fiber.StatusMovedPermanently)
}

func (api *api) oktaLogout(c *fiber.Ctx) error {
	sess, err := api.store.Get(c)
	if err != nil {
		return err
	}
	idToken := cast.ToString(sess.Get("id_token"))
	sess.Delete("id_token")
	sess.Delete("access_token")
	sess.Delete("email")
	if err := sess.Save(); err != nil {
		return err
	}

	q := make(url.Values)
	q.Add("id_token_hint", idToken)
	q.Add("post_logout_redirect_uri", api.oktaOpts.OktaLogoutRedirectURL)
	redirectPath := fmt.Sprintf("%s/v1/logout?%s", api.getHostPath(), q.Encode())

	c.Response().Header.Add("Cache-Control", "no-cache")
	return c.Redirect(redirectPath, fiber.StatusMovedPermanently)
}

// https://developer.okta.com/docs/guides/sign-into-web-app/go/define-callback/
func (api *api) oktaCallback(c *fiber.Ctx) error {
	// Check the state that was returned in the query string is the same as the above state
	if c.Query("state") != state {
		return fiber.NewError(fiber.StatusInternalServerError, "The state was not as expected")
	}
	// Make sure the code was provided
	if c.Query("code") == "" {
		return fiber.NewError(
			fiber.StatusInternalServerError,
			"The code was not returned or is not accessible",
		)
	}

	exchange, err := api.exchangeCode(c.Query("code"))
	if err != nil {
		return err
	}

	_, verificationError := api.verifyToken(exchange.IdToken)
	if verificationError == nil {
		// fetch profile
		client := resty.New().
			SetTimeout(30 * time.Second).
			SetBaseURL(api.getHostPath()).
			OnAfterResponse(func(c *resty.Client, resp *resty.Response) error {
				if resp.IsError() {
					return fmt.Errorf("status code: %d, response: %s", resp.StatusCode(), resp.String())
				}
				return nil
			})

		m := make(map[string]interface{})
		if _, err := client.R().
			SetHeaders(map[string]string{
				"Authorization": "Bearer " + exchange.AccessToken,
				"Accept":        "application/json",
			}).
			SetResult(&m).
			Post("/v1/userinfo"); err != nil {
			slog.Error("error in /v1/userinfo", slog.String("error", err.Error()))
			return err
		}

		sess, err := api.store.Get(c)
		if err != nil {
			return err
		}
		sess.Set("id_token", exchange.IdToken)
		sess.Set("access_token", exchange.AccessToken)
		sess.Set("email", m["email"])
		if err := sess.Save(); err != nil {
			return err
		}
	} else {
		slog.Error("error in verifyToken", slog.String("error", verificationError.Error()))
	}

	return c.Redirect("/", fiber.StatusMovedPermanently)
}

func (api *api) getHostPath() string {
	hostPath, _ := url.Parse(api.oktaOpts.OktaIssuer)
	hostPath.Path = path.Join(hostPath.Path, api.oktaOpts.OktaAPIPath)
	return hostPath.String()
}

func (api *api) exchangeCode(code string) (Exchange, error) {
	client := resty.New().
		SetTimeout(30 * time.Second).
		SetBaseURL(api.getHostPath()).
		OnAfterResponse(func(c *resty.Client, resp *resty.Response) error {
			if resp.IsError() {
				return fmt.Errorf("status code: %d, response: %s", resp.StatusCode(), resp.String())
			}
			return nil
		})

	var exchange Exchange
	if _, err := client.R().
		SetQueryParam("grant_type", "authorization_code").
		SetQueryParam("redirect_uri", api.oktaOpts.OktaRedirectURL).
		SetQueryParam("code", code).
		SetHeaders(map[string]string{
			"Authorization": "Basic " + base64.StdEncoding.EncodeToString(
				[]byte(api.oktaOpts.OktaClientID+":"+api.oktaOpts.OktaClientSecret)),
			"Accept":         "application/json",
			"Content-Type":   "application/x-www-form-urlencoded",
			"Connection":     "close",
			"Content-Length": "0",
		}).
		SetResult(&exchange).
		Post("/v1/token"); err != nil {
		slog.Error("error in /v1/token", slog.String("error", err.Error()))
		return exchange, err
	}
	return exchange, nil
}

func (api *api) verifyToken(t string) (*verifier.Jwt, error) {
	tv := map[string]string{}
	tv["nonce"] = nonce
	tv["aud"] = api.oktaOpts.OktaClientID
	jv := verifier.JwtVerifier{
		Issuer:           api.oktaOpts.OktaIssuer,
		ClaimsToValidate: tv,
	}

	result, err := jv.New().VerifyIdToken(t)

	if err != nil {
		return nil, fmt.Errorf("%s", err)
	}

	if result != nil {
		return result, nil
	}

	return nil, fmt.Errorf("token could not be verified: %s", "")
}

func (api *api) oktaAuthenticator() fiber.Handler {
	return func(c *fiber.Ctx) error {
		sess, err := api.store.Get(c)
		if err != nil {
			return err
		}
		if sess.Get("id_token") == nil {
			c.Response().Header.Add("Cache-Control", "no-cache")
			return c.Redirect("/login", fiber.StatusMovedPermanently)
		}

		// check if the user has any role, if not, by default assign them to "user"
		email := cast.ToString(sess.Get("email"))
		roles, err := api.enforcer.GetRolesForUser(email)
		if err != nil {
			return err
		}
		if len(roles) == 0 {
			if _, err := api.enforcer.AddRoleForUser(email, "user"); err != nil {
				return err
			}
			api.enforcer.SavePolicy()
		}

		return c.Next()
	}
}
