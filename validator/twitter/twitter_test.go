package twitter

import (
	"testing"

	"github.com/google/uuid"
	"github.com/nextdotid/proof-server/config"
	"github.com/nextdotid/proof-server/types"
	"github.com/nextdotid/proof-server/util"
	mycrypto "github.com/nextdotid/proof-server/util/crypto"
	"github.com/nextdotid/proof-server/validator"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func before_each(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	config.Init("../../config/config.test.json")
}

func generate() Twitter {
	pubkey, _ := mycrypto.StringToPubkey("0x04666b700aeb6a6429f13cbb263e1bc566cd975a118b61bc796204109c1b351d19b7df23cc47f004e10fef41df82bad646b027578f8881f5f1d2f70c80dfcd8031")
	created_at, _ := util.TimestampStringToTime("1647503071")
	return Twitter{
		Base: &validator.Base{
			Platform:      types.Platforms.Twitter,
			Previous:      "",
			Action:        types.Actions.Create,
			Pubkey:        pubkey,
			Identity:      "yeiwb",
			ProofLocation: "1504363098328924163",
			Text:          "",
			Uuid:          uuid.MustParse("c6fa1483-1bad-4f07-b661-678b191ab4b3"),
			CreatedAt:     created_at,
		},
	}
}

func Test_GeneratePostPayload(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		before_each(t)

		tweet := generate()
		result := tweet.GeneratePostPayload()
		assert.Contains(t, result["default"], "Verifying my Twitter ID")
		assert.Contains(t, result["default"], tweet.Identity)
		assert.Contains(t, result["default"], "%SIG_BASE64%")
	})
}

func Test_Validate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		before_each(t)

		tweet := generate()
		assert.Nil(t, tweet.Validate())
		assert.Greater(t, len(tweet.Text), 10)
		assert.NotEmpty(t, tweet.Text)
		assert.Equal(t, "yeiwb", tweet.Identity)
	})

	t.Run("should return identity error", func(t *testing.T) {
		before_each(t)

		tweet := generate()
		tweet.Identity = "foobar"
		assert.NotNil(t, tweet.Validate())
	})

	t.Run("should return proof location not found", func(t *testing.T) {
		before_each(t)

		tweet := generate()
		tweet.ProofLocation = "123456"
		assert.NotNil(t, tweet.Validate())
	})
}
