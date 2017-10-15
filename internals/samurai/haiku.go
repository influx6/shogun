package samurai

import "math/rand"

var (
	haikus = []string{
		"Wielding his great sword he stands.",
		"For honour and country he fights.",
		"A hero who fights with pride.",
		"A hero who slays with honor.",
		"He is a brave soul, he stands for peace.",
		"Always for peace, never for vengeance.",
		"In the midst of battle, his name stays on.",
	}
)

func haiku() string {
	hlen := len(haikus)
	index := rand.Intn(hlen)
	return haikus[index]
}
