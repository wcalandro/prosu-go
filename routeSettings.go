package main

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-bongo/bongo"

	"github.com/globalsign/mgo/bson"
	osuapi "github.com/wcalandro/osuapi-go"

	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/sessions"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type settingsPageData struct {
	User            User
	Translations    settingsPageTranslations
	IsAuthenticated bool
	OsuPlayer       OsuPlayer
	Modes           [4]string
	ErrorFlash      []interface{}
	SuccessFlash    []interface{}
}

type settingsPageTranslations struct {
	Navbar                     navbarTranslations
	SettingsHeader             string
	TweetPostingText           string
	TweetPostingStatusEnabled  string
	TweetPostingStatusDisabled string
	EnableTweetPosting         string
	DisableTweetPosting        string
	OsuUsernameText            string
	OsuUsernamePlaceholder     string
	GameModeText               string
	UpdateSettingsButton       string
	NoDataWarning              string
}

var allOsuModes = [4]string{"osu!standard", "osu!taiko", "osu!catch", "osu!mania"}

func routeSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionError := ctx.Value("session_error").(string)
	if sessionError != "" {
		log.Error("There was an error getting the user's session")
		log.Error(sessionError)
		reqID := middleware.GetReqID(ctx)
		routeError(w, "Error getting user session", errors.New(sessionError), reqID, 500)
		return
	}
	session := ctx.Value("session").(*sessions.Session)
	isAuthenticated := ctx.Value("isAuthenticated").(bool)

	// Privileged page. If the user isn't authenticated, we need to redirect the user to login
	if isAuthenticated == false {
		http.Redirect(w, r, "/connect/twitter", 302)
		return
	}

	var user User
	userError := ctx.Value("user_error").(string)
	if userError != "" {
		log.Error("There was an error getting the user's account info")
		log.Error(userError)
		reqID := middleware.GetReqID(ctx)
		routeError(w, "Error getting user account info", errors.New(userError), reqID, 500)
		return
	}
	user = *ctx.Value("user").(*User)

	// Localization
	lang := session.Values["language"].(string)
	accept := r.Header.Get("Accept-Language")
	localizer := i18n.NewLocalizer(bundle, lang, accept)

	translations := translateSettingsPage(localizer, isAuthenticated, user)

	// Grab the osu! player in the user's settings if it is currently set
	player := OsuPlayer{}
	if bson.IsObjectIdHex(user.OsuSettings.Player.Hex()) {
		err := connection.Collection("osuplayermodels").FindById(bson.ObjectIdHex(user.OsuSettings.Player.Hex()), &player)
		if err != nil {
			routeError(w, "Error getting osu! player information from database", err, middleware.GetReqID(ctx), 500)
			return
		}
	}

	errorFlashes := session.Flashes("settings_error")
	successFlashes := session.Flashes("settings_success")
	pageData := settingsPageData{
		User:            user,
		OsuPlayer:       player,
		IsAuthenticated: true,
		Translations:    translations,
		Modes:           allOsuModes,
		ErrorFlash:      errorFlashes,
		SuccessFlash:    successFlashes,
	}

	templates.ExecuteTemplate(w, "settings.html", pageData)
}

func translateSettingsPage(localizer *i18n.Localizer, isAuthenticated bool, user User) settingsPageTranslations {
	navbar := translateNavbar(localizer, isAuthenticated, user)

	settingsHeaderText := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "SettingsPageSettingsHeader",
	})

	settingsTweetPostingText := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "SettingsPageTweetPostingText",
	})

	settingsTweetPostingEnabled := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "SettingsPageTweetPostingStatusEnabled",
	})

	settingsTweetPostingDisabled := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "SettingsPageTweetPostingStatusDisabled",
	})

	settingsOsuUsernameText := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "SettingsPageOsuUsernameText",
	})

	settingsOsuUsernamePlaceholder := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "SettingsPageOsuUsernamePlaceholder",
	})

	settingsEnableTweetPosting := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "SettingsPageEnableTweetPostingButton",
	})

	settingsDisableTweetPosting := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "SettingsPageDisableTweetPostingButton",
	})

	gameModeText := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "SettingsPageGameModeText",
	})

	updateSettings := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "SettingsPageUpdateSettingsButton",
	})

	noDataWarning := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "SettingsPageNoDataWarning",
	})

	return settingsPageTranslations{
		Navbar:                     navbar,
		SettingsHeader:             settingsHeaderText,
		TweetPostingText:           settingsTweetPostingText,
		TweetPostingStatusEnabled:  settingsTweetPostingEnabled,
		TweetPostingStatusDisabled: settingsTweetPostingDisabled,
		EnableTweetPosting:         settingsEnableTweetPosting,
		DisableTweetPosting:        settingsDisableTweetPosting,
		OsuUsernameText:            settingsOsuUsernameText,
		OsuUsernamePlaceholder:     settingsOsuUsernamePlaceholder,
		GameModeText:               gameModeText,
		UpdateSettingsButton:       updateSettings,
		NoDataWarning:              noDataWarning,
	}
}

func enableTweetPosting(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionError := ctx.Value("session_error").(string)
	if sessionError != "" {
		log.Error("There was an error getting the user's session")
		log.Error(sessionError)
		reqID := middleware.GetReqID(ctx)
		routeError(w, "Error getting user session", errors.New(sessionError), reqID, 500)
		return
	}
	isAuthenticated := ctx.Value("isAuthenticated").(bool)

	// Privileged page. If the user isn't authenticated, we need to redirect the user to login
	if isAuthenticated == false {
		http.Redirect(w, r, "/connect/twitter", 302)
		return
	}

	var user User
	userError := ctx.Value("user_error").(string)
	if userError != "" {
		log.Error("There was an error getting the user's account info")
		log.Error(userError)
		reqID := middleware.GetReqID(ctx)
		routeError(w, "Error getting user account info", errors.New(userError), reqID, 500)
		return
	}
	user = *ctx.Value("user").(*User)
	if user.OsuSettings.Enabled {
		http.Redirect(w, r, "/settings", 302)
		return
	}

	user.OsuSettings.Enabled = true
	err := connection.Collection("usermodels").Save(&user)
	if err != nil {
		routeError(w, "Error saving user when enabling tweets", err, middleware.GetReqID(ctx), 500)
		return
	}
	http.Redirect(w, r, "/settings", 302)
}

func disableTweetPosting(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionError := ctx.Value("session_error").(string)
	if sessionError != "" {
		log.Error("There was an error getting the user's session")
		log.Error(sessionError)
		reqID := middleware.GetReqID(ctx)
		routeError(w, "Error getting user session", errors.New(sessionError), reqID, 500)
		return
	}
	isAuthenticated := ctx.Value("isAuthenticated").(bool)

	// Privileged page. If the user isn't authenticated, we need to redirect the user to login
	if isAuthenticated == false {
		http.Redirect(w, r, "/connect/twitter", 302)
		return
	}

	var user User
	userError := ctx.Value("user_error").(string)
	if userError != "" {
		log.Error("There was an error getting the user's account info")
		log.Error(userError)
		reqID := middleware.GetReqID(ctx)
		routeError(w, "Error getting user account info", errors.New(userError), reqID, 500)
		return
	}
	user = *ctx.Value("user").(*User)
	if user.OsuSettings.Enabled == false {
		http.Redirect(w, r, "/settings", 302)
		return
	}

	user.OsuSettings.Enabled = false
	err := connection.Collection("usermodels").Save(&user)
	if err != nil {
		routeError(w, "Error saving user when disabling tweets", err, middleware.GetReqID(ctx), 500)
		return
	}
	http.Redirect(w, r, "/settings", 302)
}

func updateSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionError := ctx.Value("session_error").(string)
	if sessionError != "" {
		log.Error("There was an error getting the user's session")
		log.Error(sessionError)
		reqID := middleware.GetReqID(ctx)
		routeError(w, "Error getting user session", errors.New(sessionError), reqID, 500)
		return
	}
	session := ctx.Value("session").(*sessions.Session)
	isAuthenticated := ctx.Value("isAuthenticated").(bool)

	// Privileged page. If the user isn't authenticated, we need to redirect the user to login
	if isAuthenticated == false {
		http.Redirect(w, r, "/connect/twitter", 302)
		return
	}

	var user User
	userError := ctx.Value("user_error").(string)
	if userError != "" {
		log.Error("There was an error getting the user's account info")
		log.Error(userError)
		reqID := middleware.GetReqID(ctx)
		routeError(w, "Error getting user account info", errors.New(userError), reqID, 500)
		return
	}
	user = *ctx.Value("user").(*User)
	if user.OsuSettings.Enabled == false {
		http.Redirect(w, r, "/settings", 302)
		return
	}

	// Parse the form
	err := r.ParseForm()
	if err != nil {
		captureError(err)
		session.AddFlash("Error parsing form", "settings_error")
		session.Save(r, w)
		http.Redirect(w, r, "/settings", 302)
		return
	}

	// Check that the mode is valid
	modeNumber, err := strconv.Atoi(r.Form.Get("game_mode"))
	if err != nil {
		session.AddFlash("Invalid mode", "settings_error")
		session.Save(r, w)
		http.Redirect(w, r, "/settings", 302)
		return
	}
	if modeNumber < 0 || modeNumber > 3 {
		session.AddFlash("Invalid mode", "settings_error")
		session.Save(r, w)
		http.Redirect(w, r, "/settings", 302)
		return
	}

	// Get osu! player information
	playerName := r.Form.Get("osu_username")
	osuPlayer, err := api.GetUser(osuapi.M{"u": playerName, "m": strconv.Itoa(modeNumber)})

	if err != nil {
		captureError(err)
		session.AddFlash("Error getting user information", "settings_error")
		session.Save(r, w)
		http.Redirect(w, r, "/settings", 302)
		return
	}

	// Check if the user already exists in our database
	dbOsuPlayer := &OsuPlayer{}
	err = connection.Collection("osuplayermodels").FindOne(bson.M{"userid": osuPlayer.UserID}, dbOsuPlayer)

	if err != nil {
		if _, ok := err.(*bongo.DocumentNotFoundError); ok {
			// Player isn't in the database yet.
			dbOsuPlayer.UserID = osuPlayer.UserID
			dbOsuPlayer.PlayerName = osuPlayer.Username
			dbOsuPlayer.LastChecked = time.Now().Unix()
			dbOsuPlayer.Modes = OsuModes{
				Standard: OsuModeChecks{
					Checks: []bson.ObjectId{},
				},
				Mania: OsuModeChecks{
					Checks: []bson.ObjectId{},
				},
				Taiko: OsuModeChecks{
					Checks: []bson.ObjectId{},
				},
				CTB: OsuModeChecks{
					Checks: []bson.ObjectId{},
				},
			}
			err := connection.Collection("osuplayermodels").Save(dbOsuPlayer)
			if err != nil {
				captureError(err)
				session.AddFlash("Error saving new osu! player", "settings_error")
				session.Save(r, w)
				http.Redirect(w, r, "/settings", 302)
				return
			}
			osuRequest := &OsuRequest{
				OsuPlayer:   dbOsuPlayer.GetId(),
				DateChecked: time.Now().Unix(),
				Data: OsuRequestData{
					PlayerID:   osuPlayer.UserID,
					PlayerName: osuPlayer.Username,
					Counts: requestDataCounts{
						Count50s:  osuPlayer.Count50,
						Count100s: osuPlayer.Count100,
						Count300s: osuPlayer.Count300,
						SS:        osuPlayer.CountRankSS + osuPlayer.CountRankSSH,
						S:         osuPlayer.CountRankS + osuPlayer.CountRankSH,
						A:         osuPlayer.CountRankA,
						Plays:     osuPlayer.Playcount,
					},
					Scores: requestDataScores{
						Ranked: osuPlayer.RankedScore,
						Total:  osuPlayer.TotalScore,
					},
					PP: requestDataPP{
						Raw:         osuPlayer.PP,
						Rank:        osuPlayer.GlobalRank,
						CountryRank: osuPlayer.CountryRank,
					},
					Country:  osuPlayer.Country,
					Level:    osuPlayer.Level,
					Accuracy: osuPlayer.Accuracy,
				},
			}

			err = connection.Collection("osurequestmodels").Save(osuRequest)
			if err != nil {
				captureError(err)
				session.AddFlash("Error saving osu! data request to database", "settings_error")
				session.Save(r, w)
				http.Redirect(w, r, "/settings", 302)
				return
			}
			if modeNumber == 0 {
				// osu! standard
				dbOsuPlayer.Modes.Standard.Checks = append(dbOsuPlayer.Modes.Standard.Checks, osuRequest.GetId())
			} else if modeNumber == 1 {
				// osu! mania
				dbOsuPlayer.Modes.Mania.Checks = append(dbOsuPlayer.Modes.Mania.Checks, osuRequest.GetId())
			} else if modeNumber == 2 {
				// osu! taiko
				dbOsuPlayer.Modes.Taiko.Checks = append(dbOsuPlayer.Modes.Taiko.Checks, osuRequest.GetId())
			} else if modeNumber == 3 {
				// osu! catch
				dbOsuPlayer.Modes.CTB.Checks = append(dbOsuPlayer.Modes.CTB.Checks, osuRequest.GetId())
			}
			err = connection.Collection("osuplayermodels").Save(dbOsuPlayer)
			if err != nil {
				captureError(err)
				session.AddFlash("Error re-saving osu! player to database", "settings_error")
				session.Save(r, w)
				http.Redirect(w, r, "/settings", 302)
				return
			}
			user.OsuSettings.Player = dbOsuPlayer.GetId()
			user.OsuSettings.Mode = modeNumber
			err = connection.Collection("osuplayermodels").Save(dbOsuPlayer)
			if err != nil {
				captureError(err)
				session.AddFlash("Error saving final settings", "settings_error")
				session.Save(r, w)
				http.Redirect(w, r, "/settings", 302)
				return
			}
		} else {
			captureError(err)
			session.AddFlash("Error checking if the user already exists in the database", "settings_error")
			session.Save(r, w)
			http.Redirect(w, r, "/settings", 302)
			return
		}
	}

}
