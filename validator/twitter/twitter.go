package twitter

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	t "github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/nextdotid/proof-server/config"
	mycrypto "github.com/nextdotid/proof-server/util/crypto"
	"github.com/sirupsen/logrus"

	"github.com/nextdotid/proof-server/validator"
)

type Twitter struct {
	validator.Base

	// Filled when tweet fetched successfully.
	TweetText string
}

const (
	MATCH_TEMPLATE = "^Prove myself: I'm 0x([0-9a-f]{66}) on NextID. Signature: (.*)$"
	POST_STRUCT = "Prove myself: I'm 0x%s on NextID. Signature: %%SIG_BASE64%%"
)

var (
	client *t.Client
	l      = logrus.WithFields(logrus.Fields{"module": "validator", "validator": "twitter"})
	re     = regexp.MustCompile(MATCH_TEMPLATE)
)

func (twitter *Twitter) GeneratePostPayload() (post string) {
	return fmt.Sprintf(POST_STRUCT, mycrypto.CompressedPubkeyHex(twitter.Pubkey))
}

func (twitter *Twitter) GenerateSignPayload() (payload string) {
	var payloadStruct map[string]interface{}
	payloadStruct = map[string]interface{}{
		"action":   string(twitter.Action),
		"platform": "twitter",
		"identity": twitter.Identity,
		"prev":     nil,
	}
	if twitter.Previous != "" {
		payloadStruct["prev"] = twitter.Previous
	}

	payloadBytes, err := json.Marshal(payloadStruct)
	if err != nil {
		l.Warnf("Error when marshaling struct: %s", err.Error())
		return ""
	}

	return string(payloadBytes)
}

func (twitter *Twitter) Validate() (result bool) {
	initClient()
	tweetID, err := strconv.ParseInt(twitter.ProofLocation, 10, 64)
	if err != nil {
		l.Warnf("Error when parsing tweet ID %s: %s", twitter.ProofLocation, err.Error())
		return false
	}

	tweet, _, err := client.Statuses.Show(tweetID, &t.StatusShowParams{
		TweetMode: "extended",
	})
	if err != nil {
		l.Warnf("Error when getting tweet %s: %s", twitter.ProofLocation, err.Error())
		return false
	}
	if tweet.User.ScreenName != twitter.Identity {
		l.Warnf("Screen name mismatch: expect %s - actual %s", twitter.Identity, tweet.User.ScreenName)
		return false
	}

	twitter.TweetText = tweet.FullText
	return twitter.validateText()
}

func (twitter *Twitter) validateText() bool {
	l := l.WithFields(logrus.Fields{"function": "validateText", "tweet": twitter.ProofLocation})
	matched := re.FindStringSubmatch(twitter.TweetText)
	if len(matched) < 3 {
		l.Warnf("Tweet struct mismatch. Found: %+v", matched)
		return false
	}

	pubkeyHex := matched[1]
	pubkeyRecovered, err := mycrypto.StringToPubkey(pubkeyHex)
	if err != nil {
		l.Warnf("Pubkey recover failed: %s", err.Error())
		return false
	}
	if crypto.PubkeyToAddress(*twitter.Pubkey) != crypto.PubkeyToAddress(*pubkeyRecovered) {
		l.Warnf("Pubkey mismatch")
		return false
	}

	sigBase64 := matched[2]
	sigBytes, err := base64.StdEncoding.DecodeString(sigBase64)
	if err != nil {
		l.Warnf("Error when decoding signature %s: %s", sigBase64, err.Error())
		return false
	}
	return mycrypto.ValidatePersonalSignature(twitter.GenerateSignPayload(), sigBytes, pubkeyRecovered)
}

func initClient() {
	if client != nil {
		return
	}
	oauthToken := oauth1.NewToken(
		config.C.Platform.Twitter.AccessToken,
		config.C.Platform.Twitter.AccessTokenSecret,
	)
	oauthConfig := oauth1.NewConfig(
		config.C.Platform.Twitter.ConsumerKey,
		config.C.Platform.Twitter.ConsumerSecret,
	)
	httpClient := oauthConfig.Client(oauth1.NoContext, oauthToken)
	client = t.NewClient(httpClient)
}
