package myhandlers

import (
	"context"
	"main/globalcfg"
	"math/rand"
)

func getRandomPicByRate(rate int) string {
	r, err := g.Q().GetPicByUserRate(context.Background(), rate)
	if err != nil {
		log.Errorf("getRandomPicByRate err:%v", err)
	}
	return r
}

func getRandomNsfwAdult() string {
	return getRandomPicByRate(6)
}

func getRandomNsfwRacy() string {
	if rand.Int()%2 == 0 {
		return getRandomPicByRate(4)
	} else {
		return getRandomPicByRate(4)
	}
}
