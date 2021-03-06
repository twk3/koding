package webhook

import (
	"fmt"
	"koding/db/mongodb/modelhelper"
	"math/rand"
	"socialapi/config"
	"socialapi/models"
	"testing"
	"time"

	"github.com/koding/bongo"
	"github.com/koding/logging"
	"github.com/koding/runner"
	. "github.com/smartystreets/goconvey/convey"
)

func tearUp(t *testing.T, f func()) {
	r := runner.New("test")
	if err := r.Init(); err != nil {
		t.Fatalf("something went wrong: %s", err)
	}
	defer r.Close()

	appConfig := config.MustRead(r.Conf.Path)
	r.Log.SetLevel(logging.CRITICAL)

	modelhelper.Initialize(appConfig.Mongo)
	defer modelhelper.Close()

	botAcc, err := models.CreateAccountInBothDbsWithNick("bot")
	if err != nil || botAcc == nil {
		t.Fatalf("could not create bot account: %s", err)
	}
	f()
}

func TestSaveMessage(t *testing.T) {

	tearUp(t, func() {
		Convey("while testing bot", t, func() {

			bot, err := NewBot()
			So(err, ShouldBeNil)

			rand.Seed(time.Now().UTC().UnixNano())

			groupName := models.RandomGroupName()

			account, err := models.CreateAccountInBothDbsWithNick("sinan")
			So(err, ShouldBeNil)
			So(account, ShouldNotBeNil)

			channel := models.CreateTypedGroupedChannelWithTest(account.Id, models.Channel_TYPE_TOPIC, groupName)

			ci, _ := CreateTestChannelIntegration(t)

			Convey("bot should be able to create message", func() {
				message := &Message{}
				message.Body = "testmessage"
				message.ChannelId = channel.Id
				message.ChannelIntegrationId = ci.Id
				err := bot.SaveMessage(message)
				So(err, ShouldBeNil)

				m, err := channel.FetchLastMessage()
				So(err, ShouldBeNil)
				So(m, ShouldNotBeNil)
				So(m.Body, ShouldEqual, message.Body)
				So(m.InitialChannelId, ShouldEqual, message.ChannelId)
				So(m.AccountId, ShouldEqual, bot.account.Id)
				So(m.TypeConstant, ShouldEqual, models.ChannelMessage_TYPE_BOT)
				ID := fmt.Sprintf("%d", ci.Id)
				So(*(m.GetPayload("channelIntegrationId")), ShouldEqual, ID)

			})
		})
	})

}

func TestFetchBotChannel(t *testing.T) {

	tearUp(t, func() {
		Convey("while testing bot", t, func() {
			bot, err := NewBot()
			So(err, ShouldBeNil)

			acc, err := models.CreateAccountInBothDbsWithNick("bot-" + models.RandomName())
			So(err, ShouldBeNil)
			So(acc, ShouldNotBeNil)

			groupName := models.RandomGroupName()
			Convey("we should be able to create bot channel for each user", func() {
				// make sure the bot channel for the user does not exist
				channel, err := fetchBotChannel(acc, groupName)
				So(err, ShouldEqual, bongo.RecordNotFound)

				channel, err = bot.fetchOrCreateChannel(acc, groupName)
				So(err, ShouldBeNil)
				So(channel, ShouldNotBeNil)
				So(channel.TypeConstant, ShouldEqual, models.Channel_TYPE_BOT)
				So(channel.CreatorId, ShouldEqual, acc.Id)
			})

			Convey("we should be able to fetch bot channel when it is already created", func() {
				// make sure the channel already exists
				channel, err := bot.createBotChannel(acc, groupName)
				So(err, ShouldBeNil)
				So(channel, ShouldNotBeNil)
				So(channel.TypeConstant, ShouldEqual, models.Channel_TYPE_BOT)
				So(channel.CreatorId, ShouldEqual, acc.Id)

				channel.AddParticipant(acc.Id)

				testchannel, err := bot.fetchOrCreateChannel(acc, groupName)
				So(err, ShouldBeNil)
				So(testchannel, ShouldNotBeNil)
				So(testchannel.Id, ShouldEqual, channel.Id)
			})

		})
	})

}
