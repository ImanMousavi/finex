package auth

import (
	"encoding/base64"
	"os"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/gofiber/fiber/v2"
	"github.com/zsmartex/go-finex/config"
	"github.com/zsmartex/go-finex/models"
)

// Auth struct represents parsed jwt information.
type Auth struct {
	UID        string   `json:"uid"`
	State      string   `json:"state"`
	Email      string   `json:"email"`
	Username   string   `json:"username"`
	Role       string   `json:"role"`
	ReferralID string   `json:"referral_id"`
	Level      int32    `json:"level"`
	Audience   []string `json:"aud,omitempty"`

	jwt.StandardClaims
}

func GetCurrentUser(c *fiber.Ctx) *models.Member {
	var err error
	var auth Auth
	member := &models.Member{}
	token := c.Get("Authorization")
	token = strings.Replace(token, "Bearer ", "", -1)

	public_key_pem, err := base64.StdEncoding.DecodeString(os.Getenv("JWT_PUBLIC_KEY"))

	if err != nil {
		c.Status(500).JSON(fiber.Map{
			"errors": []string{"decode_and_verify"},
		})
		return nil
	}

	public_key, err := jwt.ParseRSAPublicKeyFromPEM(public_key_pem)

	if err != nil {
		c.Status(500).JSON(fiber.Map{
			"errors": []string{"decode_and_verify"},
		})
		return nil
	}

	_, err = jwt.ParseWithClaims(token, &auth, func(t *jwt.Token) (interface{}, error) {
		return public_key, nil
	})

	if err != nil {
		c.Status(500).JSON(fiber.Map{
			"errors": []string{"decode_and_verify"},
		})
		return nil
	}

	config.DataBase.Where(
		&models.Member{
			UID: auth.UID,
		},
	).Attrs(
		models.Member{
			Email:    auth.Email,
			Username: &auth.Username,
			Role:     auth.Role,
			State:    auth.State,
			Level:    auth.Level,
		},
	).FirstOrCreate(&member)

	return member
}
