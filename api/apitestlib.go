// Copyright (c) 2015 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package api

import (
	"time"

	"github.com/mattermost/platform/model"
	"github.com/mattermost/platform/store"
	"github.com/mattermost/platform/utils"

	l4g "github.com/alecthomas/log4go"
)

type TestHelper struct {
	BasicClient  *model.Client
	BasicTeam    *model.Team
	BasicUser    *model.User
	BasicUser2   *model.User
	BasicChannel *model.Channel
	BasicPost    *model.Post

	SystemAdminClient  *model.Client
	SystemAdminTeam    *model.Team
	SystemAdminUser    *model.User
	SystemAdminChannel *model.Channel
}

func SetupEnterprise() *TestHelper {
	if Srv == nil {
		utils.TranslationsPreInit()
		utils.LoadConfig("config.json")
		utils.InitTranslations(utils.Cfg.LocalizationSettings)
		utils.Cfg.TeamSettings.MaxUsersPerTeam = 50
		*utils.Cfg.RateLimitSettings.Enable = false
		utils.DisableDebugLogForTest()
		utils.License.Features.SetDefaults()
		NewServer()
		InitStores()
		InitRouter()
		StartServer()
		utils.InitHTML()
		InitApi()
		utils.EnableDebugLogForTest()
		Srv.Store.MarkSystemRanUnitTests()

		*utils.Cfg.TeamSettings.EnableOpenServer = true
	}

	return &TestHelper{}
}

func Setup() *TestHelper {
	if Srv == nil {
		utils.TranslationsPreInit()
		utils.LoadConfig("config.json")
		utils.InitTranslations(utils.Cfg.LocalizationSettings)
		utils.Cfg.TeamSettings.MaxUsersPerTeam = 50
		*utils.Cfg.RateLimitSettings.Enable = false
		utils.DisableDebugLogForTest()
		NewServer()
		InitStores()
		InitRouter()
		StartServer()
		InitApi()
		utils.EnableDebugLogForTest()
		Srv.Store.MarkSystemRanUnitTests()

		*utils.Cfg.TeamSettings.EnableOpenServer = true
	}

	return &TestHelper{}
}

func (me *TestHelper) InitBasic() *TestHelper {
	me.BasicClient = me.CreateClient()
	me.BasicTeam = me.CreateTeam(me.BasicClient)
	me.BasicUser = me.CreateUser(me.BasicClient)
	LinkUserToTeam(me.BasicUser, me.BasicTeam)
	me.BasicUser2 = me.CreateUser(me.BasicClient)
	LinkUserToTeam(me.BasicUser2, me.BasicTeam)
	me.BasicClient.SetTeamId(me.BasicTeam.Id)
	me.LoginBasic()
	me.BasicChannel = me.CreateChannel(me.BasicClient, me.BasicTeam)
	me.BasicPost = me.CreatePost(me.BasicClient, me.BasicChannel)

	return me
}

func (me *TestHelper) InitSystemAdmin() *TestHelper {
	me.SystemAdminClient = me.CreateClient()
	me.SystemAdminTeam = me.CreateTeam(me.SystemAdminClient)
	me.SystemAdminUser = me.CreateUser(me.SystemAdminClient)
	LinkUserToTeam(me.SystemAdminUser, me.SystemAdminTeam)
	me.SystemAdminClient.SetTeamId(me.SystemAdminTeam.Id)
	UpdateUserRoles(me.SystemAdminUser, model.ROLE_SYSTEM_USER.Id+" "+model.ROLE_SYSTEM_ADMIN.Id)
	me.SystemAdminUser.Password = "Password1"
	me.LoginSystemAdmin()
	me.SystemAdminChannel = me.CreateChannel(me.SystemAdminClient, me.SystemAdminTeam)

	return me
}

func (me *TestHelper) CreateClient() *model.Client {
	return model.NewClient("http://localhost" + utils.Cfg.ServiceSettings.ListenAddress)
}

func (me *TestHelper) CreateWebSocketClient() (*model.WebSocketClient, *model.AppError) {
	return model.NewWebSocketClient("ws://localhost"+utils.Cfg.ServiceSettings.ListenAddress, me.BasicClient.AuthToken)
}

func (me *TestHelper) CreateTeam(client *model.Client) *model.Team {
	id := model.NewId()
	team := &model.Team{
		DisplayName: "dn_" + id,
		Name:        "name" + id,
		Email:       "success+" + id + "@simulator.amazonses.com",
		Type:        model.TEAM_OPEN,
	}

	utils.DisableDebugLogForTest()
	r := client.Must(client.CreateTeam(team)).Data.(*model.Team)
	utils.EnableDebugLogForTest()
	return r
}

func (me *TestHelper) CreateUser(client *model.Client) *model.User {
	id := model.NewId()

	user := &model.User{
		Email:    "success+" + id + "@simulator.amazonses.com",
		Username: "un_" + id,
		Nickname: "nn_" + id,
		Password: "Password1",
	}

	utils.DisableDebugLogForTest()
	ruser := client.Must(client.CreateUser(user, "")).Data.(*model.User)
	ruser.Password = "Password1"
	store.Must(Srv.Store.User().VerifyEmail(ruser.Id))
	utils.EnableDebugLogForTest()
	return ruser
}

func LinkUserToTeam(user *model.User, team *model.Team) {
	utils.DisableDebugLogForTest()

	err := JoinUserToTeam(team, user)
	if err != nil {
		l4g.Error(err.Error())
		l4g.Close()
		time.Sleep(time.Second)
		panic(err)
	}

	utils.EnableDebugLogForTest()
}

func UpdateUserToTeamAdmin(user *model.User, team *model.Team) {
	utils.DisableDebugLogForTest()

	tm := &model.TeamMember{TeamId: team.Id, UserId: user.Id, Roles: model.ROLE_TEAM_USER.Id + " " + model.ROLE_TEAM_ADMIN.Id}
	if tmr := <-Srv.Store.Team().UpdateMember(tm); tmr.Err != nil {
		utils.EnableDebugLogForTest()
		l4g.Error(tmr.Err.Error())
		l4g.Close()
		time.Sleep(time.Second)
		panic(tmr.Err)
	}
	utils.EnableDebugLogForTest()
}

func MakeUserChannelAdmin(user *model.User, channel *model.Channel) {
	utils.DisableDebugLogForTest()

	if cmr := <-Srv.Store.Channel().GetMember(channel.Id, user.Id); cmr.Err == nil {
		cm := cmr.Data.(model.ChannelMember)
		cm.Roles = "channel_admin channel_user"
		if sr := <-Srv.Store.Channel().UpdateMember(&cm); sr.Err != nil {
			utils.EnableDebugLogForTest()
			panic(sr.Err)
		}
	} else {
		utils.EnableDebugLogForTest()
		panic(cmr.Err)
	}

	utils.EnableDebugLogForTest()
}

func MakeUserChannelUser(user *model.User, channel *model.Channel) {
	utils.DisableDebugLogForTest()

	if cmr := <-Srv.Store.Channel().GetMember(channel.Id, user.Id); cmr.Err == nil {
		cm := cmr.Data.(model.ChannelMember)
		cm.Roles = "channel_user"
		if sr := <-Srv.Store.Channel().UpdateMember(&cm); sr.Err != nil {
			utils.EnableDebugLogForTest()
			panic(sr.Err)
		}
	} else {
		utils.EnableDebugLogForTest()
		panic(cmr.Err)
	}

	utils.EnableDebugLogForTest()
}

func (me *TestHelper) CreateChannel(client *model.Client, team *model.Team) *model.Channel {
	return me.createChannel(client, team, model.CHANNEL_OPEN)
}

func (me *TestHelper) CreatePrivateChannel(client *model.Client, team *model.Team) *model.Channel {
	return me.createChannel(client, team, model.CHANNEL_PRIVATE)
}

func (me *TestHelper) createChannel(client *model.Client, team *model.Team, channelType string) *model.Channel {
	id := model.NewId()

	channel := &model.Channel{
		DisplayName: "dn_" + id,
		Name:        "name_" + id,
		Type:        channelType,
		TeamId:      team.Id,
	}

	utils.DisableDebugLogForTest()
	r := client.Must(client.CreateChannel(channel)).Data.(*model.Channel)
	utils.EnableDebugLogForTest()
	return r
}

func (me *TestHelper) CreatePost(client *model.Client, channel *model.Channel) *model.Post {
	id := model.NewId()

	post := &model.Post{
		ChannelId: channel.Id,
		Message:   "message_" + id,
	}

	utils.DisableDebugLogForTest()
	r := client.Must(client.CreatePost(post)).Data.(*model.Post)
	utils.EnableDebugLogForTest()
	return r
}

func (me *TestHelper) LoginBasic() {
	utils.DisableDebugLogForTest()
	me.BasicClient.Must(me.BasicClient.Login(me.BasicUser.Email, me.BasicUser.Password))
	utils.EnableDebugLogForTest()
}

func (me *TestHelper) LoginBasic2() {
	utils.DisableDebugLogForTest()
	me.BasicClient.Must(me.BasicClient.Login(me.BasicUser2.Email, me.BasicUser2.Password))
	utils.EnableDebugLogForTest()
}

func (me *TestHelper) LoginSystemAdmin() {
	utils.DisableDebugLogForTest()
	me.SystemAdminClient.Must(me.SystemAdminClient.Login(me.SystemAdminUser.Email, me.SystemAdminUser.Password))
	utils.EnableDebugLogForTest()
}

func TearDown() {
	if Srv != nil {
		StopServer()
	}
}
