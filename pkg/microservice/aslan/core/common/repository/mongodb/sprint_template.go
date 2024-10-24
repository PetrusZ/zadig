/*
Copyright 2022 The KodeRover Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/koderover/zadig/v2/pkg/microservice/aslan/config"
	"github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/models"
	"github.com/koderover/zadig/v2/pkg/shared/handler"
	mongotool "github.com/koderover/zadig/v2/pkg/tool/mongo"
)

type SprintTemplateQueryOption struct {
	ID          string
	Name        string
	ProjectName string
}

type SprintTemplateListOption struct {
	ProjectName string
}

type SprintTemplateColl struct {
	*mongo.Collection
	mongo.Session

	coll string
}

func NewSprintTemplateColl() *SprintTemplateColl {
	name := models.SprintTemplate{}.TableName()
	return &SprintTemplateColl{
		Collection: mongotool.Database(config.MongoDatabase()).Collection(name),
		coll:       name,
	}
}

func NewSprintTemplateCollWithSession(session mongo.Session) *SprintTemplateColl {
	name := models.SprintTemplate{}.TableName()
	return &SprintTemplateColl{
		Collection: mongotool.Database(config.MongoDatabase()).Collection(name),
		Session:    session,
		coll:       name,
	}
}

func (c *SprintTemplateColl) GetCollectionName() string {
	return c.coll
}

func (c *SprintTemplateColl) EnsureIndex(ctx context.Context) error {
	index := mongo.IndexModel{
		Keys: bson.D{
			bson.E{Key: "project_name", Value: 1},
			bson.E{Key: "key", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}
	_, err := c.Indexes().CreateOne(mongotool.SessionContext(ctx, c.Session), index)

	return err
}

func (c *SprintTemplateColl) Create(ctx *handler.Context, obj *models.SprintTemplate) error {
	if obj == nil {
		return fmt.Errorf("nil object")
	}

	obj.CreateTime = time.Now().Unix()
	obj.UpdateTime = time.Now().Unix()
	_, err := c.InsertOne(mongotool.SessionContext(ctx, c.Session), obj)
	return err
}

func (c *SprintTemplateColl) Update(ctx *handler.Context, obj *models.SprintTemplate) error {
	query := bson.M{"_id": obj.ID}
	change := bson.M{"$set": obj}
	obj.UpdateTime = time.Now().Unix()
	_, err := c.UpdateOne(mongotool.SessionContext(ctx, c.Session), query, change)
	return err
}

func (c *SprintTemplateColl) UpsertByName(ctx *handler.Context, obj *models.SprintTemplate) error {
	query := bson.M{"name": obj.Name}
	change := bson.M{"$set": obj}
	obj.UpdateTime = time.Now().Unix()
	_, err := c.UpdateOne(mongotool.SessionContext(ctx, c.Session), query, change, options.Update().SetUpsert(true))
	return err
}

func (c *SprintTemplateColl) GetByID(ctx *handler.Context, idStr string) (*models.SprintTemplate, error) {
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return nil, err
	}
	query := bson.M{
		"_id": id,
	}

	sprintTemplate := new(models.SprintTemplate)
	if err := c.Collection.FindOne(mongotool.SessionContext(ctx, c.Session), query).Decode(sprintTemplate); err != nil {
		return nil, err
	}

	return sprintTemplate, nil
}

func (c *SprintTemplateColl) Find(ctx *handler.Context, opt *SprintTemplateQueryOption) (*models.SprintTemplate, error) {
	if opt == nil {
		return nil, errors.New("nil FindOption")
	}
	query := bson.M{}
	if len(opt.ID) > 0 {
		id, err := primitive.ObjectIDFromHex(opt.ID)
		if err != nil {
			return nil, err
		}
		query["_id"] = id
	}
	if len(opt.Name) > 0 {
		query["name"] = opt.Name
	}
	if len(opt.ProjectName) > 0 {
		query["project_name"] = opt.Name
	}
	resp := new(models.SprintTemplate)
	err := c.Collection.FindOne(mongotool.SessionContext(ctx, c.Session), query).Decode(resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *SprintTemplateColl) List(ctx *handler.Context, option *SprintTemplateListOption) ([]*models.SprintTemplate, error) {
	resp := make([]*models.SprintTemplate, 0)
	query := bson.M{}
	if option.ProjectName != "" {
		query["project_name"] = option.ProjectName
	}
	opt := options.Find().
		SetSort(bson.D{{"_name", -1}})

	cursor, err := c.Collection.Find(mongotool.SessionContext(ctx, c.Session), query, opt)
	if err != nil {
		return nil, err
	}
	err = cursor.All(mongotool.SessionContext(ctx, c.Session), &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *SprintTemplateColl) DeleteByID(ctx *handler.Context, idStr string) error {
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return err
	}
	query := bson.M{"_id": id}
	_, err = c.DeleteOne(mongotool.SessionContext(ctx, c.Session), query)
	return err
}
