package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"gopkg.in/jdkato/prose.v2"
)

/*
Lexicon format:
{
  "word": {
    "next": freq
  }
}
*/

// Lexicon type
type Lexicon = map[string]map[string]int

var lexicon Lexicon

const lexiconFile = "/tmp/lexicon"

func main() {
	// Initialize
	rand.Seed(time.Now().Unix())
	lexicon = make(Lexicon)
	loadLex()

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TOKEN"))

	if err != nil {
		log.Printf("Error!")
		log.Printf(err.Error())
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		// Clean up message
		messageCleaned := strings.ReplaceAll(update.Message.Text, "@"+bot.Self.UserName, "")
		messageCleaned = strings.ToLower(messageCleaned)

		// Tokenize input
		doc, _ := prose.NewDocument(messageCleaned)
		// Learn from input
		learn(doc.Tokens())

		// Only respond if in private chat or mentioned
		if !update.Message.Chat.IsPrivate() &&
			!((update.Message.Chat.IsGroup() || update.Message.Chat.IsSuperGroup()) &&
				strings.Contains(update.Message.Text, bot.Self.UserName)) {
			continue
		}

		// Generate response
		response := generateResponse(doc.Tokens())

		// Create Message object
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, response)

		// Only reply to message outside of private chats
		if !update.Message.Chat.IsPrivate() {
			msg.ReplyToMessageID = update.Message.MessageID
		}

		bot.Send(msg)
	}
}

func generateResponse(tokens []prose.Token) string {
	// Limit response to 256 words
	responseLength := 256
	responseArr := make([]string, responseLength)
	generateUntil := []string{".", ",", "?", "!"}

	// Choose random token from array
	responseArr[0] = tokens[rand.Intn(len(tokens))].Text

	stop := false
	for i := 1; i < responseLength; i++ {
		responseArr[i] = getNextWord(responseArr[i-1])

		for _, elem := range generateUntil {
			if elem == responseArr[i] {
				stop = true
			}
		}
		if stop || len(responseArr[i]) == 0 {
			break
		}
	}

	return normalizeResponse(strings.Join(responseArr, " "))
}

func normalizeResponse(text string) string {
	noSpacePrefix := []string{"n't", ".", ",", "?", "!", "\"", "'s", "'ll", "'re", ")"}
	noSpaceSuffix := []string{")"}

	for _, elem := range noSpacePrefix {
		text = strings.ReplaceAll(text, " "+elem, elem)
	}
	for _, elem := range noSpaceSuffix {
		text = strings.ReplaceAll(text, elem+" ", elem)
	}

	return text
}

func getNextWord(word string) string {
	if assoc, ok := lexicon[word]; ok {
		// Word associations exist
		// Generate a cumulative array
		words := make([]string, len(assoc))
		weights := make([]int, len(assoc))
		index := 0

		for key, value := range assoc {
			prevWeight := 0
			if index > 0 {
				prevWeight = weights[index-1]
			}

			words[index] = key
			weights[index] = value + prevWeight
			index++
		}

		// Create a random number
		randResult := rand.Intn(weights[index-1])

		// Find the selected word
		for i := 0; i < len(weights); i++ {
			if weights[i] > randResult {
				// Word found
				return words[i]
			}
		}

		// If all else fails
		return words[0]
	}
	return ""
}

func learn(tokens []prose.Token) {
	// Loop through the tokens
	for index, token := range tokens {
		// Only loop until the second last token
		if index == len(tokens)-1 {
			continue
		}

		nextToken := tokens[index+1]

		// Check if lexicon contains the token
		if assoc, ok := lexicon[token.Text]; ok {
			// Lexicon contains token as key
			// Check if map of next words contains the next word
			if _, ok := assoc[nextToken.Text]; ok {
				// Map contains next word, add one to the count
				lexicon[token.Text][nextToken.Text]++
			} else {
				// Map does not contain the next word, initialize with 1
				lexicon[token.Text][nextToken.Text] = 1
			}
		} else {
			// Lexicon does not contain token as key
			// Create the new key and initialize with next word association
			assoc := make(map[string]int)
			assoc[nextToken.Text] = 1
			lexicon[token.Text] = assoc
		}
	}

	// Save learning outcome to file
	saveLex()
}

func saveLex() {
	jsonLex, _ := json.Marshal(lexicon)
	err := ioutil.WriteFile(lexiconFile, jsonLex, 0644)
	check(err)
}

func loadLex() {
	dat, _ := ioutil.ReadFile(lexiconFile)
	json.Unmarshal(dat, &lexicon)
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
