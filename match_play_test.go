// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestMatchPlay(t *testing.T) {
	clearDb()
	defer clearDb()
	var err error
	db, err = OpenDatabase(testDbPath)
	assert.Nil(t, err)
	defer db.Close()
	eventSettings, _ = db.GetEventSettings()

	match1 := Match{Type: "practice", DisplayName: "1", Status: "complete", Winner: "R"}
	match2 := Match{Type: "practice", DisplayName: "2"}
	match3 := Match{Type: "qualification", DisplayName: "1", Status: "complete", Winner: "B"}
	match4 := Match{Type: "elimination", DisplayName: "SF1-1", Status: "complete", Winner: "T"}
	match5 := Match{Type: "elimination", DisplayName: "SF1-2"}
	db.CreateMatch(&match1)
	db.CreateMatch(&match2)
	db.CreateMatch(&match3)
	db.CreateMatch(&match4)
	db.CreateMatch(&match5)

	// Check that all matches are listed on the page.
	recorder := getHttpResponse("/match_play")
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "P1")
	assert.Contains(t, recorder.Body.String(), "P2")
	assert.Contains(t, recorder.Body.String(), "Q1")
	assert.Contains(t, recorder.Body.String(), "SF1-1")
	assert.Contains(t, recorder.Body.String(), "SF1-2")
}

func TestMatchPlayLoad(t *testing.T) {
	clearDb()
	defer clearDb()
	var err error
	db, err = OpenDatabase(testDbPath)
	assert.Nil(t, err)
	defer db.Close()
	eventSettings, _ = db.GetEventSettings()
	mainArena.Setup()

	db.CreateTeam(&Team{Id: 101})
	db.CreateTeam(&Team{Id: 102})
	db.CreateTeam(&Team{Id: 103})
	db.CreateTeam(&Team{Id: 104})
	db.CreateTeam(&Team{Id: 105})
	db.CreateTeam(&Team{Id: 106})
	match := Match{Type: "elimination", DisplayName: "QF4-3", Status: "complete", Winner: "R", Red1: 101,
		Red2: 102, Red3: 103, Blue1: 104, Blue2: 105, Blue3: 106}
	db.CreateMatch(&match)
	recorder := getHttpResponse("/match_play")
	assert.Equal(t, 200, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), "101")
	assert.NotContains(t, recorder.Body.String(), "102")
	assert.NotContains(t, recorder.Body.String(), "103")
	assert.NotContains(t, recorder.Body.String(), "104")
	assert.NotContains(t, recorder.Body.String(), "105")
	assert.NotContains(t, recorder.Body.String(), "106")

	// Load the match and check for the team numbers again.
	recorder = getHttpResponse(fmt.Sprintf("/match_play/%d/load", match.Id))
	assert.Equal(t, 302, recorder.Code)
	recorder = getHttpResponse("/match_play")
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "101")
	assert.Contains(t, recorder.Body.String(), "102")
	assert.Contains(t, recorder.Body.String(), "103")
	assert.Contains(t, recorder.Body.String(), "104")
	assert.Contains(t, recorder.Body.String(), "105")
	assert.Contains(t, recorder.Body.String(), "106")

	// Load a test match.
	recorder = getHttpResponse("/match_play/0/load")
	assert.Equal(t, 302, recorder.Code)
	recorder = getHttpResponse("/match_play")
	assert.Equal(t, 200, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), "101")
	assert.NotContains(t, recorder.Body.String(), "102")
	assert.NotContains(t, recorder.Body.String(), "103")
	assert.NotContains(t, recorder.Body.String(), "104")
	assert.NotContains(t, recorder.Body.String(), "105")
	assert.NotContains(t, recorder.Body.String(), "106")
}

func TestMatchPlayShowResult(t *testing.T) {
	clearDb()
	defer clearDb()
	var err error
	db, err = OpenDatabase(testDbPath)
	assert.Nil(t, err)
	defer db.Close()
	eventSettings, _ = db.GetEventSettings()
	mainArena.Setup()

	recorder := getHttpResponse("/match_play/1/show_result")
	assert.Equal(t, 500, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Invalid match")
	match := Match{Type: "qualification", DisplayName: "1", Status: "complete"}
	db.CreateMatch(&match)
	recorder = getHttpResponse(fmt.Sprintf("/match_play/%d/show_result", match.Id))
	assert.Equal(t, 500, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "No result found")
	db.CreateMatchResult(&MatchResult{MatchId: match.Id})
	recorder = getHttpResponse(fmt.Sprintf("/match_play/%d/show_result", match.Id))
	assert.Equal(t, 302, recorder.Code)
	assert.Equal(t, match.Id, mainArena.savedMatch.Id)
	assert.Equal(t, match.Id, mainArena.savedMatchResult.MatchId)
}

func TestMatchPlayErrors(t *testing.T) {
	clearDb()
	defer clearDb()
	var err error
	db, err = OpenDatabase(testDbPath)
	assert.Nil(t, err)
	defer db.Close()
	eventSettings, _ = db.GetEventSettings()

	// Load an invalid match.
	recorder := getHttpResponse("/match_play/1114/load")
	assert.Equal(t, 500, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Invalid match")
}

func TestCommitMatch(t *testing.T) {
	clearDb()
	defer clearDb()
	var err error
	db, err = OpenDatabase(testDbPath)
	assert.Nil(t, err)
	defer db.Close()
	eventSettings, _ = db.GetEventSettings()

	// Committing test match should do nothing.
	match := &Match{Id: 0, Type: "test", Red1: 101, Red2: 102, Red3: 103, Blue1: 104, Blue2: 105, Blue3: 106}
	err = CommitMatchScore(match, &MatchResult{MatchId: match.Id})
	assert.Nil(t, err)
	matchResult, err := db.GetMatchResultForMatch(match.Id)
	assert.Nil(t, err)
	assert.Nil(t, matchResult)

	// Committing the same match more than once should create a second match result record.
	match.Id = 1
	match.Type = "qualification"
	db.CreateMatch(match)
	matchResult = &MatchResult{MatchId: match.Id, BlueScore: Score{AutoHigh: 1}}
	err = CommitMatchScore(match, matchResult)
	assert.Nil(t, err)
	assert.Equal(t, 1, matchResult.PlayNumber)
	match, _ = db.GetMatchById(1)
	assert.Equal(t, "B", match.Winner)
	matchResult = &MatchResult{MatchId: match.Id, RedScore: Score{AutoHigh: 1}}
	err = CommitMatchScore(match, matchResult)
	assert.Nil(t, err)
	assert.Equal(t, 2, matchResult.PlayNumber)
	match, _ = db.GetMatchById(1)
	assert.Equal(t, "R", match.Winner)
	matchResult = &MatchResult{MatchId: match.Id}
	err = CommitMatchScore(match, matchResult)
	assert.Nil(t, err)
	assert.Equal(t, 3, matchResult.PlayNumber)
	match, _ = db.GetMatchById(1)
	assert.Equal(t, "T", match.Winner)
}

func TestCommitEliminationTie(t *testing.T) {
	clearDb()
	defer clearDb()
	var err error
	db, err = OpenDatabase(testDbPath)
	assert.Nil(t, err)
	defer db.Close()
	eventSettings, _ = db.GetEventSettings()
	mainArena.Setup()

	match := &Match{Id: 0, Type: "qualification", Red1: 1, Red2: 2, Red3: 3, Blue1: 4, Blue2: 5, Blue3: 6}
	db.CreateMatch(match)
	matchResult := &MatchResult{MatchId: match.Id, RedScore: Score{AutoHighHot: 1}, RedFouls: []Foul{Foul{}}}
	CommitMatchScore(match, matchResult)
	assert.Nil(t, err)
	match, _ = db.GetMatchById(1)
	assert.Equal(t, "T", match.Winner)
	match.Type = "elimination"
	db.SaveMatch(match)
	CommitMatchScore(match, matchResult)
	match, _ = db.GetMatchById(1)
	assert.Equal(t, "B", match.Winner)
}

func TestMatchPlayWebsocketCommands(t *testing.T) {
	clearDb()
	defer clearDb()
	var err error
	db, err = OpenDatabase(testDbPath)
	assert.Nil(t, err)
	defer db.Close()
	db.CreateTeam(&Team{Id: 254})
	mainArena.Setup()

	server, wsUrl := startTestServer()
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial(wsUrl+"/match_play/websocket", nil)
	assert.Nil(t, err)
	defer conn.Close()
	ws := &Websocket{conn}

	// Should get a few status updates right after connection.
	readWebsocketType(t, ws, "status")
	readWebsocketType(t, ws, "matchTiming")
	readWebsocketType(t, ws, "matchTime")
	readWebsocketType(t, ws, "setAudienceDisplay")
	readWebsocketType(t, ws, "scoringStatus")

	// Test that a server-side error is communicated to the client.
	ws.Write("nonexistenttype", nil)
	assert.Contains(t, readWebsocketError(t, ws), "Invalid message type")

	// Test match setup commands.
	ws.Write("substituteTeam", nil)
	assert.Contains(t, readWebsocketError(t, ws), "Invalid alliance station")
	ws.Write("substituteTeam", map[string]interface{}{"team": 254, "position": "B5"})
	assert.Contains(t, readWebsocketError(t, ws), "Invalid alliance station")
	ws.Write("substituteTeam", map[string]interface{}{"team": 254, "position": "B1"})
	readWebsocketType(t, ws, "status")
	assert.Equal(t, 254, mainArena.currentMatch.Blue1)
	ws.Write("substituteTeam", map[string]interface{}{"team": 0, "position": "B1"})
	readWebsocketType(t, ws, "status")
	assert.Equal(t, 0, mainArena.currentMatch.Blue1)
	ws.Write("toggleBypass", nil)
	assert.Contains(t, readWebsocketError(t, ws), "Failed to parse")
	ws.Write("toggleBypass", "R4")
	assert.Contains(t, readWebsocketError(t, ws), "Invalid alliance station")
	ws.Write("toggleBypass", "R3")
	readWebsocketType(t, ws, "status")
	assert.Equal(t, true, mainArena.AllianceStations["R3"].Bypass)
	ws.Write("toggleBypass", "R3")
	readWebsocketType(t, ws, "status")
	assert.Equal(t, false, mainArena.AllianceStations["R3"].Bypass)

	// Go through match flow.
	ws.Write("abortMatch", nil)
	assert.Contains(t, readWebsocketError(t, ws), "Cannot abort match")
	ws.Write("startMatch", nil)
	assert.Contains(t, readWebsocketError(t, ws), "Cannot start match")
	mainArena.AllianceStations["R1"].Bypass = true
	mainArena.AllianceStations["R2"].Bypass = true
	mainArena.AllianceStations["R3"].Bypass = true
	mainArena.AllianceStations["B1"].Bypass = true
	mainArena.AllianceStations["B2"].Bypass = true
	mainArena.AllianceStations["B3"].Bypass = true
	ws.Write("startMatch", nil)
	readWebsocketType(t, ws, "status")
	assert.Equal(t, START_MATCH, mainArena.MatchState)
	ws.Write("commitResults", nil)
	assert.Contains(t, readWebsocketError(t, ws), "Cannot reset match")
	ws.Write("discardResults", nil)
	assert.Contains(t, readWebsocketError(t, ws), "Cannot reset match")
	ws.Write("abortMatch", nil)
	readWebsocketType(t, ws, "status")
	readWebsocketType(t, ws, "setAudienceDisplay")
	assert.Equal(t, POST_MATCH, mainArena.MatchState)
	mainArena.redRealtimeScore.CurrentScore.AutoHighHot = 29
	mainArena.blueRealtimeScore.CurrentScore.AutoHighHot = 37
	ws.Write("commitResults", nil)
	readWebsocketType(t, ws, "reload")
	assert.Equal(t, 29, mainArena.savedMatchResult.RedScore.AutoHighHot)
	assert.Equal(t, 37, mainArena.savedMatchResult.BlueScore.AutoHighHot)
	assert.Equal(t, PRE_MATCH, mainArena.MatchState)
	ws.Write("discardResults", nil)
	readWebsocketType(t, ws, "reload")
	assert.Equal(t, PRE_MATCH, mainArena.MatchState)

	// Test changing the audience display.
	ws.Write("setAudienceDisplay", "logo")
	readWebsocketType(t, ws, "setAudienceDisplay")
	assert.Equal(t, "logo", mainArena.audienceDisplayScreen)
}

func TestMatchPlayWebsocketNotifications(t *testing.T) {
	clearDb()
	defer clearDb()
	var err error
	db, err = OpenDatabase(testDbPath)
	assert.Nil(t, err)
	defer db.Close()
	db.CreateTeam(&Team{Id: 254})
	mainArena.Setup()

	server, wsUrl := startTestServer()
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial(wsUrl+"/match_play/websocket", nil)
	assert.Nil(t, err)
	defer conn.Close()
	ws := &Websocket{conn}

	// Should get a few status updates right after connection.
	readWebsocketType(t, ws, "status")
	readWebsocketType(t, ws, "matchTiming")
	readWebsocketType(t, ws, "matchTime")
	readWebsocketType(t, ws, "setAudienceDisplay")
	readWebsocketType(t, ws, "scoringStatus")

	mainArena.AllianceStations["R1"].Bypass = true
	mainArena.AllianceStations["R2"].Bypass = true
	mainArena.AllianceStations["R3"].Bypass = true
	mainArena.AllianceStations["B1"].Bypass = true
	mainArena.AllianceStations["B2"].Bypass = true
	mainArena.AllianceStations["B3"].Bypass = true
	mainArena.StartMatch()
	mainArena.Update()
	messages := readWebsocketMultiple(t, ws, 3)
	statusReceived, matchTime := getStatusMatchTime(t, messages)
	assert.Equal(t, true, statusReceived)
	assert.Equal(t, 2, matchTime.MatchState)
	assert.Equal(t, 0, matchTime.MatchTimeSec)
	_, ok := messages["setAudienceDisplay"]
	assert.True(t, ok)
	mainArena.scoringStatusNotifier.Notify(nil)
	readWebsocketType(t, ws, "scoringStatus")

	// Should get a tick notification when an integer second threshold is crossed.
	mainArena.matchStartTime = time.Now().Add(-time.Second + 10*time.Millisecond) // Not crossed yet
	mainArena.Update()
	mainArena.matchStartTime = time.Now().Add(-time.Second - 10*time.Millisecond) // Crossed
	mainArena.Update()
	mainArena.matchStartTime = time.Now().Add(-2*time.Second + 10*time.Millisecond) // Not crossed yet
	mainArena.Update()
	mainArena.matchStartTime = time.Now().Add(-2*time.Second - 10*time.Millisecond) // Crossed
	mainArena.Update()
	err = mapstructure.Decode(readWebsocketType(t, ws, "matchTime"), &matchTime)
	assert.Nil(t, err)
	assert.Equal(t, 2, matchTime.MatchState)
	assert.Equal(t, 1, matchTime.MatchTimeSec)
	err = mapstructure.Decode(readWebsocketType(t, ws, "matchTime"), &matchTime)
	assert.Nil(t, err)
	assert.Equal(t, 2, matchTime.MatchState)
	assert.Equal(t, 2, matchTime.MatchTimeSec)

	// Check across a match state boundary.
	mainArena.matchStartTime = time.Now().Add(-time.Duration(mainArena.matchTiming.AutoDurationSec) * time.Second)
	mainArena.Update()
	statusReceived, matchTime = readWebsocketStatusMatchTime(t, ws)
	assert.Equal(t, true, statusReceived)
	assert.Equal(t, 3, matchTime.MatchState)
	assert.Equal(t, mainArena.matchTiming.AutoDurationSec, matchTime.MatchTimeSec)
}

// Handles the status and matchTime messages arriving in either order.
func readWebsocketStatusMatchTime(t *testing.T, ws *Websocket) (bool, MatchTimeMessage) {
	return getStatusMatchTime(t, readWebsocketMultiple(t, ws, 2))
}

func getStatusMatchTime(t *testing.T, messages map[string]interface{}) (bool, MatchTimeMessage) {
	_, statusReceived := messages["status"]
	message, ok := messages["matchTime"]
	var matchTime MatchTimeMessage
	if assert.True(t, ok) {
		err := mapstructure.Decode(message, &matchTime)
		assert.Nil(t, err)
	}
	return statusReceived, matchTime
}