package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"strings"
	"time"
)

type repository interface {
	getBreakfast(ctx context.Context, username string, breakfastID uint64) (breakfast, error)
	getRandomBreakfast(ctx context.Context, username string) (breakfast, error)
}

type breakfast struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Image       string `json:"image"`
	Description string `json:"description"`
}

type breakfasts []breakfast

func newRepository(filename string) (a breakfasts, err error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return a, json.Unmarshal(buf, &a)
}

func mustNewRepository(filename string) breakfasts {
	a, err := newRepository(filename)
	if err != nil {
		panic(err)
	}
	return a
}

func (a breakfasts) getBreakfast(_ context.Context, username string, breakfastID uint64) (breakfast, error) {
	fakeDatabaseOperation(username)
	for _, b := range a {
		if b.ID == breakfastID {
			return b, nil
		}
	}
	return breakfast{}, fmt.Errorf("no breakfast with ID %d", breakfastID)
}

func (a breakfasts) getRandomBreakfast(_ context.Context, username string) (breakfast, error) {
	fakeDatabaseOperation(username)
	if len(a) <= 0 {
		return breakfast{}, errors.New("no breakfasts available")
	}
	return a[rand.Intn(len(a))], nil
}

func fakeDatabaseOperation(username string) {
	var shardDelay time.Duration
	{
		var min, max int
		switch strings.ToLower(username)[0] {
		case 'a', 'b', 'c', 'd', 'e':
			min, max = 15, 35
		case 'f', 'g', 'h', 'i', 'j':
			min, max = 20, 40
		case 'k', 'l', 'm', 'n', 'o':
			min, max = 150, 300
		case 'p', 'q', 'r', 's', 't':
			min, max = 60, 80
		case 'u', 'v', 'w', 'x', 'y':
			min, max = 10, 30
		default:
			min, max = 10, 20
		}
		shardDelay = time.Duration(rand.Intn(max-min)+min) * time.Millisecond
	}
	var clockDelay time.Duration
	{
		if time.Now().Format("04") == "00" {
			clockDelay = 300 * time.Millisecond
		}
	}
	time.Sleep(shardDelay + clockDelay)
}
