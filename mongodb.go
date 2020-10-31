package main

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ConnectMongo() {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb+srv://dbAdmin:qjf314pi@cluster0-6y9s1.azure.mongodb.net/test?retryWrites=true"))

	if err != nil {
		log.Fatal(err)
	}

	// Check the connection
	err = client.Ping(context.Background(), nil)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected to MongoDB!")

	// handle for the trainers collection in the test database
	// collection := client.Database("test").Collection("trainers")

	// Close the connection
	err = client.Disconnect(context.TODO())

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connection to MongoDB closed.")
}
