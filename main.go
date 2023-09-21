package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/plugins/executor"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type operations = string

const (
	InsertOne operations = "insertOne"
	DeleteOne operations = "deleteOne"
	UpdateOne operations = "updateOne"
	DBTimeout            = 10 * time.Second
)

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.POST("/api/v1/template.execute", func(c echo.Context) error {

		request := new(executor.ExecuteTemplateArgs)
		// get body from request
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return fmt.Errorf("Error reading body: %v", err)
		}
		// unmarshal body into request
		err = json.Unmarshal(body, request)
		if err != nil {
			return fmt.Errorf("Error unmarshalling body: %v", err)
		}

		pluginPayload := make(map[string]interface{}, 0)
		err = json.Unmarshal(request.Template.Plugin.Value, &pluginPayload)
		if err != nil {
			return fmt.Errorf("Error unmarshalling plugin payload: %v", err)
		}
		operation := pluginPayload["operation"].(operations)
		connectionURI, ok := pluginPayload["connectionURI"].(string)
		if !ok {
			return fmt.Errorf("Error getting connectionURI from plugin payload: %v", err)
		}
		database, ok := pluginPayload["database"].(string)
		if !ok {
			return fmt.Errorf("Error getting database from plugin payload: %v", err)
		}
		collection, ok := pluginPayload["collection"].(string)
		if !ok {
			return fmt.Errorf("Error getting collection from plugin payload: %v", err)
		}

		ctx := c.Request().Context()
		message := ""
		insertedID := ""
		var dbCollection *mongo.Collection

		switch operation {
		case InsertOne:
			client, err := mongo.Connect(ctx, options.Client().ApplyURI(connectionURI))
			if err != nil {
				return fmt.Errorf("Error connecting to database: %v", err)
			}
			defer client.Disconnect(ctx)
			dbCollection = client.Database(database).Collection(collection)
			resp, err := dbCollection.InsertOne(ctx, pluginPayload["document"])
			if err != nil {
				return fmt.Errorf("Error inserting document: %v", err)
			}
			insertedID = resp.InsertedID.(primitive.ObjectID).Hex()
			message = fmt.Sprintf("Inserted document with ID: %s", insertedID)
		case DeleteOne:
			client, err := mongo.Connect(ctx, options.Client().ApplyURI(connectionURI))
			if err != nil {
				return fmt.Errorf("Error connecting to database: %v", err)
			}
			dbCollection = client.Database(database).Collection(collection)
			_, err = dbCollection.DeleteOne(ctx, bson.M{"_id": pluginPayload["id"].(string)})
			if err != nil {
				return fmt.Errorf("Error deleting document: %v", err)
			}
			message = fmt.Sprintf("Deleted document with ID: %s", pluginPayload["id"].(string))
		case UpdateOne:
			client, err := mongo.Connect(ctx, options.Client().ApplyURI(connectionURI))
			if err != nil {
				return fmt.Errorf("Error connecting to database: %v", err)
			}
			dbCollection = client.Database(database).Collection(collection)
			_, err = dbCollection.UpdateOne(ctx, bson.M{"_id": pluginPayload["id"].(string)}, bson.M{"$set": pluginPayload["update"]})
			if err != nil {
				return fmt.Errorf("Error updating document: %v", err)
			}
			message = fmt.Sprintf("Updated document with ID: %s", pluginPayload["id"].(string))
		}

		response := executor.ExecuteTemplateResponse{}
		if len(insertedID) > 0 {
			response.Body = executor.ExecuteTemplateReply{
				Node: &v1alpha1.NodeResult{
					Phase:   "Succeeded",
					Message: message,
					Outputs: &v1alpha1.Outputs{
						Parameters: []v1alpha1.Parameter{
							{
								Name:  "insertedID",
								Value: v1alpha1.AnyStringPtr(insertedID),
							},
						},
					},
				},
			}
		} else {
			response.Body = executor.ExecuteTemplateReply{
				Node: &v1alpha1.NodeResult{
					Phase:   "Succeeded",
					Message: message,
				},
			}
		}
		return c.JSON(http.StatusOK, response)
	})

	e.Logger.Fatal(e.Start(":30005"))
}
