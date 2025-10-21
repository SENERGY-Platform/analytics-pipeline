/*
 * Copyright 2018 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package lib

import (
	"time"

	"github.com/google/uuid"
)

type Response struct {
	Message string `json:"message,omitempty"`
}

type PipelineResponse struct {
	Id uuid.UUID `json:"id,omitempty"`
}

type PipelinesResponse struct {
	Data  []Pipeline `json:"data"`
	Total int64      `json:"total"`
}

type Pipeline struct {
	Id                 string    `bson:"id" json:"id"`
	Name               string    `json:"name,omitempty"`
	Description        string    `json:"description,omitempty"`
	FlowId             string    `json:"flowId,omitempty"`
	Image              string    `json:"image,omitempty"`
	WindowTime         int       `json:"windowTime,omitempty"`
	MergeStrategy      string    `json:"mergeStrategy,omitempty"`
	ConsumeAllMessages bool      `json:"consumeAllMessages,omitempty"`
	Metrics            bool      `json:"metrics,omitempty"`
	CreatedAt          time.Time `json:"createdAt,omitempty"`
	UpdatedAt          time.Time `json:"updatedAt,omitempty"`
	UserId             string
	Operators          []Operator `json:"operators,omitempty"`
}

type UpstreamConfig struct {
	Enabled bool
}

type DownstreamConfig struct {
	Enabled    bool
	InstanceID string
	ServiceID  string
}

type Operator struct {
	Id               string            `json:"id,omitempty"`
	Name             string            `json:"name,omitempty"`
	ApplicationId    uuid.UUID         `json:"applicationId,omitempty"`
	ImageId          string            `json:"imageId,omitempty"`
	DeploymentType   string            `json:"deploymentType,omitempty"`
	OperatorId       string            `json:"operatorId,omitempty"`
	Config           map[string]string `json:"config,omitempty"`
	OutputTopic      string            `json:"outputTopic,omitempty"`
	InputTopics      []InputTopic      `json:"inputTopics,omitempty"`
	InputSelections  []InputSelection  `json:"inputSelections,omitempty"`
	PersistData      bool              `json:"persistData,omitempty"`
	Cost             uint              `json:"cost"`
	UpstreamConfig   UpstreamConfig    `json:"upstream,omitempty"`
	DownstreamConfig DownstreamConfig  `json:"downstream,omitempty"`
}

type InputTopic struct {
	Name        string    `json:"name,omitempty"`
	FilterType  string    `json:"filterType,omitempty"`
	FilterValue string    `json:"filterValue,omitempty"`
	Mappings    []Mapping `json:"mappings,omitempty"`
}

type Mapping struct {
	Dest   string `json:"dest,omitempty"`
	Source string `json:"source,omitempty"`
}

type InputSelection struct {
	InputName         string   `json:"inputName,omitempty"` // references mapping name
	AspectId          string   `json:"aspectId,omitempty"`
	FunctionId        string   `json:"functionId,omitempty"`
	CharacteristicIds []string `json:"characteristicIds,omitempty"`
	SelectableId      string   `json:"selectableId,omitempty"` // either device or group. can be used for SNRGY-1172, needed to update devices in group
}
