package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoInstance struct {
	Client *mongo.Client
	Db     *mongo.Database
}

var mg MongoInstance

const dbName = "fiber-hrms"
const mongoURI = "mongodb://localhost:27017/" + dbName

type Employee struct {
	ID     string  `json:"id,omitempty" bson:"_id,omitempty`
	Name   string  `json:"name"`
	Salary float64 `json:"salary"`
	Age    float64 `json:"age"`
}

func Connect() error {
	// using this below line we can create an new uri
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	db := client.Database(dbName)

	if err != nil {
		return err
	}

	mg = MongoInstance{
		Client: client,
		Db:     db,
	}
	return nil

}

func main() {

	if err := Connect(); err != nil {
		log.Fatal(err)
	}
	app := fiber.New()

	app.Get("/employee", func(c *fiber.Ctx) error {

		// because of empty query we use empty brackets

		query := bson.D{{}}

		/*This initializes an empty MongoDB query in BSON format.
		BSON is a binary format used to serialize documents in MongoDB. */

		//inside the mongodb(mg) you go to database(Db). Collections is like tables and run the find command
		//we'll get the all employees record from cursor
		cursor, err := mg.Db.Collection("employees").Find(c.Context(), query)

		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		var employees []Employee = make([]Employee, 0)

		/* whatever data we recieve in cursor which is all the data of employee which
		is going to convert the format of struct which the golang is understandable in end we get employees but
		that will we a slice of multiple employees we did this because golang cant undersand which mongodb send do we convert to struct */

		cursor.All(c.Context(), &employees)

		// here we send a response as json to the frontend

		return c.JSON(employees)
	})

	app.Post("/employee", func(c *fiber.Ctx) error {
		collection := mg.Db.Collection("employees")

		employee := new(Employee)

		// it parses the body that db send by user probably not sure and converts it into the thing that golang understoods
		if err := c.BodyParser(employee); err != nil {
			return c.Status(400).SendString(err.Error())
		}

		employee.ID = ""

		// Here we insert a data from the user to the database using InserOne query
		insertionResult, err := collection.InsertOne(c.Context(), employee)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		// the insertionResult has an insertedID has the id of the data inserted we keep that to build the query the filter is used to find a function to recheck the data in created
		filter := bson.D{{Key: "_id", Value: insertionResult.InsertedID}}
		//  CreatedRecord is the record created by you and you send that data to the frontend
		createdRecord := collection.FindOne(c.Context(), filter)

		// it is variable of type employee
		createdEmployee := &Employee{}
		// we want to decode the created employee we need to decode cuz go donesnt understand the json
		createdRecord.Decode(createdEmployee)

		return c.Status(201).JSON(createdEmployee)
	})

	app.Put("/employee/:id", func(c *fiber.Ctx) error {
		idParam := c.Params("id")

		employeeID, err := primitive.ObjectIDFromHex(idParam)

		if err != nil {
			return c.SendStatus(400)
		}

		employee := new(Employee)

		if err := c.BodyParser(employee); err != nil {
			return c.Status(400).SendString(err.Error())
		}

		query := bson.D{{Key: "_id", Value: employeeID}}

		// Update query $set to set the new data instead of old data $set is used to set the data that you are replacing

		update := bson.D{
			{Key: "$set",
				Value: bson.D{
					{Key: "name", Value: employee.Name},
					{Key: "age", Value: employee.Age},
					{Key: "salary", Value: employee.Salary},
				},
			},
		}

		err = mg.Db.Collection("employees").FindOneAndUpdate(c.Context(), query, update).Err()

		if err != nil {
			if err == mongo.ErrNoDocuments {
				return c.SendStatus(400)
			}
			return c.SendStatus(500)
		}

		employee.ID = idParam

		return c.Status(200).JSON(employee)
	})
	app.Delete("/employee/:id", func(c *fiber.Ctx) error {

		employeeID, err := primitive.ObjectIDFromHex(c.Params("id"))

		if err != nil {
			return c.SendStatus(400)
		}

		query := bson.D{{Key: "_id", Value: employeeID}}
		result, err := mg.Db.Collection("employees").DeleteOne(c.Context(), &query)

		if err != nil {
			return c.SendStatus(500)
		}

		if result.DeletedCount < 1 {
			return c.SendStatus(404)
		}

		return c.Status(200).JSON("Record Deleted")
	})

	log.Fatal(app.Listen(":3000"))
}
