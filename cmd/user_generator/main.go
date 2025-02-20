package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/goombaio/namegenerator"
	"github.com/rs/zerolog/log"
)

const (
	timeInterval           = 1
	numberOfDifferentUsers = 1000
	topic                  = "users"
	ageMin                 = 15
	ageMax                 = 100
	defaultAge             = 45
	numberOfGenders        = 2
	defaultGender          = "male"
	kafkaProducer          = "localhost:9094"
	deletedRandNum         = 100
	deleteUsersPercent     = 5
)

type User struct {
	UserID    int    `json:"userid"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Gender    string `json:"gender"`
	Age       int    `json:"age"`
	Deleted   bool   `json:"deleted"`
}

func main() {
	log.Info().Msg("Attempting to connect to Kafka broker as producer at localhost:9094 ...")

	producer, err := newKafkaProducer([]string{kafkaProducer})
	if err != nil {
		log.Error().Err(err).Msg("failed to create sync producer")

		return
	}

	defer func() {
		if err = producer.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close kafka producer")
		}
	}()

	users := generateUsers(numberOfDifferentUsers)

	for {
		user, err := getRandomUser(users)
		if err != nil {
			log.Panic().Err(err).Msg("failed to get random user")
		}

		if err := sendUserToKafka(producer, topic, user); err != nil {
			log.Panic().Err(err).Msg("failed to send user to kafka")
		}

		log.Info().Msg("message sent")
		time.Sleep(timeInterval * time.Second)
	}
}

func generateUsers(count int) []User {
	users := make([]User, count)

	for i := range count {
		newUser := generateUser()
		users[i] = newUser
	}

	return users
}

func generateUser() User {
	seed := time.Now().UTC().UnixNano()
	nameGenerator := namegenerator.NewNameGenerator(seed)
	fullName := nameGenerator.Generate()
	names := strings.Split(fullName, "-")

	gender, err := generateGender()
	if err != nil {
		log.Error().Err(err).Msg("error in generateUser() function: gender, using default")

		gender = defaultGender
	}

	age, err := generateAge(ageMin, ageMax)
	if err != nil {
		log.Error().Err(err).Msg("error in generateUser() function: gender, using default")

		age = defaultAge
	}

	userID, err := rand.Int(rand.Reader, big.NewInt(int64(numberOfDifferentUsers)))
	if err != nil {
		log.Error().Err(err).Msg("error generating random user ID")
	}

	deleted, err := randonDeleted()
	if err != nil {
		log.Error().Err(err).Msg("error generating deleted value for user")
	}

	return User{
		UserID:    int(userID.Int64()),
		FirstName: capitalizeFirstLetter(names[0]),
		LastName:  capitalizeFirstLetter(names[1]),
		Gender:    gender,
		Age:       age,
		Deleted:   deleted,
	}
}

func capitalizeFirstLetter(name string) string {
	result := strings.ToUpper(string(name[0])) + name[1:]

	return result
}

func generateGender() (string, error) {
	randomBigInt, err := rand.Int(rand.Reader, big.NewInt(int64(numberOfGenders)))
	if err != nil {
		return "", fmt.Errorf("gender generation error: %w", err)
	}

	if int(randomBigInt.Int64()) == 0 {
		return "female", nil
	}

	return "male", nil
}

func generateAge(minVal, maxVal int) (int, error) {
	rangeSize := maxVal - minVal + 1

	randomBigInt, err := rand.Int(rand.Reader, big.NewInt(int64(rangeSize)))
	if err != nil {
		return 0, fmt.Errorf("age generation error: %w", err)
	}

	return minVal + int(randomBigInt.Int64()), nil
}

func getRandomUser(users []User) (User, error) {
	index, err := rand.Int(rand.Reader, big.NewInt(int64(len(users))))
	if err != nil {
		return User{}, fmt.Errorf("random user selection error: %w", err)
	}

	return users[int(index.Int64())], nil
}

func newKafkaProducer(brokers []string) (sarama.SyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return producer, fmt.Errorf("error creating producer in newKafkaProducer(): %w", err)
	}

	return producer, nil
}

func sendUserToKafka(producer sarama.SyncProducer, topic string, user User) error {
	bytes, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	message := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(bytes),
	}

	_, _, err = producer.SendMessage(message)
	if err != nil {
		return fmt.Errorf("error sending message to Kafka: %w", err)
	}

	return nil
}

func randonDeleted() (bool, error) {
	randNum, err := rand.Int(rand.Reader, big.NewInt(int64(deletedRandNum)))
	if err != nil {
		return false, fmt.Errorf("random number generation error when creating metric called deleted: %w", err)
	}

	num := int(randNum.Int64()) % deletedRandNum

	if num < deleteUsersPercent {
		return true, nil
	}

	return false, nil
}
