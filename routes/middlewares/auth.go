package middlewares

import (
	"database/sql"
	"encoding/base64"
	"os"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/gofiber/fiber/v2"
	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/models"
)

var (
	AuthzInvalidSession = "authz.invalid_session"
	JwtDecodeAndVerify  = "jwt.decode_and_verify"
	ServerInternalError = "server.internal_error"
)

// Auth struct represents parsed jwt information.
// {
//   "iat": 1625267254,
//   "exp": 1625270854,
//   "sub": "session",
//   "iss": "barong",
//   "aud": [
//     "peatio",
//     "barong"
//   ],
//   "jti": "42ad71c6adeca427362e",
//   "uid": "ID0814BA5B0D",
//   "email": "business@zsmart.tech",
//   "role": "admin",
//   "level": 3,
//   "state": "active",
//   "referral_uid": null
// }
type Auth struct {
	UID         string         `json:"uid"`
	State       string         `json:"state"`
	Email       string         `json:"email"`
	Username    sql.NullString `json:"username"`
	Role        string         `json:"role"`
	ReferralUID sql.NullString `json:"referral_uid"`
	Level       int32          `json:"level"`
	Audience    []string       `json:"aud,omitempty"`

	jwt.StandardClaims
}

func Authenticate(c *fiber.Ctx) error {
	var err error
	var auth Auth

	var member *models.Member

	token := c.Get("Authorization")

	if len(token) == 0 {
		return c.Status(401).JSON(fiber.Map{
			"errors": []string{AuthzInvalidSession},
		})
	}

	token = strings.Replace(token, "Bearer ", "", -1)

	public_key_pem, err := base64.StdEncoding.DecodeString(os.Getenv("JWT_PUBLIC_KEY"))

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"errors": []string{ServerInternalError},
		})
	}

	public_key, err := jwt.ParseRSAPublicKeyFromPEM(public_key_pem)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"errors": []string{ServerInternalError},
		})
	}

	_, err = jwt.ParseWithClaims(token, &auth, func(t *jwt.Token) (interface{}, error) {
		return public_key, nil
	})

	if err != nil {
		return c.Status(422).JSON(fiber.Map{
			"errors": []string{JwtDecodeAndVerify},
		})
	}

	config.DataBase.Where("uid = ?", auth.UID).Assign(
		&models.Member{
			Email:       auth.Email,
			Role:        auth.Role,
			State:       auth.State,
			Level:       auth.Level,
			ReferralUID: auth.ReferralUID,
		},
	).FirstOrCreate(&member)
	config.DataBase.Where("uid = ?", auth.UID).Updates(&models.Member{
		Role:  auth.Role,
		State: auth.State,
		Level: auth.Level,
	})

	c.Locals("CurrentUser", member)

	return c.Next()
}
