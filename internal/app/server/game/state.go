package server

import (
	"errors"
	"log"
)

type GameState struct {
	s   *GameServer
	Day bool

	VotedTotal      int
	Votes           []int
	VotesCount      []int
	VoteChan        chan int
	Killed          int
	KillChan        chan int
	Checked         int
	CheckChan       chan int
	CheckChanClosed bool
}

func (s *GameState) Close() {
	close(s.VoteChan)
	close(s.KillChan)
	if !s.CheckChanClosed {
		close(s.CheckChan)
	}
}

func CreateGameState(s *GameServer) *GameState {
	return &GameState{
		s:               s,
		Day:             true,
		VotedTotal:      0,
		Votes:           make([]int, 4),
		VotesCount:      make([]int, 4),
		VoteChan:        make(chan int),
		Killed:          -1,
		KillChan:        make(chan int, 1),
		Checked:         -1,
		CheckChan:       make(chan int, 1),
		CheckChanClosed: false,
	}
}

func (s *GameState) getAliveCnt() int {
	res := 0
	for _, p := range s.s.players {
		if p.Alive {
			res += 1
		}
	}
	return res
}

func (s *GameState) SetupNewDay() {
	s.Day = true
	for i := range s.Votes {
		s.Votes[i] = -100
		s.VotesCount[i] = 0
	}
	s.VotedTotal = 0
}

func (s *GameState) SetupNewNight() {
	s.Day = false
	s.Killed = -1
	s.Checked = -1
}

func (s *GameState) TryFinishVote() {
	if s.Day && s.VotedTotal == s.getAliveCnt() {
		voted_off := 0
		for i := range s.VotesCount {
			if s.VotesCount[i] > s.VotesCount[voted_off] {
				voted_off = i
			}
		}
		s.VoteChan <- voted_off
	}
}

func (s *GameState) Vote(voter int, voting int) error {
	if s.Votes[voter] != -100 {
		return errors.New("Уже голосовал!")
	}
	if !s.Day {
		return errors.New("Vote only at day time!")
	}

	s.Votes[voter] = voting
	s.VotedTotal += 1
	if voting != -1 {
		s.VotesCount[voting] += 1
	}
	s.TryFinishVote()
	return nil
}

func (s *GameState) Kill(killer int, killing int) error {
	if s.Killed != -1 {
		return errors.New("Уже убивал!")
	}
	if s.Day {
		return errors.New("Kill only at night time!")
	}

	s.Killed = killing
	s.KillChan <- killing
	return nil
}

func (s *GameState) Check(checker int, checking int) error {
	log.Println("Check state...")
	if s.Checked != -1 {
		return errors.New("Уже проверял!")
	}
	if s.Day {
		return errors.New("Check only at night time!")
	}

	s.Checked = checking
	log.Printf("chan is %v", s.CheckChan)
	s.CheckChan <- checking
	return nil
}
