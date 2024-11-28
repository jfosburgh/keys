package words

import (
	_ "embed"
	"math/rand"
	"strings"
)

const LETTERS = "eniarltosudycghpmkbwfzvxqj"

//go:embed wordlists/20k.txt
var words20kString string

type WordList []string

var (
	Words20k WordList
)

func InitWordLists() {
	Words20k = strings.Split(words20kString, "\n")
}

func (w WordList) TopK(k int) WordList {
	return w[:min(k, len(w))]
}

func (w WordList) LongerThan(l int) WordList {
	newList := WordList{}

	for _, word := range w {
		if len(word) >= l {
			newList = append(newList, word)
		}
	}

	return newList
}

func (w WordList) FilterOutLetter(letterList []string) WordList {
	newList := WordList{}

	letters := strings.Join(letterList, "")

	for _, word := range w {
		if !strings.ContainsAny(word, letters) {
			newList = append(newList, word)
		}
	}

	return newList
}

func (w WordList) TakeWords(length int) string {
	words := []string{}
	upper := len(w)

	for range length {
		words = append(words, w[rand.Intn(upper)])
	}

	return strings.Join(words, " ")
}

func (w WordList) TakeChars(length int) string {
	words := ""
	upper := len(w)

	for len(words) < length-3 {
		words += w[rand.Intn(upper)] + " "
	}

	return words[:len(words)-1]
}
