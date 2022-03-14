package datasource

import "time"

func Data() (interface{}, error) {
	return struct {
		Pong   bool      `json:"pong"`
		Reason string    `json:"reason"`
		Date   time.Time `json:"date"`
		Quote  string    `json:"quote"`
	}{
		true,
		"the bug does not seem to be here :( where is it then? who knows? at this point, i have no clue! i'm lost now. i have no clue what to do. i need help. i don't know what i'm doing. where should i look for help? where could the bug be? is there even a bug? right now, i'm questioning everything!",
		time.Now(),
		"dumbass! (that 70's show)",
	}, nil
}
